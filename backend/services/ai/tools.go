package ai

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"Conductino/backend/models"
)

// Tool names (stable wire identifiers).
const (
	ToolListWorkspace       = "list_workspace"
	ToolReadSource          = "read_source"
	ToolReadSummary         = "read_summary"
	ToolProposeSummaryEdit  = "propose_summary_edit"
)

// Allowed source extensions for read_source (lowercase, no dot).
var allowedReadExt = map[string]bool{
	"txt": true, "md": true, "markdown": true,
	// PDF/DOCX land later; reject until extractors exist so tools never invent content.
}

// PathResolver is the workspace containment boundary (Filesystem.Resolve + tree).
type PathResolver interface {
	Resolve(path string) (string, error)
	Root() string
	ListRoot() (*models.FileTreeNode, error)
}

// FileOpener extracts displayable content from an absolute path already
// Resolve-checked by the host.
type FileOpener interface {
	OpenFile(absPath string) (models.OpenedDocument, error)
}

// ToolHost holds per-request deps for folder-scoped tools.
type ToolHost struct {
	FS                 PathResolver
	Docs               FileOpener
	SummaryText        string // snapshot from frontend (Phase 6 → storage)
	PrimarySummaryID   string
	lastProposalText   string // set by propose_summary_edit for final payload
}

// ToolResult is what the model sees after a tool runs.
type ToolResult struct {
	Name    string
	OK      bool
	Content string
}

// Catalog returns a short system description of available tools for the prompt.
func ToolCatalog() string {
	return strings.TrimSpace(`
### Tools (optional)
You may call at most two tools before answering. Use ONLY this XML form on its own line:
<tool name="list_workspace"/>
<tool name="read_source" path="relative/path.txt"/>
<tool name="read_summary"/>
<tool name="propose_summary_edit" text="1-2 factual sentences to insert into the research summary"/>

Rules:
- Paths are relative to the open research folder. Never use absolute paths or "..".
- read_source is limited to .txt/.md. If unsupported, say so.
- propose_summary_edit drafts an insertion for the user to accept; it does not write files.
- After tool results appear, give a normal answer (no further tool tags unless needed).
`)
}

// toolCallRe matches a single tool invocation line.
var toolCallRe = regexp.MustCompile(`(?i)<tool\s+name="([a-z_]+)"([^>]*)/>`)

var toolAttrRe = regexp.MustCompile(`([a-z_]+)="([^"]*)"`)

// ParseToolCalls extracts tool invocations from model text (order preserved).
func ParseToolCalls(text string) []struct {
	Name string
	Args map[string]string
} {
	var out []struct {
		Name string
		Args map[string]string
	}
	matches := toolCallRe.FindAllStringSubmatch(text, -1)
	for _, m := range matches {
		name := strings.ToLower(strings.TrimSpace(m[1]))
		args := map[string]string{}
		for _, a := range toolAttrRe.FindAllStringSubmatch(m[2], -1) {
			k := strings.ToLower(a[1])
			if k == "name" {
				continue
			}
			args[k] = a[2]
		}
		out = append(out, struct {
			Name string
			Args map[string]string
		}{Name: name, Args: args})
	}
	return out
}

// StripToolTags removes tool XML so the final answer is clean prose.
func StripToolTags(text string) string {
	return strings.TrimSpace(toolCallRe.ReplaceAllString(text, ""))
}

// Dispatch runs one tool with Resolve + extension guards.
func (h *ToolHost) Dispatch(name string, args map[string]string) ToolResult {
	if h == nil {
		return ToolResult{Name: name, OK: false, Content: "tools unavailable"}
	}
	switch strings.ToLower(strings.TrimSpace(name)) {
	case ToolListWorkspace:
		return h.listWorkspace()
	case ToolReadSource:
		return h.readSource(args["path"])
	case ToolReadSummary:
		return h.readSummary()
	case ToolProposeSummaryEdit:
		return h.proposeSummaryEdit(args["text"])
	default:
		return ToolResult{Name: name, OK: false, Content: fmt.Sprintf("unknown tool %q", name)}
	}
}

func (h *ToolHost) listWorkspace() ToolResult {
	if h.FS == nil {
		return ToolResult{Name: ToolListWorkspace, OK: false, Content: "no workspace mounted"}
	}
	root, err := h.FS.ListRoot()
	if err != nil {
		return ToolResult{Name: ToolListWorkspace, OK: false, Content: "could not list workspace"}
	}
	if root == nil {
		return ToolResult{Name: ToolListWorkspace, OK: true, Content: "(empty workspace — open a research folder)"}
	}
	var b strings.Builder
	b.WriteString("Workspace tree (relative paths):\n")
	writeTree(&b, root, 0)
	out := b.String()
	if len(out) > 4000 {
		out = out[:3980] + "\n…"
	}
	return ToolResult{Name: ToolListWorkspace, OK: true, Content: out}
}

