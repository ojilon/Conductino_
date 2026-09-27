package extract

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Normalized intermediate (plan 07) — one canonical text per file version.
//
// Every successfully extracted document can also materialize:
//   mdText    — Markdown-ish view (headings as `#`, lists as `- `,
//               tables as `a | b`, pages as `--- page N ---` markers)
//   pageMap   — JSON: page → first blockId (+ char offset)
//
// Both AI tools and the summary mirrors read this intermediate — never two
// divergent renderings. Pure functions; no I/O.

// wireBlock is the generic shape of one wire block (all arms marshal it).
type wireBlock struct {
	ID        string          `json:"id"`
	Type      string          `json:"type"`
	Level     int             `json:"level,omitempty"`
	Segments  []wireSegment   `json:"segments"`
	ListItems [][]wireSegment `json:"listItems,omitempty"`
}

type wireSegment struct {
	Text string `json:"text"`
}

func (b wireBlock) text() string {
	var sb strings.Builder
	for _, s := range b.Segments {
		sb.WriteString(s.Text)
	}
	return sb.String()
}

// NormalizeBlocksJSON converts wire blocks ({blocks:[...]}) to the canonical
// intermediate: md text plus a page map. `page`-type blocks are the source
// of truth for pagination; formats without them form one page.
func NormalizeBlocksJSON(blocksJSON string) (mdText, pageMapJSON string, err error) {
	var envelope struct {
		Blocks []wireBlock `json:"blocks"`
	}
	if err := json.Unmarshal([]byte(blocksJSON), &envelope); err != nil {
		return "", "", err
	}
	var md strings.Builder
	var pages []PageMapEntry
	page := 1
	pageHasBlock := false
	markPage := func() {
		pages = append(pages, PageMapEntry{Page: page, BlockID: "", Offset: md.Len()})
	}
	markPage()
	emit := func(blockID, text string) {
		if md.Len() > 0 {
			md.WriteString("\n\n")
		}
		if !pageHasBlock {
			pages[len(pages)-1].BlockID = blockID
			pageHasBlock = true
		}
		md.WriteString(text)
	}
	for _, b := range envelope.Blocks {
		switch b.Type {
		case "page":
			page++
			pageHasBlock = false
			if md.Len() > 0 {
				md.WriteString("\n\n")
			}
			md.WriteString(fmt.Sprintf("--- page %d ---", page))
			markPage()
		case "heading":
			level := b.Level
			if level < 1 {
				level = 1
			}
			if level > 3 {
				level = 3
			}
			if t := strings.TrimSpace(b.text()); t != "" {
				emit(b.ID, strings.Repeat("#", level)+" "+t)
			}
		case "list":
			var items []string
			for _, it := range b.ListItems {
				var sb strings.Builder
				for _, s := range it {
					sb.WriteString(s.Text)
				}
				if t := strings.TrimSpace(sb.String()); t != "" {
					items = append(items, "- "+t)
				}
			}
			if len(items) > 0 {
				emit(b.ID, strings.Join(items, "\n"))
			}
		default:
			if t := strings.TrimSpace(b.text()); t != "" {
				emit(b.ID, t)
			}
		}
	}
	pm, err := json.Marshal(pages)
	if err != nil {
		return "", "", err
	}
	return md.String(), string(pm), nil
}

// PlainToMarkdown normalizes free text (frontend snapshots, chat-authored
// content) into the same mirror shape: `#` lines stay headings, blank-line
// separated runs become paragraphs.
func PlainToMarkdown(text string) string {
	var out []string
	for _, para := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n\n") {
		para = strings.TrimSpace(para)
		if para == "" {
			continue
		}
		out = append(out, para)
	}
	return strings.Join(out, "\n\n")
}

// BlocksFromMarkdown converts mirror markdown back to wire blocks for the
// publish path (mirror → .docx). `#` lines become headings (level by count,
// max 3), `- `/`• ` runs group into one list block, `--- page N ---`
// markers are skipped (pagination is a view concern), and everything else
// becomes paragraphs. IDs are stable content-addressed values (same scheme
// as extraction) so highlights and anchors survive a publish round-trip.
func BlocksFromMarkdown(md, relKey string) ([]DocxBlock, error) {
	var blocks []DocxBlock
	seen := map[string]int{}
	take := func(typ string, level int, text string) DocxBlock {
		occKey := typ + "\x00" + normalizeBlockText(text)
		occ := seen[occKey]
		seen[occKey] = occ + 1
		return DocxBlock{ID: stableBlockID(relKey, typ, level, text, occ), Type: typ, Level: level}
	}
	flushList := func(items []string) {
		if len(items) == 0 {
			return
		}
		blk := take("list", 0, strings.Join(items, "\n"))
		for _, it := range items {
			blk.ListItems = append(blk.ListItems, []DocxSegment{{Text: it}})
		}
		if len(blk.Segments) == 0 {
			blk.Segments = []DocxSegment{{Text: items[0]}}
		}
		blocks = append(blocks, blk)
	}
	var pending []string
	for _, para := range strings.Split(strings.ReplaceAll(md, "\r\n", "\n"), "\n\n") {
		for _, line := range strings.Split(para, "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			if strings.HasPrefix(line, "--- page") {
				continue
			}
			if item, ok := strings.CutPrefix(line, "- "); ok {
				pending = append(pending, strings.TrimSpace(item))
				continue
			}
			if item, ok := strings.CutPrefix(line, "• "); ok {
				pending = append(pending, strings.TrimSpace(item))
				continue
			}
			flushList(pending)
			pending = nil
			if strings.HasPrefix(line, "#") {
				level := 0
				for level < len(line) && line[level] == '#' {
					level++
				}
				if level > 3 {
					level = 3
				}
				if body := strings.TrimSpace(line[level:]); body != "" {
					blk := take("heading", level, body)
					blk.Level = level
					blk.Segments = []DocxSegment{{Text: body}}
					blocks = append(blocks, blk)
					continue
				}
			}
			blk := take("paragraph", 0, line)
			blk.Segments = []DocxSegment{{Text: line}}
			blocks = append(blocks, blk)
		}
		flushList(pending)
		pending = nil
	}
	if len(blocks) == 0 {
		blk := take("paragraph", 0, "")
		blk.Segments = []DocxSegment{{Text: ""}}
		blocks = append(blocks, blk)
	}
	return blocks, nil
}

// PageMapEntry is one row of the page/block map (JSON-serializable).
type PageMapEntry struct {
	Page    int    `json:"page"`
	BlockID string `json:"blockId"`
	Offset  int    `json:"offset"`
}
