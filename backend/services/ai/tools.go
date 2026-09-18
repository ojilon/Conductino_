package ai

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"Conductino/backend/models"
)

// Tool names (stable wire identifiers).
const (
	ToolListWorkspace       = "list_workspace"
	ToolReadSource          = "read_source"
	ToolReadSummary         = "read_summary"
	ToolProposeSummaryEdit  = "propose_summary_edit"
	ToolSearchWorkspace     = "search_in_workspace"
)

// Allowed source extensions for read_source (lowercase, no dot).
// txt/md via the text arm; pdf via text-layer extraction; docx via stdlib
// OOXML. Rejections stay explicit so tools never invent content.
var allowedReadExt = map[string]bool{
	"txt": true, "md": true, "markdown": true, "pdf": true, "docx": true,
}

// PathResolver is the workspace containment boundary (Filesystem.Resolve + tree).
type PathResolver interface {
	Resolve(path string) (string, error)
	Root() string
	ListRoot() (*models.FileTreeNode, error)
}

// FileOpener extracts displayable content from an absolute path already
// Resolve-checked by the host. relKey seeds stable block IDs (see
// services/blockids.go); pass the workspace-relative token.
type FileOpener interface {
	OpenFile(absPath, relKey string) (models.OpenedDocument, error)
}

// ToolHost holds per-request deps for folder-scoped tools.
type ToolHost struct {
	FS                 PathResolver
	Docs               FileOpener
	SummaryText        string // snapshot from frontend (Phase 6 → storage)
	PrimarySummaryID   string
	lastProposal       proposal // set by propose_summary_edit for final payload
}

// proposal is one structured summary edit: full rewrite power, not
// append-only. insert adds a paragraph; modify rewrites the target span;
// delete strikes it. The user gatekeeps every one in the UI.
type proposal struct {
	Op          string // insert | modify | delete
	Target      string // block id or quoted span in the summary (modify/delete)
	OldText     string // span being replaced/removed (modify/delete fallback)
	NewText     string
	Highlight   string // exact substring the UI underlines (modify)
}

// ToolResult is what the model sees after a tool runs.
type ToolResult struct {
	Name    string
	OK      bool
	Content string
}

// Tool audit (issue 17): bounded ring of tool invocations for debugging and
// usage review. Only names + relative paths are logged — never file content,
// prompts, or proposal text (plan 02 §5).
type toolAuditEntry struct {
	At   int64  // Unix millis
	Tool string // tool name
	Arg  string // redacted arg summary (path / op+target / query)
	OK   bool
}

const toolAuditCap = 100

var (
	toolAuditMu sync.Mutex
	toolAudit   []toolAuditEntry
)

func auditTool(name string, args map[string]string, ok bool) {
	arg := ""
	switch strings.ToLower(strings.TrimSpace(name)) {
	case ToolReadSource, ToolSearchWorkspace:
		arg = args["path"] + args["query"]
	case ToolProposeSummaryEdit:
		arg = "op=" + args["op"] + " target=" + args["target"]
	}
	arg = strings.TrimSpace(arg)
	if len(arg) > 120 {
		arg = arg[:120]
	}
	toolAuditMu.Lock()
	defer toolAuditMu.Unlock()
	if len(toolAudit) >= toolAuditCap {
		toolAudit = toolAudit[1:]
	}
	toolAudit = append(toolAudit, toolAuditEntry{
		At:   time.Now().UnixMilli(),
		Tool: name,
		Arg:  arg,
		OK:   ok,
	})
}

// toolAuditSummary counts calls per tool for the usage line ("tools: 4 calls").
func toolAuditSummary() string {
	toolAuditMu.Lock()
	defer toolAuditMu.Unlock()
	if len(toolAudit) == 0 {
		return ""
	}
	counts := map[string]int{}
	var order []string
	for _, e := range toolAudit {
		if _, ok := counts[e.Tool]; !ok {
			order = append(order, e.Tool)
		}
		counts[e.Tool]++
	}
	parts := make([]string, 0, len(order))
	for _, t := range order {
		parts = append(parts, fmt.Sprintf("%s×%d", t, counts[t]))
	}
	return "tools: " + strings.Join(parts, " ")
}

