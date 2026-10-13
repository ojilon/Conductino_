package read

import (
	"fmt"
	"hash/fnv"
	"io"
	"os"
	"path/filepath"
	"strings"

	"Conductino/backend/model"
)

const (
	MaxTextBytes  = 2 << 20 // 2 MiB — refuse anything bigger, it's not prose
	MaxTextBlocks = 1000    // cap blocks so one file can't flood the UI
)

// ParseText is pure bytes -> wire JSON. No disk, no network: easy to test.
// Blank-line separated paragraphs; single newlines stay inside a block;
// `# `/`## `/`### ` leaders become headings.
func ParseText(relKey string, data []byte) (blocksJSON, title string) {
	if relKey == "" {
		relKey = "untitled"
	}
	paras := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n\n")
	blocks := make([]model.Block, 0, len(paras))
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
		blk := model.Block{ID: stableID(relKey, typ, level, body, occ), Type: typ, Segments: []model.Segment{{Text: body}}}
		if level > 0 {
			blk.Level = level
		}
		blocks = append(blocks, blk)
	}
	if len(blocks) == 0 {
		blocks = append(blocks, model.Block{ID: stableID(relKey, "paragraph", 0, "", 0), Type: "paragraph", Segments: []model.Segment{{Text: ""}}})
	}
	base := filepath.Base(relKey)
	title = strings.TrimSuffix(base, filepath.Ext(base))
	if title == "" {
		title = "untitled"
	}
	return model.MarshalBlocks(blocks), title
}

// OpenText reads one txt/md file. Size-gated BEFORE reading, bounded while
// reading, so a huge file can never eat your RAM.
func OpenText(absPath, relKey string) (model.Doc, *model.Fail) {
	info, err := os.Stat(absPath)
	if err != nil {
		return model.Doc{}, model.FailOf(err)
	}
	if info.IsDir() {
		return model.Doc{}, &model.Fail{Reason: model.ReasonParseError, Detail: fmt.Sprintf("not a file: %s", absPath)}
	}
	if info.Size() > MaxTextBytes {
		return model.Doc{}, &model.Fail{Reason: model.ReasonTooLarge, Detail: fmt.Sprintf("text file too large (%d bytes)", info.Size())}
	}
	if info.Size() == 0 {
		return emptyDoc(absPath, "text"), nil
	}
	f, err := os.Open(absPath)
	if err != nil {
		return model.Doc{}, model.FailOf(err)
	}
	defer f.Close() // "close when I leave, no matter how"
	data, err := io.ReadAll(io.LimitReader(f, MaxTextBytes+1))
	if err != nil {
		return model.Doc{}, &model.Fail{Reason: model.ReasonParseError, Detail: err.Error()}
	}
	if len(data) > MaxTextBytes {
		return model.Doc{}, &model.Fail{Reason: model.ReasonTooLarge, Detail: fmt.Sprintf("text file too large (%d bytes)", len(data))}
	}
	wire, title := ParseText(relKey, data)
	return model.Doc{Title: title, Kind: "text", BlocksJSON: wire, PageCount: 1}, nil
}

// emptyDoc returns a blank openable document for 0-byte files. A fresh file
// opens as an empty page the user/AI can fill — never a parse_error.
func emptyDoc(absPath, kind string) model.Doc {
	base := filepath.Base(absPath)
	title := strings.TrimSuffix(base, filepath.Ext(base))
	if title == "" {
		title = "untitled"
	}
	return model.Doc{
		Title:      title,
		BlocksJSON: model.MarshalBlocks([]model.Block{{ID: "b-empty", Type: "paragraph", Segments: []model.Segment{{Text: ""}}}}),
		PageCount:  1,
		Kind:       kind,
	}
}

func titleOf(absPath string) string {
	t := strings.TrimSuffix(filepath.Base(absPath), filepath.Ext(absPath))
	if t == "" {
		return "untitled"
	}
	return t
}

// stableID is content-addressed: FNV-1a over relKey + type + level +
// normalized text + occurrence among identical siblings. An unchanged file
// re-extracts to identical IDs, so highlights and chat anchors survive
// reopening; edited blocks get new IDs (correct — the old anchor no longer
// describes them).
func stableID(relKey, typ string, level int, text string, occurrence int) string {
	h := fnv.New64a()
	fmt.Fprintf(h, "%s\x00%s\x00%d\x00%d\x00%s",
		strings.ToLower(strings.TrimSpace(relKey)), typ, level, occurrence, norm(text))
	return fmt.Sprintf("b-%012x", h.Sum64()&0xffffffffffff)
}

// norm collapses whitespace so "Same." and "Same.\n" hash identically.
func norm(text string) string { return strings.Join(strings.Fields(text), " ") }
