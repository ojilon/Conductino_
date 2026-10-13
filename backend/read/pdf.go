package read

import (
	"fmt"
	"os"
	"strings"

	"Conductino/backend/model"
	pdf "github.com/ledongthuc/pdf"
)

// PDF text-layer extraction (pure Go, no cgo). Fidelity is text rows in
// content-stream order with paragraph recovery by vertical gap; layout,
// tables, and images are dropped. Scanned/image-only PDFs have no text
// layer and extract as empty pages.

const (
	maxPDFBytes  = 50 << 20 // 50 MiB on disk — Stat-gated before opening
	maxPDFPages  = 500      // hard page cap so one file can't flood the UI
	maxPageChars = 20000    // per-page text cap
)

// OpenPDF extracts one PDF's text layer into paragraph blocks with a
// `page`-type break block before each page (canvas + chat anchor on it).
func OpenPDF(absPath, relKey string) (model.Doc, *model.Fail) {
	info, err := os.Stat(absPath)
	if err != nil {
		return model.Doc{}, model.FailOf(err)
	}
	if info.IsDir() {
		return model.Doc{}, &model.Fail{Reason: model.ReasonParseError, Detail: fmt.Sprintf("not a file: %s", absPath)}
	}
	if info.Size() > maxPDFBytes {
		return model.Doc{}, &model.Fail{Reason: model.ReasonTooLarge, Detail: fmt.Sprintf("pdf too large (%d bytes)", info.Size())}
	}
	if info.Size() == 0 {
		return emptyDoc(absPath, "pdf"), nil
	}
	f, r, err := pdf.Open(absPath)
	if err != nil {
		return model.Doc{}, &model.Fail{Reason: model.ReasonParseError, Detail: "not a readable pdf: " + err.Error()}
	}
	defer f.Close()

	numPages := r.NumPage()
	if numPages > maxPDFPages {
		numPages = maxPDFPages
	}
	blocks := []model.Block{}
	seen := map[string]int{}
	take := func(typ string, level int, text string) string {
		occKey := typ + "\x00" + norm(text)
		occ := seen[occKey]
		seen[occKey] = occ + 1
		return stableID(relKey, typ, level, text, occ)
	}
	totalChars := 0
	for p := 1; p <= numPages && len(blocks) < MaxTextBlocks; p++ {
		// Page-break anchor: the canvas leaf + chat both key pages by it.
		blocks = append(blocks, model.Block{ID: take("page", 0, fmt.Sprintf("page %d", p)), Type: "page", Segments: []model.Segment{{Text: ""}}})
		rows, err := r.Page(p).GetTextByRow()
		if err != nil {
			continue // unreadable page — keep the break, skip content
		}
		var cur strings.Builder
		var prevPos int64
		var prevSize float64
		flush := func() {
			t := strings.TrimSpace(cur.String())
			cur.Reset()
			if t == "" {
				return
			}
			if len(blocks) >= MaxTextBlocks {
				return
			}
			blocks = append(blocks, model.Block{
				ID: take("paragraph", 0, t), Type: "paragraph",
				Segments: []model.Segment{{Text: t}},
			})
		}
		for _, row := range rows {
			if row == nil {
				continue
			}
			var line strings.Builder
			maxSize := 0.0
			for _, w := range row.Content {
				if strings.TrimSpace(w.S) == "" {
					continue
				}
				if line.Len() > 0 {
					line.WriteString(" ")
				}
				line.WriteString(strings.TrimSpace(w.S))
				if w.FontSize > maxSize {
					maxSize = w.FontSize
				}
			}
			t := strings.TrimSpace(line.String())
			if t == "" {
				flush() // blank row = paragraph break
				prevPos, prevSize = 0, 0
				continue
			}
			// New paragraph when the vertical gap exceeds ~1.6 lines.
			// Positions may be degenerate (0) in odd files — then every
			// row stays in one paragraph rather than splitting wrongly.
			if cur.Len() > 0 && prevPos > 0 && row.Position > 0 {
				gap := prevPos - row.Position
				threshold := prevSize*1.6 + 2
				if threshold < 8 {
					threshold = 8
				}
				if gap > int64(threshold) {
					flush()
				} else {
					cur.WriteString(" ")
				}
			} else if cur.Len() > 0 {
				cur.WriteString(" ")
			}
			cur.WriteString(t)
			if totalChars += len(t); totalChars > MaxTextBytes {
				cur.WriteString(" […document truncated]")
				flush()
				break
			}
			if cur.Len() > maxPageChars {
				flush()
			}
			prevPos, prevSize = row.Position, maxSize
		}
		flush()
	}
	if len(blocks) == 0 {
		blocks = append(blocks, model.Block{ID: take("paragraph", 0, ""), Type: "paragraph", Segments: []model.Segment{{Text: ""}}})
	}
	return model.Doc{
		Title:      titleOf(absPath),
		BlocksJSON: model.MarshalBlocks(blocks),
		PageCount:  numPages,
		Kind:       "pdf",
	}, nil
}
