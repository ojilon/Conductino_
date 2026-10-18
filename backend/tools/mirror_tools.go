package tools

import (
	"fmt"
	"strings"

	"Conductino/backend/docpipe"
	"Conductino/backend/skills"
)

// The mirror-pipeline tools (see backend/docpipe/doc.go for the pipeline):
//
//	get_summary_target — which .docx receives summaries in this workspace
//	read_mirror        — the document as the user SEES it (pages + metadata)
//	read_skill         — one instruction file from the skills/ folder
//	shell              — basic sandboxed command in the open folder
//
// They live in their own file so tools.go keeps its original shape.

// readMirrorMaxChars caps what one read_mirror call returns to the model.
const readMirrorMaxChars = 12000

// getSummaryTarget tells the model where summaries belong. The target is
// the file the user marked in the library ("Make this the target summary
// document"); it lives in a tiny JSON marker both the UI and tools read.
func (h *ToolHost) getSummaryTarget() ToolResult {
	fail := func(msg string) ToolResult {
		return ToolResult{Name: ToolGetSummaryTarget, OK: false, Content: msg}
	}
	target, ok := docpipe.LoadTarget()
	if !ok {
		return fail("no summary target is marked in this workspace yet — ask the user to right-click a .docx in the library and choose “Make this the target summary document”")
	}
	if h.FS != nil && target.Root != "" && target.Root != h.FS.Root() {
		return fail(fmt.Sprintf("the marked summary target (%s) belongs to a different folder than the open workspace", target.RelPath))
	}
	return ToolResult{
		Name: ToolGetSummaryTarget,
		OK:   true,
		Content: fmt.Sprintf("Summary target: %q (title %q). Route summaries and highlighted text here: propose edits with propose_summary_edit, then publish_summary writes them into this file.",
			target.RelPath, target.Title),
	}
}

// readMirror returns the document the way the frontend canvas shows it:
// page-oriented text plus the metadata header (kind, pages, page size,
// font, editable). This is the AI's "what the user is seeing" view.
func (h *ToolHost) readMirror(rel string) ToolResult {
	fail := func(msg string) ToolResult {
		return ToolResult{Name: ToolReadMirror, OK: false, Content: msg}
	}
	rel = strings.TrimSpace(rel)
	if rel == "" {
		return fail("path required")
	}
	if h.FS == nil {
		return fail("no workspace mounted")
	}
	// Same forgiving resolution as read_source: canonicalize the model-typed
	// path against the live tree (missing extension, typos → did-you-mean).
	if canonical, suggestions := h.resolveSourcePath(rel); canonical != "" {
		rel = canonical
	} else if len(suggestions) > 0 {
		return fail(fmt.Sprintf("no file %q in the workspace — did you mean: %s", rel, strings.Join(suggestions, ", ")))
	}
	content, meta, ok := docpipe.LoadFor(h.FS.Root(), rel)
	if !ok {
		return fail(fmt.Sprintf("no mirror for %q — mirrors are built when a file is opened in the app; use read_source for raw extraction instead", rel))
	}
	return ToolResult{
		Name:    ToolReadMirror,
		OK:      true,
		Content: docpipe.DescribeForAI(content, meta, readMirrorMaxChars),
	}
}

// readSkill returns one instruction file from the skills folder. The user
// points the model at custom instructions with "@skills/<path>" in chat;
// this tool is how the model actually reads them.
func (h *ToolHost) readSkill(rel string) ToolResult {
	fail := func(msg string) ToolResult {
		return ToolResult{Name: ToolReadSkill, OK: false, Content: msg}
	}
	rel = strings.TrimSpace(rel)
	if rel == "" {
		return fail("path required (relative to the skills folder, e.g. my_instructions/reading-rules.md)")
	}
	text, err := skills.ReadFile(rel)
	if err != nil {
		return fail(err.Error())
	}
	if len(text) > readMirrorMaxChars {
		text = text[:readMirrorMaxChars] + "\n… [skill file truncated]"
	}
	return ToolResult{Name: ToolReadSkill, OK: true, Content: text}
}

// shell runs one basic command in the sandbox (see shell.go): cwd is the
// open research folder, 10s timeout, capped output, small deny-list.
func (h *ToolHost) shell(command string) ToolResult {
	if strings.TrimSpace(command) == "" {
		return ToolResult{Name: ToolShell, OK: false, Content: "command required"}
	}
	workDir := ""
	if h.FS != nil {
		workDir = h.FS.Root()
	}
	out, err := RunShell(workDir, command)
	if err != nil {
		return ToolResult{Name: ToolShell, OK: false, Content: err.Error()}
	}
	return ToolResult{Name: ToolShell, OK: true, Content: out}
}
