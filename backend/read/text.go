package read

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const (
	MaxTextBytes  = 2 << 20 // 2 MiB — refuse anything bigger, it's not prose
	MaxTextBlocks = 1000    // cap blocks so one file can't flood the UI
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

// ParseText is pure bytes -> wire JSON. No disk, no network: easy to test.
func ParseText(relKey string, data []byte) (blocksJSON, title string) {
	if relKey == "" {
		relKey = "untitled"
	}
	paras := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n\n")
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
		key := typ + "\x00" + norm(body)
		occ := seen[key]
		seen[key] = occ + 1
		blk := textBlock{ID: stableID(relKey, typ, level, body, occ), Type: typ, Segments: []textSegment{{Text: body}}}
		if level > 0 {
			blk.Level = level
		}
		blocks = append(blocks, blk)
	}
	if len(blocks) == 0 {
		blocks = append(blocks, textBlock{ID: stableID(relKey, "paragraph", 0, "", 0), Type: "paragraph", Segments: []textSegment{{Text: ""}}})
	}
	wire, _ := json.Marshal(map[string]any{"blocks": blocks})
	base := filepath.Base(relKey)
	title = strings.TrimSuffix(base, filepath.Ext(base))
	if title == "" {
		title = "untitled"
	}
	return string(wire), title
}

// OpenText reads one txt/md file. Size-gated BEFORE reading, bounded while
// reading, so a huge file can never eat your RAM.
func OpenText(absPath, relKey string) (Doc, *Fail) {
	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return Doc{}, &Fail{ReasonNotFound, err.Error()}
		}
		if os.IsPermission(err) {
			return Doc{}, &Fail{ReasonPermissionDenied, err.Error()}
		}
		return Doc{}, &Fail{ReasonParseError, err.Error()}
	}
	if info.IsDir() {
		return Doc{}, &Fail{ReasonParseError, fmt.Sprintf("not a file: %s", absPath)}
	}
	if info.Size() > MaxTextBytes {
		return Doc{}, &Fail{ReasonTooLarge, fmt.Sprintf("text file too large (%d bytes)", info.Size())}
	}
	if info.Size() == 0 {
		wire, _ := json.Marshal(map[string]any{"blocks": []textBlock{{ID: "b-empty", Type: "paragraph", Segments: []textSegment{{Text: ""}}}}})
		return Doc{Title: titleOf(absPath), Kind: "text", Blocks: string(wire), Pages: 1}, nil
	}
	f, err := os.Open(absPath)
	if err != nil {
		if os.IsPermission(err) {
			return Doc{}, &Fail{ReasonPermissionDenied, err.Error()}
		}
		return Doc{}, &Fail{ReasonParseError, err.Error()}
	}
	defer f.Close() // "close when I leave, no matter how" — your first Go idiom
	data, err := io.ReadAll(io.LimitReader(f, MaxTextBytes+1))
	if err != nil {
		return Doc{}, &Fail{ReasonParseError, err.Error()}
	}
	if len(data) > MaxTextBytes {
		return Doc{}, &Fail{ReasonTooLarge, fmt.Sprintf("text file too large (%d bytes)", len(data))}
	}
	wire, title := ParseText(relKey, data)
	return Doc{Title: title, Kind: "text", Blocks: wire, Pages: 1}, nil
}

func titleOf(absPath string) string {
	t := strings.TrimSuffix(filepath.Base(absPath), filepath.Ext(absPath))
	if t == "" {
		return "untitled"
	}
	return t
}

// stableID + norm from backend/extract/ids.go: ID from WHAT it says, not
// WHERE it sits — reopening an unchanged file yields identical IDs.
func stableID(relKey, typ string, level int, text string, occurrence int) string {
	h := fnv.New64a()
	fmt.Fprintf(h, "%s\x00%s\x00%d\x00%d\x00%s",
		strings.ToLower(strings.TrimSpace(relKey)), typ, level, occurrence, norm(text))
	return fmt.Sprintf("b-%012x", h.Sum64()&0xffffffffffff)
}

func norm(text string) string { return strings.Join(strings.Fields(text), " ") }
