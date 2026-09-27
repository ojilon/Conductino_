package tools

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"Conductino/backend/models"
)

// Workspace keyword search. Go-native (no grep/rg on Windows targets),
// always inside the Resolve jail. Plan 07 gives hits page numbers
// (`path:page`) from the extract page map; until then, hits cite paths.

// Search caps: bounded work on a low-spec machine, bounded prompt text.
const (
	searchMaxFiles   = 60
	searchMaxMatches = 12
	searchSnippetRad = 80
	searchMaxOut     = 4000
)

// searchWorkspace is keyword search over readable workspace files. Every
// path goes through Resolve (root jail); only read_source-readable
// extensions are scanned; results are quoted snippets, never full dumps.
func (h *ToolHost) searchWorkspace(query string) ToolResult {
	query = strings.TrimSpace(query)
	if query == "" {
		return ToolResult{Name: ToolSearchWorkspace, OK: false, Content: "query required"}
	}
	if h.FS == nil || h.Docs == nil {
		return ToolResult{Name: ToolSearchWorkspace, OK: false, Content: "filesystem not available"}
	}
	root, err := h.FS.ListRoot()
	if err != nil || root == nil {
		return ToolResult{Name: ToolSearchWorkspace, OK: false, Content: "could not list workspace"}
	}
	var paths []string
	var walk func(n *models.FileTreeNode)
	walk = func(n *models.FileTreeNode) {
		if n == nil || len(paths) >= searchMaxFiles {
			return
		}
		if n.Kind != "folder" {
			rel := n.Path
			if rel == "" {
				rel = n.Label
			}
			ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(rel), "."))
			if allowedReadExt[ext] && !filepath.IsAbs(rel) && !strings.Contains(rel, "..") {
				paths = append(paths, rel)
			}
		}
		for i := range n.Children {
			walk(&n.Children[i])
		}
	}
	walk(root)

	needle := strings.ToLower(query)
	var lines []string
	scanned := 0
	for _, rel := range paths {
		if len(lines) >= searchMaxMatches {
			break
		}
		abs, err := h.FS.Resolve(rel)
		if err != nil {
			continue // jail rejection — skip silently, don't leak the path
		}
		opened, err := h.Docs.OpenFile(abs, rel)
		if err != nil || opened.Reason != "" {
			continue
		}
		scanned++
		for _, para := range blocksToParagraphs(opened.BlocksJSON) {
			if len(lines) >= searchMaxMatches {
				break
			}
			idx := strings.Index(strings.ToLower(para), needle)
			if idx < 0 {
				continue
			}
			start := max(0, idx-searchSnippetRad)
			end := min(len(para), idx+len(query)+searchSnippetRad)
			snip := strings.TrimSpace(para[start:end])
			lines = append(lines, fmt.Sprintf("- %s — “…%s…\"", rel, snip))
		}
	}
	var b strings.Builder
	if len(lines) == 0 {
		fmt.Fprintf(&b, "Search %q — no matches (%d file(s) scanned).", query, scanned)
	} else {
		fmt.Fprintf(&b, "Search %q — %d match(es) in %d file(s) scanned:\n%s",
			query, len(lines), scanned, strings.Join(lines, "\n"))
	}
	out := b.String()
	if len(out) > searchMaxOut {
		out = out[:searchMaxOut-20] + "\n…[truncated]"
	}
	return ToolResult{Name: ToolSearchWorkspace, OK: true, Content: out}
}

// blocksToParagraphs decodes BlocksJSON to per-block plain text (one entry
// per paragraph block; headings/lists flattened the same way).
func blocksToParagraphs(blocksJSON string) []string {
	if strings.TrimSpace(blocksJSON) == "" {
		return nil
	}
	var wrap struct {
		Blocks []struct {
			Segments []struct {
				Text string `json:"text"`
			} `json:"segments"`
		} `json:"blocks"`
	}
	if err := json.Unmarshal([]byte(blocksJSON), &wrap); err != nil {
		return nil
	}
	var out []string
	for _, bl := range wrap.Blocks {
		var sb strings.Builder
		for _, s := range bl.Segments {
			sb.WriteString(s.Text)
		}
		if t := strings.TrimSpace(sb.String()); t != "" {
			out = append(out, t)
		}
	}
	return out
}