// Catalog returns a short system description of available tools for the prompt.
func ToolCatalog() string {
	return strings.TrimSpace(`
### Tools (optional)
You may call at most two tools before answering. Use ONLY this XML form on its own line:
<tool name="list_workspace"/>
<tool name="read_source" path="relative/path.txt"/>
<tool name="read_summary"/>
<tool name="propose_summary_edit" op="insert" text="1-2 factual sentences"/>
<tool name="propose_summary_edit" op="modify" target="block id or quoted span" text="replacement text"/>
<tool name="propose_summary_edit" op="delete" target="block id or quoted span"/>
<tool name="search_in_workspace" query="keyword"/>

Rules:
- Paths are relative to the open research folder. Never use absolute paths or "..".
- read_source reads .txt/.md/.pdf/.docx. If the exact name is slightly off
  (missing extension, typo), you get a did-you-mean list — retry with it.
- Prefer search_in_workspace over guessing file contents when the user asks
  "where", "which file", or "find" — it returns quoted snippets, never full dumps.
- The summary is a living document: harmonize new material with what is already
  there — redefine a stale definition, restructure a section, or remove what a
  new source disproves. Never restrict yourself to appending.
- Every propose_summary_edit drafts exactly one change for the user to accept;
  it does not write files. For modify/delete, name the target span precisely.
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
			// Values serialized by toolCallToTag are XML-escaped.
			args[k] = xmlAttrUnescape(a[2])
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

// xmlAttrUnescape reverses toolCallToTag escaping (openai_compat.go).
func xmlAttrUnescape(s string) string {
	r := strings.NewReplacer(`&quot;`, `"`, `&lt;`, `<`, `&gt;`, `>`, `&amp;`, `&`)
	return r.Replace(s)
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
		return h.proposeSummaryEdit(args["text"], args["op"], args["target"], args["oldtext"])
	case ToolSearchWorkspace:
		return h.searchWorkspace(args["query"])
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
	// Resolve against the live tree first: exact (case-insensitive) →
	// stem-with-extension ("Summary" → "Summary.docx") → did-you-mean.
	// Models regularly drop extensions or mistype a letter; without this
	// every near-miss becomes "can't read the file".
	note := ""
	if canonical, suggestions := h.resolveSourcePath(rel); canonical != "" {
		if !strings.EqualFold(canonical, rel) {
			note = fmt.Sprintf("Resolved %q to %q.\n", rel, canonical)
		}
		rel = canonical
	} else if len(suggestions) > 0 {
		quoted := make([]string, 0, len(suggestions))
		for _, s := range suggestions {
			quoted = append(quoted, fmt.Sprintf("%q", s))
		}
		return ToolResult{Name: ToolReadSource, OK: false, Content: fmt.Sprintf(
			"file %q not found in workspace — did you mean: %s?", rel, strings.Join(quoted, ", "))}
	}
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(rel), "."))
	if !allowedReadExt[ext] {
		return ToolResult{Name: ToolReadSource, OK: false, Content: fmt.Sprintf("extension .%s not readable via tools yet", ext)}
	}
	abs, err := h.FS.Resolve(rel)
	if err != nil {
		return ToolResult{Name: ToolReadSource, OK: false, Content: "path outside workspace or invalid"}
	}
	opened, err := h.Docs.OpenFile(abs, rel)
	if err != nil {
		return ToolResult{Name: ToolReadSource, OK: false, Content: "could not read file"}
	}
	if opened.Reason != "" {
		// A missing file after a jail-clean resolve gets suggestions too.
		if _, suggestions := h.resolveSourcePath(rel); len(suggestions) > 0 {
			quoted := make([]string, 0, len(suggestions))
			for _, s := range suggestions {
				quoted = append(quoted, fmt.Sprintf("%q", s))
			}
			return ToolResult{Name: ToolReadSource, OK: false, Content: fmt.Sprintf(
				"could not read %q (%s) — did you mean: %s?", rel, opened.Detail, strings.Join(quoted, ", "))}
		}
		return ToolResult{Name: ToolReadSource, OK: false, Content: opened.Detail}
	}
	text := blocksJSONToPlain(opened.BlocksJSON)
	if len(text) > 6000 {
		text = text[:5980] + "\n…[truncated]"
	}
	return ToolResult{
		Name:    ToolReadSource,
		OK:      true,
		Content: fmt.Sprintf("%sFile %q (title %q):\n%s", note, rel, opened.Title, text),
	}
}

