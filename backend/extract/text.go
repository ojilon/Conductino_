package extract

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"Conductino/backend/models"
)

// textBlock mirrors the TS DocumentBlock shape just enough for paragraphs.
// Marshalled with encoding/json — never string-concatenated — so arbitrary
// file content cannot break the wire format.
type textBlock struct {
	ID       string        `json:"id"`
	Type     string        `json:"type"`
	Level    int           `json:"level,omitempty"`
	Segments []textSegment `json:"segments"`
}

type textSegment struct {
	Text string `json:"text"`
}

// marshalTextBlocks encodes blocks to the {blocks:[...]} wire shape with
// encoding/json — never string-concatenated — so arbitrary file content
// cannot break the wire format.
func marshalTextBlocks(blocks []textBlock) (string, error) {
	wire, err := json.Marshal(map[string]any{"blocks": blocks})
	if err != nil {
		return "", err
	}
	return string(wire), nil
}

const (
	maxTextBytes  = 2 << 20 // 2 MiB — refuse anything bigger, it's not prose
	maxTextBlocks = 1000    // cap block count so one file can't flood the UI
)

func openTextFile(absPath, relKey string) (models.OpenedDocument, error) {
	info, err := os.Stat(absPath)
	if err != nil {
		reason, _ := ReasonOf(err)
		// ReasonOf falls back to parse_error for unclassified os errors,
		// which is correct here; keep not_found / permission_denied exact.
		return models.OpenedDocument{}, &OpenError{Reason: reason, Detail: err.Error(), Err: err}
	}
	if info.IsDir() {
		err := fmt.Errorf("not a file: %s", absPath)
		return models.OpenedDocument{}, &OpenError{Reason: models.ReasonParseError, Detail: err.Error(), Err: err}
	}
	if info.Size() > maxTextBytes {
		// Size-gated BEFORE reading: oversized files are refused without
		// ever loading them into memory (issue 23).
		err := fmt.Errorf("text file too large (%d bytes)", info.Size())
		return models.OpenedDocument{}, &OpenError{Reason: models.ReasonTooLarge, Detail: err.Error(), Err: err}
	}
	if info.Size() == 0 {
		// Fresh empty file — blank page, not a parse error.
		return emptyDocument(absPath, "text"), nil
	}
	f, err := os.Open(absPath)
	if err != nil {
		reason, _ := ReasonOf(err)
		return models.OpenedDocument{}, &OpenError{Reason: reason, Detail: err.Error(), Err: err}
	}
	defer f.Close()
	// Bounded read: LimitReader caps RSS at maxTextBytes+1 even if the file
	// grew between Stat and Open (TOCTOU).
	raw, err := io.ReadAll(io.LimitReader(f, maxTextBytes+1))
	if err != nil {
		return models.OpenedDocument{}, &OpenError{Reason: models.ReasonParseError, Detail: err.Error(), Err: err}
	}
	if len(raw) > maxTextBytes {
		err := fmt.Errorf("text file too large (%d bytes)", len(raw))
		return models.OpenedDocument{}, &OpenError{Reason: models.ReasonTooLarge, Detail: err.Error(), Err: err}
	}
	// Blank-line separated paragraphs; single newlines stay inside a block.
	paras := strings.Split(strings.ReplaceAll(string(raw), "\r\n", "\n"), "\n\n")
	blocks := make([]textBlock, 0, len(paras))
	seen := map[string]int{} // content-address occurrence among identical siblings
	for _, p := range paras {
		if len(blocks) >= maxTextBlocks {
			break
		}
		if strings.TrimSpace(p) == "" {
			continue
		}
		// Markdown-ish headings at the start of a paragraph (# / ## / ###).
		line := strings.TrimSpace(p)
		typ, level, body := "paragraph", 0, p
		if strings.HasPrefix(line, "### ") {
			typ, level, body = "heading", 3, strings.TrimPrefix(line, "### ")
		} else if strings.HasPrefix(line, "## ") {
			typ, level, body = "heading", 2, strings.TrimPrefix(line, "## ")
		} else if strings.HasPrefix(line, "# ") {
			typ, level, body = "heading", 1, strings.TrimPrefix(line, "# ")
		}
		occKey := typ + "\x00" + normalizeBlockText(body)
		occ := seen[occKey]
		seen[occKey] = occ + 1
		blk := textBlock{
			ID:       stableBlockID(relKey, typ, level, body, occ),
			Type:     typ,
			Segments: []textSegment{{Text: body}},
		}
		if level > 0 {
			blk.Level = level
		}
		blocks = append(blocks, blk)
	}
	if len(blocks) == 0 {
		blocks = append(blocks, textBlock{ID: stableBlockID(relKey, "paragraph", 0, "", 0), Type: "paragraph", Segments: []textSegment{{Text: ""}}})
	}
	wire, err := marshalTextBlocks(blocks)
	if err != nil {
		return models.OpenedDocument{}, err
	}
	base := filepath.Base(absPath)
	return models.OpenedDocument{
		Title:      strings.TrimSuffix(base, filepath.Ext(base)),
		BlocksJSON: wire,
		PageCount:  1,
		Kind:       "text",
	}, nil
}
