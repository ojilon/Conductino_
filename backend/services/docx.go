package services

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"Conductino/backend/models"
)

// Phase 8: stdlib DOCX read/write (ZIP + OOXML). Zero extra deps, pure Go.
// Fidelity is paragraph/run text + bold/italic — enough for summary files
// and source reading. Tables/images/numbering are skipped (text-only stopgap
// while a fuller library is evaluated; see tasks.md §DOCX).

const maxDocxBytes = 20 << 20 // 20 MiB

// ---- shared block shapes (JSON wire matches frontend DocumentBlock) ----

type docxBlock struct {
	ID        string         `json:"id"`
	Type      string         `json:"type"`
	Level     int            `json:"level,omitempty"`
	Segments  []docxSegment  `json:"segments"`
	ListItems [][]docxSegment `json:"listItems,omitempty"`
}

type docxSegment struct {
	Text   string `json:"text"`
	Em     bool   `json:"em,omitempty"`
	Strong bool   `json:"strong,omitempty"`
}

// BlocksToJSON marshals blocks into the {blocks:[...]} wire shape.
func BlocksToJSON(blocks []docxBlock) (string, error) {
	if len(blocks) == 0 {
		blocks = []docxBlock{{ID: "empty", Type: "paragraph", Segments: []docxSegment{{Text: ""}}}}
	}
	b, err := json.Marshal(map[string]any{"blocks": blocks})
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// ParseBlocksJSON decodes the frontend/AI wire format.
func ParseBlocksJSON(s string) ([]docxBlock, error) {
	var envelope struct {
		Blocks []docxBlock `json:"blocks"`
	}
	if err := json.Unmarshal([]byte(s), &envelope); err != nil {
		return nil, err
	}
	return envelope.Blocks, nil
}

// ---- READ: openDocxFile ----

func openDocxFile(absPath string) (models.OpenedDocument, error) {
	info, err := os.Stat(absPath)
	if err != nil {
		reason, _ := ReasonOf(err)
		return models.OpenedDocument{}, &OpenError{Reason: reason, Detail: err.Error(), Err: err}
	}
	if info.Size() > maxDocxBytes {
		err := fmt.Errorf("docx too large (%d bytes)", info.Size())
		return models.OpenedDocument{}, &OpenError{Reason: models.ReasonTooLarge, Detail: err.Error(), Err: err}
	}

	zr, err := zip.OpenReader(absPath)
	if err != nil {
		return models.OpenedDocument{}, &OpenError{Reason: models.ReasonParseError, Detail: "not a valid zip/docx: " + err.Error(), Err: err}
	}
	defer zr.Close()

	var docXML []byte
	for _, f := range zr.File {
		if f.Name == "word/document.xml" {
			rc, err := f.Open()
			if err != nil {
				return models.OpenedDocument{}, &OpenError{Reason: models.ReasonParseError, Detail: err.Error(), Err: err}
			}
			docXML, err = io.ReadAll(io.LimitReader(rc, maxDocxBytes))
			rc.Close()
			if err != nil {
				return models.OpenedDocument{}, &OpenError{Reason: models.ReasonParseError, Detail: err.Error(), Err: err}
			}
			break
		}
	}
	if docXML == nil {
		return models.OpenedDocument{}, &OpenError{Reason: models.ReasonParseError, Detail: "word/document.xml missing", Err: fmt.Errorf("missing document.xml")}
	}

	blocks, err := parseDocumentXML(docXML)
	if err != nil {
		return models.OpenedDocument{}, &OpenError{Reason: models.ReasonParseError, Detail: err.Error(), Err: err}
	}
	wire, err := BlocksToJSON(blocks)
	if err != nil {
		return models.OpenedDocument{}, err
	}
	base := filepath.Base(absPath)
	return models.OpenedDocument{
		Title:      strings.TrimSuffix(base, filepath.Ext(base)),
		BlocksJSON: wire,
		PageCount:  1,
		Kind:       "docx",
	}, nil
}

// OOXML document.xml fragment types (namespace-agnostic local names).
type wDocument struct {
	Body wBody `xml:"body"`
}
type wBody struct {
	Paragraphs []wParagraph `xml:"p"`
}
type wParagraph struct {
	Props *wPPr  `xml:"pPr"`
	Runs  []wRun `xml:"r"`
}
type wPPr struct {
	Style *wStyle `xml:"pStyle"`
}
type wStyle struct {
	Val string `xml:"val,attr"`
}
type wRun struct {
	Props *wRPr   `xml:"rPr"`
	Texts []wText `xml:"t"`
}
type wRPr struct {
	Bold   *struct{} `xml:"b"`
	Italic *struct{} `xml:"i"`
}
type wText struct {
	Value string `xml:",chardata"`
}

func parseDocumentXML(data []byte) ([]docxBlock, error) {
	cleaned := stripXMLNamespaces(data)
	var doc wDocument
	if err := xml.Unmarshal(cleaned, &doc); err != nil {
		return nil, fmt.Errorf("xml: %w", err)
	}
	blocks := make([]docxBlock, 0, len(doc.Body.Paragraphs))
	for i, p := range doc.Body.Paragraphs {
		if i >= maxTextBlocks {
			break
		}
		segs := make([]docxSegment, 0, len(p.Runs))
		for _, r := range p.Runs {
			var text strings.Builder
			for _, t := range r.Texts {
				text.WriteString(t.Value)
			}
			s := text.String()
			if s == "" {
				continue
			}
			seg := docxSegment{Text: s}
			if r.Props != nil {
				if r.Props.Bold != nil {
					seg.Strong = true
				}
				if r.Props.Italic != nil {
					seg.Em = true
				}
			}
			segs = append(segs, seg)
		}
		if len(segs) == 0 {
			continue
		}
		b := docxBlock{ID: fmt.Sprintf("docx-%d", i), Type: "paragraph", Segments: segs}
		if p.Props != nil && p.Props.Style != nil {
			style := strings.ToLower(p.Props.Style.Val)
			switch {
			case strings.Contains(style, "heading1") || style == "title":
				b.Type, b.Level = "heading", 1
			case strings.Contains(style, "heading2"):
				b.Type, b.Level = "heading", 2
			case strings.Contains(style, "heading3"):
				b.Type, b.Level = "heading", 3
			case strings.HasPrefix(style, "heading"):
				b.Type, b.Level = "heading", 2
			}
		}
		blocks = append(blocks, b)
	}
	if len(blocks) == 0 {
		blocks = append(blocks, docxBlock{ID: "docx-0", Type: "paragraph", Segments: []docxSegment{{Text: ""}}})
	}
	return blocks, nil
}

func stripXMLNamespaces(data []byte) []byte {
	s := string(data)
	for {
		i := strings.Index(s, "xmlns")
		if i < 0 {
			break
		}
		j := i
		for j < len(s) && s[j] != '>' && s[j] != ' ' {
			if s[j] == '=' {
				j++
				if j < len(s) && s[j] == '"' {
					j++
					for j < len(s) && s[j] != '"' {
						j++
					}
					if j < len(s) {
						j++
					}
				}
				break
			}
			j++
		}
		s = s[:i] + s[j:]
	}
	s = strings.ReplaceAll(s, "<w:", "<")
	s = strings.ReplaceAll(s, "</w:", "</")
	s = strings.ReplaceAll(s, "<r:", "<")
	s = strings.ReplaceAll(s, "</r:", "</")
	return []byte(s)
}

// ---- WRITE: blocks → DOCX ----

// WriteDOCX writes a minimal OOXML package to absPath from canonical blocks.
func WriteDOCX(absPath string, blocks []docxBlock) error {
	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		return err
	}
	f, err := os.Create(absPath)
	if err != nil {
		return err
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	defer zw.Close()

	files := map[string]string{
		"[Content_Types].xml":          contentTypesXML,
		"_rels/.rels":                  relsXML,
		"word/_rels/document.xml.rels": documentRelsXML,
		"word/document.xml":            buildDocumentXML(blocks),
	}
	for name, body := range files {
		w, err := zw.Create(name)
		if err != nil {
			return err
		}
		if _, err := io.WriteString(w, body); err != nil {
			return err
		}
	}
	return nil
}

const contentTypesXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
  <Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
</Types>`

const relsXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>
</Relationships>`

const documentRelsXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
</Relationships>`

func buildDocumentXML(blocks []docxBlock) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`)
	b.WriteString(`<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">`)
	b.WriteString(`<w:body>`)
	for _, blk := range blocks {
		writeParagraphXML(&b, blk)
	}
	b.WriteString(`<w:sectPr><w:pgSz w:w="12240" w:h="15840"/><w:pgMar w:top="1440" w:right="1440" w:bottom="1440" w:left="1440"/></w:sectPr>`)
	b.WriteString(`</w:body></w:document>`)
	return b.String()
}

