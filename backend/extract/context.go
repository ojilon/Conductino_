package extract

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Context-pack assembly (plan 11 §4: UI owns rendering + intent; budgeting
// moves behind an API). Faithful port of frontend/src/state/contextPack.ts:
// same layers, same section headers, same budgets — so swapping the TS
// builder for App.BuildContextPack changes nothing the model sees.
//
// Layers (drop lowest first if over budget):
//  1. Selection core (+ range offsets when single-block)
//  2. Local window (blocks before/after the selection block)
//  3. Document outline (headings)

// PackOpts tunes BuildContextPack. Zero values take TS defaults
// (window 2, 6000-char budget); IncludeOutline has no default — callers
// pass it explicitly (the TS helper always does: includeOutline !== false).
type PackOpts struct {
	Title          string
	BlockID        string
	SelText        string
	RangeStart     int // -1 = no range
	RangeEnd       int
	WindowBlocks   int
	IncludeOutline bool
	Budget         int
}

func (o *PackOpts) defaults() {
	if o.WindowBlocks <= 0 {
		o.WindowBlocks = 2
	}
	if o.Budget <= 0 {
		o.Budget = 6000
	}
}

// BuildContextPack assembles a budgeted context string from wire blocks.
func BuildContextPack(blocksJSON string, opts PackOpts) (string, error) {
	opts.defaults()
	var envelope struct {
		Blocks []wireBlock `json:"blocks"`
	}
	if err := json.Unmarshal([]byte(blocksJSON), &envelope); err != nil {
		return "", err
	}
	var parts []string
	if sel := strings.TrimSpace(opts.SelText); sel != "" {
		at := ""
		if opts.RangeStart >= 0 && opts.RangeEnd >= 0 && opts.BlockID != "" {
			at = fmt.Sprintf(" (block %s, chars %d–%d)", opts.BlockID, opts.RangeStart, opts.RangeEnd)
		}
		parts = append(parts, "### Selection"+at+"\n"+sel)
	}
	if title := strings.TrimSpace(opts.Title); title != "" {
		parts = append(parts, "### Document\n"+title)
	}
	if opts.BlockID != "" {
		before, after := packWindow(envelope.Blocks, opts.BlockID, opts.WindowBlocks)
		if before != "" {
			parts = append(parts, "### Before selection\n"+before)
		}
		if after != "" {
			parts = append(parts, "### After selection\n"+after)
		}
	}
	if opts.IncludeOutline {
		if outline := packOutline(envelope.Blocks, 24); outline != "" {
			parts = append(parts, "### Outline\n"+outline)
		}
	}
	pack := strings.Join(parts, "\n\n")
	if len(pack) > opts.Budget {
		pack = pack[:opts.Budget-20] + "\n\n[…truncated…]"
	}
	return pack, nil
}

func packBlockText(b wireBlock) string {
	if b.Type == "list" && len(b.ListItems) > 0 {
		var items []string
		for _, it := range b.ListItems {
			var sb strings.Builder
			for _, s := range it {
				sb.WriteString(s.Text)
			}
			items = append(items, "• "+sb.String())
		}
		return strings.Join(items, "\n")
	}
	return b.text()
}

// packWindow returns trimmed before/after text around a block (exclusive).
func packWindow(blocks []wireBlock, blockID string, radius int) (before, after string) {
	idx := -1
	for i, b := range blocks {
		if b.ID == blockID {
			idx = i
			break
		}
	}
	if idx < 0 {
		return "", ""
	}
	lo := idx - radius
	if lo < 0 {
		lo = 0
	}
	hi := idx + 1 + radius
	if hi > len(blocks) {
		hi = len(blocks)
	}
	join := func(bs []wireBlock) string {
		var out []string
		for _, b := range bs {
			if t := strings.TrimSpace(packBlockText(b)); t != "" {
				out = append(out, t)
			}
		}
		return strings.Join(out, "\n\n")
	}
	return join(blocks[lo:idx]), join(blocks[idx+1 : hi])
}

// packOutline renders heading lines (level + 120-char title, max 24).
func packOutline(blocks []wireBlock, maxHeadings int) string {
	var lines []string
	for _, b := range blocks {
		if b.Type != "heading" {
			continue
		}
		level := b.Level
		if level < 1 {
			level = 1
		}
		if level > 3 {
			level = 3
		}
		title := strings.TrimSpace(packBlockText(b))
		if len(title) > 120 {
			title = title[:120]
		}
		if title == "" {
			continue
		}
		lines = append(lines, strings.Repeat("#", level)+" "+title)
		if len(lines) >= maxHeadings {
			break
		}
	}
	return strings.Join(lines, "\n")
}
