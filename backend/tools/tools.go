package tools

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"Conductino/backend/docpipe"
	"Conductino/backend/extract"
	"Conductino/backend/mirror"
	"Conductino/backend/models"
)

// Tool names (stable wire identifiers).
const (
	ToolListWorkspace       = "list_workspace"
	ToolReadSource          = "read_source"
	ToolReadSummary         = "read_summary"
	ToolProposeSummaryEdit  = "propose_summary_edit"
	ToolSearchWorkspace     = "search_in_workspace"
	ToolRunWorkflow         = "run_workflow"
	ToolPublishSummary      = "publish_summary"
	// Mirror-pipeline tools (see mirror_tools.go + backend/docpipe/doc.go):
	ToolGetSummaryTarget = "get_summary_target" // which .docx receives summaries
	ToolReadMirror       = "read_mirror"        // document as the user sees it (pages + metadata)
	ToolReadSkill        = "read_skill"         // one instruction file from skills/
	ToolShell            = "shell"              // basic sandboxed command in the open folder
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
// extract/ids.go); pass the workspace-relative token.
type FileOpener interface {
	OpenFile(absPath, relKey string) (models.OpenedDocument, error)
}

// ToolHost holds per-request deps for folder-scoped tools.
type ToolHost struct {
	FS               PathResolver
	Docs             FileOpener
	SummaryText      string // snapshot from frontend (Phase 6 → storage)
	PrimarySummaryID string
	// SummaryPath is the workspace-relative path token of the mapped summary
	// file (frontend document metadata). publish_summary writes through ONLY
	// here — never a guessed location. Empty = in-memory summary, unpublishable.
	SummaryPath string
	// Mirror is the summary .md working copy (plan 10 §2). Nil → tools
	// degrade to snapshot-only; never a failure by itself.
	Mirror *mirror.Store
	// Workflow runs named multi-step routines (plan 11 §2: backend/workflows).
	// Nil → run_workflow errors honestly. Interface (not import) so tools
	// never depends on the runner package — only the runner depends on tools.
	Workflow WorkflowRunner
	// proposals accumulates EVERY successful propose_summary_edit this turn
	// (plan 11 §3: multi-proposal surfacing). lastProposal stays the latest
	// for single-proposal callers.
	proposals []Proposal
	// published accumulates successful publish_summary writes this turn.
	published    []PublishedDoc
	lastProposal Proposal // set by propose_summary_edit for final payload
}

// WorkflowRunner executes one named routine against this host and returns
// its report text (ok=false → honest error, nothing ran).
type WorkflowRunner interface {
	Run(name string, args map[string]string, h Dispatcher) (string, bool)
}

// Dispatcher is what a routine drives: guarded tool dispatch plus a bounded,
// readable file listing. *ToolHost satisfies it.
type Dispatcher interface {
	Dispatch(name string, args map[string]string) ToolResult
	Files() []string
}

// Proposal is one structured summary edit: full rewrite power, not
// append-only. insert adds a paragraph; modify rewrites the target span;
// delete strikes it. The user gatekeeps every one in the UI.
type Proposal struct {
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
// prompts, or Proposal text (plan 02 §5).
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

func AuditTool(name string, args map[string]string, ok bool) {
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

// ToolAuditSummary counts calls per tool for the usage line ("tools: 4 calls").
func ToolAuditSummary() string {
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
<tool name="read_mirror" path="relative/path.docx"/>
<tool name="read_summary"/>
<tool name="get_summary_target"/>
<tool name="propose_summary_edit" op="insert" text="1-2 factual sentences"/>
<tool name="propose_summary_edit" op="modify" target="block id or quoted span" text="replacement text"/>
<tool name="propose_summary_edit" op="delete" target="block id or quoted span"/>
<tool name="search_in_workspace" query="keyword"/>
<tool name="run_workflow" workflow="summarize-folder"/>
<tool name="run_workflow" workflow="add-to-summary" source="relative/path.docx" topic="what to extract"/>
<tool name="publish_summary"/>
<tool name="read_skill" path="my_instructions/reading-rules.md"/>
<tool name="shell" command="ls -la"/>

Rules:
- read_mirror shows a document the way the user SEES it in the app: page by
  page, with the metadata header (kind, page size, font, editable). Prefer it
  over read_source when the user asks about pages or "what am I looking at".
- get_summary_target names the .docx the user marked to receive ALL summaries
  in this workspace. Before summarizing, call it — never guess a target file.
  propose_summary_edit + publish_summary write into it.
- When the user mentions a path like @skills/my_instructions/foo.md, read it
  with read_skill and follow those instructions (they are user-authored).
- shell runs ONE basic command (10s timeout) in the open research folder for
  simple checks (list, count, grep). It is sandboxed and read-minded — never
  use it to modify or delete files; the document tools do that safely.
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
- The mapped summary is openly editable: after proposing, call
  publish_summary (no args) to write the working copy through to the .docx.
  Then report what changed. Sources stay read-only — only the summary path
  the app mapped can ever be written.
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
	case ToolRunWorkflow:
		return h.runWorkflow(args)
	case ToolPublishSummary:
		return h.publishSummary()
	case ToolGetSummaryTarget:
		return h.getSummaryTarget()
	case ToolReadMirror:
		return h.readMirror(args["path"])
	case ToolReadSkill:
		return h.readSkill(args["path"])
	case ToolShell:
		return h.shell(args["command"])
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

func (h *ToolHost) readSummary() ToolResult {
	snap := strings.TrimSpace(h.SummaryText)
	id := h.PrimarySummaryID
	if id == "" {
		id = "(primary)"
	}
	// Mirror first: the working copy accumulates AI edits across turns and
	// re-syncs when the frontend snapshot moves under it (user saved).
	// An empty summary is valid content ("no content yet"), not an error —
	// but only when a summary id is known; without one there is nothing to read.
	if h.Mirror != nil && h.PrimarySummaryID != "" {
		if md, log, err := h.Mirror.Read(h.PrimarySummaryID, snap); err == nil && strings.TrimSpace(md) != "" {
			var b strings.Builder
			fmt.Fprintf(&b, "Current research summary (%s) — working copy:\n%s", h.PrimarySummaryID, md)
			if len(log) > 0 {
				b.WriteString("\n\n[edits applied this session:")
				for _, e := range log {
					fmt.Fprintf(&b, "\n- %s %s %s", e.Op, e.Target, e.Note)
				}
				b.WriteString("]")
			}
			return ToolResult{Name: ToolReadSummary, OK: true, Content: truncateToolText(b.String(), 8000)}
		}
		if strings.TrimSpace(snap) == "" {
			return ToolResult{Name: ToolReadSummary, OK: true, Content: fmt.Sprintf("Current research summary (%s): (empty — no content yet)", h.PrimarySummaryID)}
		}
	}
	if snap == "" {
		return ToolResult{
			Name:    ToolReadSummary,
			OK:      false,
			Content: "no summary snapshot available — open or create a research summary in this workspace",
		}
	}
	return ToolResult{
		Name:    ToolReadSummary,
		OK:      true,
		Content: fmt.Sprintf("Current research summary (%s):\n%s", id, truncateToolText(snap, 8000)),
	}
}

// truncateToolText caps model-visible text with a machine-readable marker.
func truncateToolText(s string, max int) string {
	if len(s) > max {
		return s[:max-20] + "\n…[truncated]"
	}
	return s
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
	h.lastProposal = Proposal{Op: op, Target: target, OldText: oldText, NewText: text, Highlight: highlight}
	h.proposals = append(h.proposals, h.lastProposal)
	verb := map[string]string{"insert": "insertion", "modify": "revision", "delete": "deletion"}[op]
	// Mirror the edit into the summary working copy so follow-up turns read
	// composed state (plan 11 §3). Best-effort: the queued proposal is the
	// contract; the mirror is the convenience. Read first so a propose
	// before any read still lands (creates the working copy from snapshot).
	mirrorNote := ""
	if h.Mirror != nil && h.PrimarySummaryID != "" {
		if _, _, rerr := h.Mirror.Read(h.PrimarySummaryID, h.SummaryText); rerr == nil {
			if applied, note, merr := h.Mirror.Apply(h.PrimarySummaryID, op, target, oldText, text); merr == nil {
				if applied {
					mirrorNote = " Mirror working copy updated."
				} else {
					mirrorNote = " Mirror note: " + note + "."
				}
			}
		}
	}
	return ToolResult{
		Name: ToolProposeSummaryEdit,
		OK:   true,
		Content: fmt.Sprintf(
			"Queued summary %s proposal (user must accept in UI):\n%s\nCitation: (AI draft)%s",
			verb, text, mirrorNote,
		),
	}
}

// runWorkflow executes one named routine (plan 11 §2). add-to-summary queues
// user-gated proposals; all routines report honestly — ok=false means nothing
// ran. Unknown names list what exists.
func (h *ToolHost) runWorkflow(args map[string]string) ToolResult {
	name := strings.ToLower(strings.TrimSpace(args["workflow"]))
	if h.Workflow == nil {
		return ToolResult{Name: ToolRunWorkflow, OK: false, Content: "workflows unavailable in this build"}
	}
	if name == "" {
		return ToolResult{Name: ToolRunWorkflow, OK: false, Content: "workflow required (available: summarize-folder, add-to-summary)"}
	}
	out, ok := h.Workflow.Run(name, args, h)
	return ToolResult{Name: ToolRunWorkflow, OK: ok, Content: out}
}

// Files lists readable workspace-relative paths (live tree, capped).
// Workflows use it to plan bounded multi-file routines.
func (h *ToolHost) Files() []string {
	if h.FS == nil {
		return nil
	}
	root, err := h.FS.ListRoot()
	if err != nil || root == nil {
		return nil
	}
	var out []string
	var walk func(n *models.FileTreeNode)
	walk = func(n *models.FileTreeNode) {
		if n == nil || len(out) >= searchMaxFiles {
			return
		}
		if n.Kind != "folder" {
			rel := n.Path
			if rel == "" {
				rel = n.Label
			}
			ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(rel), "."))
			if allowedReadExt[ext] && !filepath.IsAbs(rel) && !strings.Contains(rel, "..") {
				out = append(out, rel)
			}
		}
		for i := range n.Children {
			walk(&n.Children[i])
		}
	}
	walk(root)
	return out
}

// Proposals returns every successful propose_summary_edit this turn, in
// order. The chat loop surfaces all of them so multi-claim turns never
// silently drop earlier edits (only the last used to survive).
func (h *ToolHost) Proposals() []Proposal {
	if h == nil {
		return nil
	}
	return h.proposals
}

// PublishedDiff is one write-through edit the UI decorates in-file.
type PublishedDiff struct {
	Op     string // insert | modify | delete
	Target string // block id or span (modify/delete)
	Text   string // new content (insert/modify) or removed span (delete)
	// OldText is the pre-image (modify/delete) for reject-restore.
	OldText string
	Note    string // mirror note (e.g. "span not found — appended as insert instead")
}

// PublishedDoc is one summary file written this turn.
type PublishedDoc struct {
	SummaryID string
	Path      string
	Diffs     []PublishedDiff
}

// PublishedDocs returns every successful publish_summary this turn.
func (h *ToolHost) PublishedDocs() []PublishedDoc {
	if h == nil {
		return nil
	}
	return h.published
}

// CanPublish reports whether an automatic write-through has everything it
// needs. Two paths satisfy it:
//
//	A) the mapped in-app summary (Mirror + PrimarySummaryID + SummaryPath), or
//	B) the workspace summary TARGET file (marked in the library) with at
//	   least one proposal this turn to siphon into it.
//
// The chat loop uses it for the single-path guarantee: proposed-but-
// unpublished turns publish themselves instead of stranding card-less
// proposals.
func (h *ToolHost) CanPublish() bool {
	if h == nil || h.FS == nil {
		return false
	}
	// Path A: mapped summary with a mirror working copy.
	if h.Mirror != nil && strings.TrimSpace(h.PrimarySummaryID) != "" {
		rel := strings.TrimSpace(h.SummaryPath)
		if rel != "" && !filepath.IsAbs(rel) && !strings.Contains(rel, "..") {
			return true
		}
	}
	// Path B: the user-marked summary target file for THIS workspace.
	if target, ok := docpipe.LoadTarget(); ok && target.Root == h.FS.Root() && len(h.Proposals()) > 0 {
		return true
	}
	return false
}

// publishSummary writes proposed edits through to a summary .docx.
//
// Path A (original): the mapped in-app summary — the mirror working copy is
// written to SummaryPath (Resolve-jailed like every other path), and the
// unpublished mirror ops become in-file diffs for the UI.
//
// Path B (summary target siphon): when no summary document is mapped, the
// workspace SUMMARY TARGET file (right-click → "Make this the target summary
// document") receives this turn's proposals directly. That is how "all
// summaries are siphoned to that document" works even when it is not open.
func (h *ToolHost) publishSummary() ToolResult {
	fail := func(msg string) ToolResult {
		return ToolResult{Name: ToolPublishSummary, OK: false, Content: msg}
	}
	if h.Mirror == nil || h.PrimarySummaryID == "" || strings.TrimSpace(h.SummaryPath) == "" || h.FS == nil {
		// Path A unavailable — try the user-marked summary target file.
		return h.publishToTarget(fail)
	}
	rel := strings.TrimSpace(h.SummaryPath)
	if filepath.IsAbs(rel) || strings.Contains(rel, "..") {
		return fail("summary path rejected: must be relative and inside workspace")
	}
	if _, err := h.FS.Resolve(rel); err != nil {
		return fail("summary path outside workspace or invalid")
	}
	md, _, err := h.Mirror.Read(h.PrimarySummaryID, h.SummaryText)
	if err != nil {
		return fail("could not read summary working copy: " + err.Error())
	}
	if strings.TrimSpace(md) == "" {
		return fail("summary working copy is empty — nothing to publish")
	}
	blocks, err := extract.BlocksFromMarkdown(md, rel)
	if err != nil {
		return fail("could not shape working copy into blocks: " + err.Error())
	}
	wire, err := extract.BlocksToJSON(blocks)
	if err != nil {
		return fail("could not encode blocks: " + err.Error())
	}
	if _, err := extract.WriteSummaryDOCXPath(h.FS.Root(), rel, wire); err != nil {
		return fail("could not write summary file: " + err.Error())
	}
	var diffs []PublishedDiff
	for _, e := range h.Mirror.Unpublished(h.PrimarySummaryID) {
		diffs = append(diffs, PublishedDiff{Op: e.Op, Target: e.Target, Text: e.Text, OldText: e.OldText, Note: e.Note})
	}
	_ = h.Mirror.MarkPublished(h.PrimarySummaryID)
	h.published = append(h.published, PublishedDoc{SummaryID: h.PrimarySummaryID, Path: rel, Diffs: diffs})
	var b strings.Builder
	fmt.Fprintf(&b, "Published %d edit(s) to %q. Reload it to see the changes.", len(diffs), rel)
	for _, d := range diffs {
		fmt.Fprintf(&b, "\n- %s %s %.80s", d.Op, d.Target, strings.TrimSpace(d.Text))
	}
	return ToolResult{Name: ToolPublishSummary, OK: true, Content: b.String()}
}

// publishToTarget is Path B of publish_summary: siphon this turn's proposals
// into the workspace summary TARGET file (docpipe.SummaryTarget). The file is
// read fresh from disk, the proposals are applied to its blocks, and the
// .docx is rewritten — then the UI gets the same PublishedDoc/Diffs payload
// as Path A, so yellow in-file review works identically when it is open.
func (h *ToolHost) publishToTarget(fail func(string) ToolResult) ToolResult {
	if h.FS == nil || h.Docs == nil {
		return fail("no summary mapped and no workspace available — open the target document and choose “Make summary”, or mark a summary target in the library")
	}
	target, ok := docpipe.LoadTarget()
	if !ok || target.Root != h.FS.Root() {
		return fail("no summary mapped, and no summary target is marked for this workspace — ask the user to right-click a .docx in the library and choose “Make this the target summary document”")
	}
	proposals := h.Proposals()
	if len(proposals) == 0 {
		return fail("nothing proposed yet — call propose_summary_edit first, then publish_summary")
	}
	abs, err := h.FS.Resolve(target.RelPath)
	if err != nil {
		return fail("summary target is outside the workspace: " + target.RelPath)
	}
	opened, err := h.Docs.OpenFile(abs, target.RelPath)
	if err != nil {
		return fail("could not read the summary target: " + err.Error())
	}
	if opened.Reason != "" {
		return fail("could not read the summary target: " + opened.Reason + " " + opened.Detail)
	}
	blocks, err := extract.ParseBlocksJSON(opened.BlocksJSON)
	if err != nil {
		return fail("could not parse the summary target: " + err.Error())
	}
	blocks, applied := applyProposalsToBlocks(blocks, proposals)
	wire, err := extract.BlocksToJSON(blocks)
	if err != nil {
		return fail("could not encode blocks: " + err.Error())
	}
	if _, err := extract.WriteSummaryDOCXPath(h.FS.Root(), target.RelPath, wire); err != nil {
		return fail("could not write the summary target: " + err.Error())
	}
	var diffs []PublishedDiff
	for _, p := range applied {
		op := p.Op
		if op == "" {
			op = "insert"
		}
		diffs = append(diffs, PublishedDiff{Op: op, Target: p.Target, Text: p.NewText, OldText: p.OldText, Note: "siphoned via summary target"})
	}
	h.published = append(h.published, PublishedDoc{SummaryID: h.PrimarySummaryID, Path: target.RelPath, Diffs: diffs})
	var b strings.Builder
	fmt.Fprintf(&b, "Siphoned %d edit(s) into the summary target %q. Reload it to see the changes.", len(diffs), target.RelPath)
	for _, d := range diffs {
		fmt.Fprintf(&b, "\n- %s %s %.80s", d.Op, d.Target, strings.TrimSpace(d.Text))
	}
	return ToolResult{Name: ToolPublishSummary, OK: true, Content: b.String()}
}

// applyProposalsToBlocks applies proposals to a fresh block list:
//   - insert: append one new paragraph block,
//   - modify: find the first block containing OldText (or the Target span)
//     and replace that block's text,
//   - delete: find and remove that block.
//
// Unmatched modify/delete fall back to an honest insert (never lost text).
// Returns the new block list and the proposals actually applied.
func applyProposalsToBlocks(blocks []extract.DocxBlock, proposals []Proposal) ([]extract.DocxBlock, []Proposal) {
	applied := make([]Proposal, 0, len(proposals))
	fullText := func(b extract.DocxBlock) string {
		var sb strings.Builder
		for _, s := range b.Segments {
			sb.WriteString(s.Text)
		}
		return sb.String()
	}
	for _, p := range proposals {
		text := strings.TrimSpace(p.NewText)
		op := strings.ToLower(strings.TrimSpace(p.Op))
		if op == "" {
			op = "insert"
		}
		needle := strings.TrimSpace(p.OldText)
		if needle == "" {
			needle = strings.TrimSpace(p.Target)
		}
		switch op {
		case "modify", "delete":
			idx := -1
			if needle != "" {
				for i, b := range blocks {
					if strings.Contains(fullText(b), needle) {
						idx = i
						break
					}
				}
			}
			if idx >= 0 && op == "delete" {
				blocks = append(blocks[:idx], blocks[idx+1:]...)
				applied = append(applied, p)
				continue
			}
			if idx >= 0 && text != "" {
				blocks[idx].Segments = []extract.DocxSegment{{Text: text}}
				applied = append(applied, p)
				continue
			}
			// Not found — fall through to insert so the text is never lost.
		}
		if text != "" {
			newID := fmt.Sprintf("ai-%d", len(blocks)+len(applied)+1)
			blocks = append(blocks, extract.DocxBlock{ID: newID, Type: "paragraph", Segments: []extract.DocxSegment{{Text: text}}})
			applied = append(applied, p)
		}
	}
	return blocks, applied
}

// LastProposal returns the latest successful propose_summary_edit.
func (h *ToolHost) LastProposal() Proposal {
	if h == nil {
		return Proposal{}
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