// resolveSourcePath maps a model-typed path to the canonical tree path.
// Returns ("", nil) when the tree is unavailable (caller tries direct) or
// ("", suggestions) when nothing matches. Matching order: exact
// case-insensitive → unique stem (extension-insensitive) → normalized
// fuzzy containment, capped at 3 suggestions.
func (h *ToolHost) resolveSourcePath(rel string) (string, []string) {
	if h.FS == nil {
		return "", nil
	}
	root, err := h.FS.ListRoot()
	if err != nil || root == nil {
		return "", nil
	}
	var files []string
	var walk func(n *models.FileTreeNode)
	walk = func(n *models.FileTreeNode) {
		if n == nil {
			return
		}
		if n.Kind != "folder" {
			p := n.Path
			if p == "" {
				p = n.Label
			}
			if p != "" {
				files = append(files, p)
			}
		}
		for i := range n.Children {
			walk(&n.Children[i])
		}
	}
	walk(root)
	for _, f := range files {
		if strings.EqualFold(f, rel) {
			return f, nil
		}
	}
	stem := strings.ToLower(strings.TrimSuffix(rel, filepath.Ext(rel)))
	var stemHits []string
	for _, f := range files {
		if strings.ToLower(strings.TrimSuffix(f, filepath.Ext(f))) == stem {
			stemHits = append(stemHits, f)
		}
	}
	if len(stemHits) == 1 {
		return stemHits[0], nil
	}
	if len(stemHits) > 1 {
		return "", stemHits[:min(3, len(stemHits))]
	}
	norm := func(s string) string {
		var b strings.Builder
		for _, r := range strings.ToLower(s) {
			if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
				b.WriteRune(r)
			}
		}
		return b.String()
	}
	// Stems: the extension ("x" vs "x.docx") must not count as distance.
	nq := norm(strings.TrimSuffix(filepath.Base(rel), filepath.Ext(rel)))
	type scored struct {
		path string
		d    int
	}
	var ranked []scored
	for _, f := range files {
		nf := norm(strings.TrimSuffix(filepath.Base(f), filepath.Ext(f)))
		if nq == "" || nf == "" {
			continue
		}
		d := levenshtein(nq, nf)
		// Containment is an exact-enough hit regardless of length gap.
		if strings.Contains(nf, nq) || strings.Contains(nq, nf) {
			d = 0
		}
		allow := len(nq) / 6
		if allow < 2 {
			allow = 2
		}
		if ll := len(nf); ll/6 > allow {
			allow = ll / 6
		}
		if d <= allow {
			ranked = append(ranked, scored{f, d})
		}
	}
	sort.Slice(ranked, func(i, j int) bool { return ranked[i].d < ranked[j].d })
	var fuzzy []string
	for i := 0; i < len(ranked) && i < 3; i++ {
		fuzzy = append(fuzzy, ranked[i].path)
	}
	if len(fuzzy) > 0 {
		return "", fuzzy
	}
	return "", nil
}

// levenshtein is rune-wise edit distance for did-you-mean ranking.
// Inputs are short normalized stems, so the O(n·m) table is trivial.
func levenshtein(a, b string) int {
	ar, br := []rune(a), []rune(b)
	if len(ar) == 0 {
		return len(br)
	}
	if len(br) == 0 {
		return len(ar)
	}
	prev := make([]int, len(br)+1)
	for j := range prev {
		prev[j] = j
	}
	for i, ca := range ar {
		cur := make([]int, len(br)+1)
		cur[0] = i + 1
		for j, cb := range br {
			cost := 0
			if ca != cb {
				cost = 1
			}
			del, ins, sub := prev[j+1]+1, cur[j]+1, prev[j]+cost
			cur[j+1] = del
			if ins < cur[j+1] {
				cur[j+1] = ins
			}
			if sub < cur[j+1] {
				cur[j+1] = sub
			}
		}
		prev = cur
	}
	return prev[len(br)]
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

func (h *ToolHost) proposeSummaryEdit(text, op, target, oldText string) ToolResult {	text = strings.TrimSpace(text)
	op = strings.ToLower(strings.TrimSpace(op))
	if op == "" {
		op = "insert"
	}
	if op != "insert" && op != "modify" && op != "delete" {
		return ToolResult{Name: ToolProposeSummaryEdit, OK: false, Content: `op must be insert, modify, or delete`}
	}
	target = strings.TrimSpace(target)
	oldText = strings.TrimSpace(oldText)
	if op != "insert" && target == "" && oldText == "" {
		return ToolResult{Name: ToolProposeSummaryEdit, OK: false, Content: "modify/delete need a target block id or quoted span"}
	}
	if op != "delete" && text == "" {
		return ToolResult{Name: ToolProposeSummaryEdit, OK: false, Content: "text required for proposal"}
	}
	if len(text) > 2000 {
		text = text[:2000]
	}
	highlight := ""
	if op == "modify" && oldText != "" && strings.Contains(text, oldText[:min(24, len(oldText))]) {
		highlight = oldText
	}
	h.lastProposal = proposal{Op: op, Target: target, OldText: oldText, NewText: text, Highlight: highlight}
	verb := map[string]string{"insert": "insertion", "modify": "revision", "delete": "deletion"}[op]
	return ToolResult{
		Name: ToolProposeSummaryEdit,
		OK:   true,
		Content: fmt.Sprintf(
			"Queued summary %s proposal (user must accept in UI):\n%s\nCitation: (AI draft)",
			verb, text,
		),
	}
}

// LastProposal returns the latest successful propose_summary_edit.
func (h *ToolHost) LastProposal() proposal {
	if h == nil {
		return proposal{}
	}
	return h.lastProposal
}

// LastProposalText is the legacy text-only accessor (insert path).
func (h *ToolHost) LastProposalText() string {
	return h.LastProposal().NewText
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
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