func writeParagraphXML(b *strings.Builder, blk docxBlock) {
	b.WriteString(`<w:p>`)
	if blk.Type == "heading" {
		level := blk.Level
		if level < 1 || level > 3 {
			level = 1
		}
		b.WriteString(fmt.Sprintf(`<w:pPr><w:pStyle w:val="Heading%d"/></w:pPr>`, level))
	}
	segs := blk.Segments
	if blk.Type == "list" && len(blk.ListItems) > 0 {
		var flat []docxSegment
		for i, row := range blk.ListItems {
			if i > 0 {
				flat = append(flat, docxSegment{Text: "\n"})
			}
			flat = append(flat, row...)
		}
		segs = flat
	}
	if len(segs) == 0 {
		b.WriteString(`<w:r><w:t></w:t></w:r>`)
	} else {
		for _, s := range segs {
			b.WriteString(`<w:r>`)
			if s.Strong || s.Em {
				b.WriteString(`<w:rPr>`)
				if s.Strong {
					b.WriteString(`<w:b/>`)
				}
				if s.Em {
					b.WriteString(`<w:i/>`)
				}
				b.WriteString(`</w:rPr>`)
			}
			b.WriteString(`<w:t xml:space="preserve">`)
			b.WriteString(xmlEscape(s.Text))
			b.WriteString(`</w:t></w:r>`)
		}
	}
	b.WriteString(`</w:p>`)
}

