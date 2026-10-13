package read

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"strings"

	"Conductino/backend/model"
)

// DOCX read arm: stdlib ZIP+OOXML, zero extra deps, pure Go.
// Fidelity is paragraph/run text + bold/italic + headings + flat lists +
// tables-as-"a | b" rows. Dropped: list levels, table grid, images,
// numbering/footnotes. (The writer lives in backend/save.)

const maxDocxBytes = 20 << 20 // 20 MiB

// OpenDocx extracts one .docx's word/document.xml into blocks.
func OpenDocx(absPath, relKey string) (model.Doc, *model.Fail) {
	info, err := os.Stat(absPath)
	if err != nil {
		return model.Doc{}, model.FailOf(err)
	}
	if info.IsDir() {
		return model.Doc{}, &model.Fail{Reason: model.ReasonParseError, Detail: fmt.Sprintf("not a file: %s", absPath)}
	}
	if info.Size() > maxDocxBytes {
		return model.Doc{}, &model.Fail{Reason: model.ReasonTooLarge, Detail: fmt.Sprintf("docx too large (%d bytes)", info.Size())}
	}
	if info.Size() == 0 {
		return emptyDoc(absPath, "docx"), nil
	}

	zr, err := zip.OpenReader(absPath)
	if err != nil {
		return model.Doc{}, &model.Fail{Reason: model.ReasonParseError, Detail: "not a valid zip/docx: " + err.Error()}
	}
	defer zr.Close()

	var docXML []byte
	for _, f := range zr.File {
		if f.Name == "word/document.xml" {
			rc, err := f.Open()
			if err != nil {
				return model.Doc{}, &model.Fail{Reason: model.ReasonParseError, Detail: err.Error()}
			}
			docXML, err = io.ReadAll(io.LimitReader(rc, maxDocxBytes))
			rc.Close()
			if err != nil {
				return model.Doc{}, &model.Fail{Reason: model.ReasonParseError, Detail: err.Error()}
			}
			break
		}
	}
	if docXML == nil {
		return model.Doc{}, &model.Fail{Reason: model.ReasonParseError, Detail: "word/document.xml missing"}
	}

	blocks, err := parseDocumentXML(docXML, relKey)
	if err != nil {
		return model.Doc{}, &model.Fail{Reason: model.ReasonParseError, Detail: err.Error()}
	}
	return model.Doc{
		Title:      titleOf(absPath),
		BlocksJSON: model.MarshalBlocks(blocks),
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
	Tables     []wTable     `xml:"tbl"`
}
type wTable struct {
	Rows []wRow `xml:"tr"`
}
type wRow struct {
	Cells []wCell `xml:"tc"`
}
type wCell struct {
	Paragraphs []wParagraph `xml:"p"`
}
type wParagraph struct {
	Props *wPPr  `xml:"pPr"`
	Runs  []wRun `xml:"r"`
}
type wPPr struct {
	Style *wStyle `xml:"pStyle"`
	// Num marks a numbered/bulleted list paragraph (w:numPr). Presence is
	// the signal; level/numbering defs are ignored (flat lists).
	Num *struct{} `xml:"numPr"`
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

func parseDocumentXML(data []byte, relKey string) ([]model.Block, error) {
	cleaned := stripXMLNamespaces(data)
	dec := xml.NewDecoder(bytes.NewReader(cleaned))
	blocks := []model.Block{}
	seen := map[string]int{}
	take := func(typ string, level int, text string) string {
		occKey := typ + "\x00" + norm(text)
		occ := seen[occKey]
		seen[occKey] = occ + 1
		return stableID(relKey, typ, level, text, occ)
	}
	// Consecutive list paragraphs group into one list block (flat).
	var pendingList [][]model.Segment
	var pendingText strings.Builder
	flushList := func() {
		if len(pendingList) == 0 {
			return
		}
		if len(blocks) >= MaxTextBlocks {
			pendingList = nil
			pendingText.Reset()
			return
		}
		text := pendingText.String()
		blocks = append(blocks, model.Block{
			// Segments stays an empty array (never null): nil marshals to
			// JSON null and crashes frontend .map() calls. The list text
			// lives in ListItems.
			ID: take("list", 0, text), Type: "list",
			Segments: []model.Segment{}, ListItems: pendingList,
		})
		pendingList = nil
		pendingText.Reset()
	}
	appendPara := func(typ string, level int, segs []model.Segment) {
		if len(blocks) >= MaxTextBlocks {
			return
		}
		var full strings.Builder
		for _, s := range segs {
			full.WriteString(s.Text)
		}
		b := model.Block{ID: take(typ, level, full.String()), Type: typ, Segments: segs}
		if level > 0 {
			b.Level = level
		}
		blocks = append(blocks, b)
	}
	// Ordered token walk: preserves document order across paragraphs and
	// tables (struct unmarshalling would group all <p> before all <tbl>).
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("xml: %w", err)
		}
		se, ok := tok.(xml.StartElement)
		if !ok || (se.Name.Local != "p" && se.Name.Local != "tbl") {
			continue
		}
		// Only body-level elements: DecodeElement consumes the subtree, so
		// table-cell paragraphs never reach this loop individually.
		if se.Name.Local == "p" {
			var p wParagraph
			if err := dec.DecodeElement(&p, &se); err != nil {
				return nil, fmt.Errorf("xml: %w", err)
			}
			typ, level, segs, isList := paraViewOf(p)
			if len(segs) == 0 {
				continue
			}
			if isList {
				var full strings.Builder
				for _, s := range segs {
					full.WriteString(s.Text)
				}
				pendingList = append(pendingList, segs)
				pendingText.WriteString(full.String() + "\n")
				continue
			}
			flushList()
			appendPara(typ, level, segs)
			continue
		}
		var t wTable
		if err := dec.DecodeElement(&t, &se); err != nil {
			return nil, fmt.Errorf("xml: %w", err)
		}
		flushList()
		// Tables have no block-model counterpart: one paragraph per row,
		// cells joined with " | " (text preserved, grid dropped).
		for _, row := range t.Rows {
			var cells []string
			for _, cell := range row.Cells {
				var cp strings.Builder
				for _, p := range cell.Paragraphs {
					_, _, segs, _ := paraViewOf(p)
					for _, s := range segs {
						cp.WriteString(s.Text)
					}
					cp.WriteString(" ")
				}
				if c := strings.TrimSpace(cp.String()); c != "" {
					cells = append(cells, c)
				}
			}
			if len(cells) == 0 {
				continue
			}
			joined := strings.Join(cells, " | ")
			appendPara("paragraph", 0, []model.Segment{{Text: joined}})
		}
	}
	flushList()
	if len(blocks) == 0 {
		blocks = append(blocks, model.Block{ID: take("paragraph", 0, ""), Type: "paragraph", Segments: []model.Segment{{Text: ""}}})
	}
	return blocks, nil
}

// paraViewOf classifies one paragraph: heading level, run segments with
// marks, and whether it is a list item (w:numPr present).
func paraViewOf(p wParagraph) (typ string, level int, segs []model.Segment, isList bool) {
	segs = make([]model.Segment, 0, len(p.Runs))
	for _, r := range p.Runs {
		var text strings.Builder
		for _, t := range r.Texts {
			text.WriteString(t.Value)
		}
		s := text.String()
		if s == "" {
			continue
		}
		seg := model.Segment{Text: s}
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
	typ = "paragraph"
	if p.Props != nil {
		if p.Props.Num != nil {
			isList = true
		}
		if p.Props.Style != nil {
			style := strings.ToLower(p.Props.Style.Val)
			switch {
			case strings.Contains(style, "heading1") || style == "title":
				typ, level = "heading", 1
			case strings.Contains(style, "heading2"):
				typ, level = "heading", 2
			case strings.Contains(style, "heading3"):
				typ, level = "heading", 3
			case strings.HasPrefix(style, "heading"):
				typ, level = "heading", 2
			}
		}
	}
	return typ, level, segs, isList
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