func writeTree(b *strings.Builder, n *models.FileTreeNode, depth int) {
	if n == nil {
		return
	}
	indent := strings.Repeat("  ", depth)
	if n.Kind == "folder" {
		fmt.Fprintf(b, "%s%s/\n", indent, n.Label)
	} else {
		p := n.Path
		if p == "" {
			p = n.Label
		}
		fmt.Fprintf(b, "%s%s  (%s)\n", indent, p, n.Ext)
	}
	for i := range n.Children {
		writeTree(b, &n.Children[i], depth+1)
	}
}

func (h *ToolHost) readSource(rel string) ToolResult {
	rel = strings.TrimSpace(rel)
	if rel == "" {
		return ToolResult{Name: ToolReadSource, OK: false, Content: "path required"}
	}
	if h.FS == nil || h.Docs == nil {
		return ToolResult{Name: ToolReadSource, OK: false, Content: "filesystem not available"}
	}
	// Reject absolute / parent escapes before Resolve (defense in depth).
	if filepath.IsAbs(rel) || strings.Contains(rel, "..") {
		return ToolResult{Name: ToolReadSource, OK: false, Content: "path rejected: must be relative and inside workspace"}
	}
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(rel), "."))
	if !allowedReadExt[ext] {
		return ToolResult{Name: ToolReadSource, OK: false, Content: fmt.Sprintf("extension .%s not readable via tools yet", ext)}
	}
	abs, err := h.FS.Resolve(rel)
	if err != nil {
		return ToolResult{Name: ToolReadSource, OK: false, Content: "path outside workspace or invalid"}
	}
	opened, err := h.Docs.OpenFile(abs)
	if err != nil {
		return ToolResult{Name: ToolReadSource, OK: false, Content: "could not read file"}
	}
	if opened.Reason != "" {
		return ToolResult{Name: ToolReadSource, OK: false, Content: opened.Detail}
	}
	text := blocksJSONToPlain(opened.BlocksJSON)
	if len(text) > 6000 {
		text = text[:5980] + "\n…[truncated]"
	}
	return ToolResult{
		Name:    ToolReadSource,
		OK:      true,
		Content: fmt.Sprintf("File %q (title %q):\n%s", rel, opened.Title, text),
	}
}

func (h *ToolHost) readSummary() ToolResult {
	snap := strings.TrimSpace(h.SummaryText)
	if snap == "" {
		return ToolResult{
			Name: ToolReadSummary,
			OK:   false,
			Content: "no summary snapshot available — open or create a research summary in this workspace",
		}
	}
	if len(snap) > 8000 {
		snap = snap[:7980] + "\n…[truncated]"
	}
	id := h.PrimarySummaryID
	if id == "" {
		id = "(primary)"
	}
	return ToolResult{
		Name:    ToolReadSummary,
		OK:      true,
		Content: fmt.Sprintf("Current research summary (%s):\n%s", id, snap),
	}
}

func (h *ToolHost) proposeSummaryEdit(text string) ToolResult {
	text = strings.TrimSpace(text)
	if text == "" {
		return ToolResult{Name: ToolProposeSummaryEdit, OK: false, Content: "text required for proposal"}
	}
	if len(text) > 2000 {
		text = text[:2000]
	}
	h.lastProposalText = text
	return ToolResult{
		Name: ToolProposeSummaryEdit,
		OK:   true,
		Content: fmt.Sprintf(
			"Queued summary insertion proposal (user must accept in UI):\n%s\nCitation: (AI draft)",
			text,
		),
	}
}

// LastProposal returns text from the latest successful propose_summary_edit.
func (h *ToolHost) LastProposal() string {
	if h == nil {
		return ""
	}
	return h.lastProposalText
}

// blocksJSONToPlain extracts segment text from OpenedDocument.BlocksJSON.
func blocksJSONToPlain(blocksJSON string) string {
	if strings.TrimSpace(blocksJSON) == "" {
		return ""
	}
	var wrap struct {
		Blocks []struct {
			Segments []struct {
				Text string `json:"text"`
			} `json:"segments"`
		} `json:"blocks"`
	}
	if err := json.Unmarshal([]byte(blocksJSON), &wrap); err != nil {
		return blocksJSON
	}
	var b strings.Builder
	for _, bl := range wrap.Blocks {
		for _, s := range bl.Segments {
			b.WriteString(s.Text)
		}
		b.WriteString("\n\n")
	}
	return strings.TrimSpace(b.String())
}
