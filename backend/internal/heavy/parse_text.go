package heavy

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"path/filepath"
	"strings"
)

// Limits mirror backend/services/documents.go so both paths agree.
const (
	MaxTextBytes  = 2 << 20 // 2 MiB
	MaxTextBlocks = 1000
)

type textBlock struct {
	ID       string        `json:"id"`
	Type     string        `json:"type"`
	Level    int           `json:"level,omitempty"`
	Segments []textSegment `json:"segments"`
}

type textSegment struct {
	Text string `json:"text"`
}

// ParseText is pure bytes -> blocks (no IO, no network). relKey seeds stable
// IDs (same FNV-1a scheme as services.stableBlockID — duplicated here on
// purpose so heavy never imports services). Returns reason != "" on
// classified failure (too_large only; callers map IO errors themselves).
func ParseText(relKey string, raw []byte) (blocksJSON, title, kind, reason, detail string) {
	if len(raw) > MaxTextBytes {
		return "", "", "", "too_large", fmt.Sprintf("text file too large (%d bytes)", len(raw))
	}
	if relKey == "" {
		relKey = "untitled"
	}
	paras := strings.Split(strings.ReplaceAll(string(raw), "\r\n", "\n"), "\n\n")
	blocks := make([]textBlock, 0, len(paras))
	seen := map[string]int{}
	for _, p := range paras {
		if len(blocks) >= MaxTextBlocks {
			break
		}
		if strings.TrimSpace(p) == "" {
			continue
		}
		line := strings.TrimSpace(p)
		typ, level, body := "paragraph", 0, p
		if strings.HasPrefix(line, "### ") {
			typ, level, body = "heading", 3, strings.TrimPrefix(line, "### ")
		} else if strings.HasPrefix(line, "## ") {
			typ, level, body = "heading", 2, strings.TrimPrefix(line, "## ")
		} else if strings.HasPrefix(line, "# ") {
			typ, level, body = "heading", 1, strings.TrimPrefix(line, "# ")
		}
		occKey := typ + "\x00" + normalize(body)
		occ := seen[occKey]
		seen[occKey] = occ + 1
		blk := textBlock{
			ID:       stableID(relKey, typ, level, body, occ),
			Type:     typ,
			Segments: []textSegment{{Text: body}},
		}
		if level > 0 {
			blk.Level = level
		}
		blocks = append(blocks, blk)
	}
	if len(blocks) == 0 {
		blocks = append(blocks, textBlock{ID: stableID(relKey, "paragraph", 0, "", 0), Type: "paragraph", Segments: []textSegment{{Text: ""}}})
	}
	wire, err := json.Marshal(map[string]any{"blocks": blocks})
	if err != nil {
		return "", "", "", "parse_error", err.Error()
	}
	base := filepath.Base(relKey)
	title = strings.TrimSuffix(base, filepath.Ext(base))
	if title == "" {
		title = "untitled"
	}
	return string(wire), title, "text", "", ""
}

func stableID(relKey, typ string, level int, text string, occurrence int) string {
	h := fnv.New64a()
	fmt.Fprintf(h, "%s\x00%s\x00%d\x00%d\x00%s",
		strings.ToLower(strings.TrimSpace(relKey)),
		typ, level, occurrence,
		normalize(text),
	)
	return fmt.Sprintf("b-%012x", h.Sum64()&0xffffffffffff)
}

func normalize(text string) string {
	return strings.Join(strings.Fields(text), " ")
}