func xmlEscape(s string) string {
	var buf bytes.Buffer
	_ = xml.EscapeText(&buf, []byte(s))
	return buf.String()
}

// DefaultSummaryDOCXName returns a stable relative path for the primary summary file.
func DefaultSummaryDOCXName(title string) string {
	safe := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			return r
		case r == ' ':
			return '-'
		default:
			return -1
		}
	}, strings.TrimSpace(title))
	if safe == "" {
		safe = "summary"
	}
	return filepath.Join("summaries", safe+".docx")
}

// WriteSummaryDOCXPath resolves a workspace-relative path and writes blocks as DOCX.
func WriteSummaryDOCXPath(absRoot, relPath string, blocksJSON string) (string, error) {
	blocks, err := ParseBlocksJSON(blocksJSON)
	if err != nil {
		return "", fmt.Errorf("parse blocks: %w", err)
	}
	clean := filepath.Clean(relPath)
	if strings.HasPrefix(clean, "..") || filepath.IsAbs(clean) {
		return "", fmt.Errorf("path escapes workspace: %s", relPath)
	}
	abs := filepath.Join(absRoot, clean)
	relCheck, err := filepath.Rel(absRoot, abs)
	if err != nil || strings.HasPrefix(relCheck, "..") {
		return "", fmt.Errorf("path escapes workspace: %s", relPath)
	}
	if err := WriteDOCX(abs, blocks); err != nil {
		return "", err
	}
	return abs, nil
}
