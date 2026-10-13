package save

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"Conductino/backend/model"
)

// Minimal .docx writer (stdlib ZIP+OOXML, zero extra deps). Emits one
// paragraph per block; lists become "• " paragraphs (readable on re-read,
// not round-trip-identical); headings keep their level.

// WriteDOCX writes a minimal OOXML package to absPath from blocks.
func WriteDOCX(absPath string, blocks []model.Block) error {
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

func buildDocumentXML(blocks []model.Block) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`)
	b.WriteString(`<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">`)
	b.WriteString(`<w:body>`)
	for _, blk := range blocks {
		// Lists have no numbering.xml in this minimal package: one bullet
		// paragraph per item ("• " prefix). Re-reading yields plain
		// paragraphs — readable, not round-trip-identical.
		if blk.Type == "list" && len(blk.ListItems) > 0 {
			for _, item := range blk.ListItems {
				item := append([]model.Segment{{Text: "• "}}, item...)
				writeParagraphXML(&b, model.Block{ID: blk.ID, Type: "paragraph", Segments: item})
			}
			continue
		}
		writeParagraphXML(&b, blk)
	}
	b.WriteString(`<w:sectPr><w:pgSz w:w="12240" w:h="15840"/><w:pgMar w:top="1440" w:right="1440" w:bottom="1440" w:left="1440"/></w:sectPr>`)
	b.WriteString(`</w:body></w:document>`)
	return b.String()
}

func writeParagraphXML(b *strings.Builder, blk model.Block) {
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
		var flat []model.Segment
		for i, row := range blk.ListItems {
			if i > 0 {
				flat = append(flat, model.Segment{Text: "\n"})
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

// DefaultSummaryDOCXName returns a stable workspace-relative path for the
// primary summary file.
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

// WriteSummaryDOCXPath resolves a workspace-relative path and writes blocks
// as DOCX. Same jail as everything else: absolute paths and `..` escapes
// are rejected.
func WriteSummaryDOCXPath(absRoot, relPath string, blocksJSON string) (string, error) {
	blocks, err := model.ParseBlocks(blocksJSON)
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
