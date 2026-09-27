package extract

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"Conductino/backend/models"
	pdf "github.com/ledongthuc/pdf"
)

// PDF text-layer extraction (pure Go, no cgo — see tasks.md §4).
// Fidelity is text rows in content-stream order with paragraph recovery by
// vertical gap; layout, tables, and images are dropped (documented stopgap).
// Scanned/image-only PDFs have no text layer and extract as empty pages.

const (
	maxPdfBytes = 50 << 20 // 50 MiB on disk — Stat-gated before opening
	maxPdfPages = 500      // hard page cap so one file can't flood the UI
	maxPageChars = 20_000  // per-page text cap (truncation marker appended)
)

func openPdfFile(absPath, relKey string) (models.OpenedDocument, error) {
	info, err := os.Stat(absPath)
	if err != nil {
		reason, _ := ReasonOf(err)
		return models.OpenedDocument{}, &OpenError{Reason: reason, Detail: err.Error(), Err: err}
	}
	if info.IsDir() {
		err := fmt.Errorf("not a file: %s", absPath)
		return models.OpenedDocument{}, &OpenError{Reason: models.ReasonParseError, Detail: err.Error(), Err: err}
	}
	if info.Size() > maxPdfBytes {
		err := fmt.Errorf("pdf too large (%d bytes)", info.Size())
		return models.OpenedDocument{}, &OpenError{Reason: models.ReasonTooLarge, Detail: err.Error(), Err: err}
	}
	if info.Size() == 0 {
		// Fresh empty file — blank page, not a parse error.
		return emptyDocument(absPath, "pdf"), nil
	}
	f, r, err := pdf.Open(absPath)
	if err != nil {
		return models.OpenedDocument{}, &OpenError{Reason: models.ReasonParseError, Detail: "not a readable pdf: " + err.Error(), Err: err}
	}
	defer f.Close()

	numPages := r.NumPage()
	if numPages > maxPdfPages {
		numPages = maxPdfPages
	}
	blocks := []textBlock{}
	seen := map[string]int{}
	// take assigns stable content-addressed IDs (issues 7+27).
	take := func(typ string, level int, text string) string {
		occKey := typ + "\x00" + normalizeBlockText(text)
		occ := seen[occKey]
		seen[occKey] = occ + 1
		return stableBlockID(relKey, typ, level, text, occ)
	}
	totalChars := 0
	for p := 1; p <= numPages && len(blocks) < maxTextBlocks; p++ {
		// Page-break anchor: the canvas leaf + chat both key pages by it.
		blocks = append(blocks, textBlock{ID: take("page", 0, fmt.Sprintf("page %d", p)), Type: "page", Segments: []textSegment{{Text: ""}}})
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
			if len(blocks) >= maxTextBlocks {
				return
			}
			blocks = append(blocks, textBlock{
				ID: take("paragraph", 0, t), Type: "paragraph",
				Segments: []textSegment{{Text: t}},
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
			if totalChars += len(t); totalChars > maxTextBytes {
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
		blocks = append(blocks, textBlock{ID: take("paragraph", 0, ""), Type: "paragraph", Segments: []textSegment{{Text: ""}}})
	}
	wire, err := marshalTextBlocks(blocks)
	if err != nil {
		return models.OpenedDocument{}, err
	}
	base := filepath.Base(absPath)
	return models.OpenedDocument{
		Title:      strings.TrimSuffix(base, filepath.Ext(base)),
		BlocksJSON: wire,
		PageCount:  numPages,
		Kind:       "pdf",
	}, nil
}
