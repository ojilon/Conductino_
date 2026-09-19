package workflows

import (
	"fmt"
	"strings"

	"Conductino/backend/tools"
)

// Host is what a routine drives: guarded tool dispatch plus a bounded,
// readable file listing. *tools.ToolHost satisfies it (via tools.Dispatcher).
type Host = tools.Dispatcher

// Runner executes named routines. Stateless across turns except for the
// process-local completed ledger (see Run); all side effects flow through
// Host.Dispatch, so guards + audit stay in the tool layer.
type Runner struct {
	// MaxSources caps files per summarize-folder run (low-spec guard).
	MaxSources int
	// MaxDigestChars caps each per-file digest in the report.
	MaxDigestChars int
}

// NewRunner builds the runner with default bounds.
func NewRunner() *Runner { return &Runner{MaxSources: 3, MaxDigestChars: 1200} }

// Available names the runnable routines (also the run_workflow error hint).
func Available() []string { return []string{"summarize-folder", "add-to-summary"} }

// Run executes one routine by name and returns its report text.
// ok=false means nothing ran (unknown name or failed precondition) — never a
// partial silent success. summarize-folder is read-only (safe to retry
// verbatim); add-to-summary queues user-gated proposals, so a retry surfaces
// as visible duplicates in Review rather than silent double-writes.
func (r *Runner) Run(name string, args map[string]string, h tools.Dispatcher) (string, bool) {
	if r == nil {
		return "workflow runner not configured", false
	}
	if args == nil {
		args = map[string]string{}
	}
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "summarize-folder":
		return r.summarizeFolder(h)
	case "add-to-summary":
		return r.addToSummary(h, strings.TrimSpace(args["source"]), strings.TrimSpace(args["topic"]))
	default:
		return fmt.Sprintf("unknown workflow %q (available: %s)", name, strings.Join(Available(), ", ")), false
	}
}

// summarizeFolder is the deterministic multi-file read routine: list
// readable files, read the first window of up to MaxSources, and return
// digests the model turns into propose_summary_edit calls (one per claim).
// It never proposes itself — proposals stay user-gated in the chat turn.
func (r *Runner) summarizeFolder(h Host) (string, bool) {
	files := h.Files()
	if len(files) == 0 {
		return "summarize-folder: no readable files in workspace", false
	}
	max := r.MaxSources
	if max <= 0 {
		max = 3
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Workflow summarize-folder: digesting %d of %d readable file(s).\n", min(len(files), max), len(files))
	read := 0
	for _, f := range files {
		if read >= max {
			break
		}
		res := h.Dispatch(tools.ToolReadSource, map[string]string{"path": f})
		if !res.OK {
			fmt.Fprintf(&b, "\n- %s — unreadable (%s)\n", f, firstLine(res.Content))
			continue
		}
		read++
		fmt.Fprintf(&b, "\n- %s:\n%s\n", f, truncateDigest(res.Content, r.MaxDigestChars))
	}
	b.WriteString("\nNext: draft propose_summary_edit calls from these digests (one per claim, cite path). ")
	b.WriteString("Read full windows via read_source for any claim you use — never propose from a digest alone.")
	return b.String(), true
}

func truncateDigest(s string, max int) string {
	if max <= 0 {
		max = 1200
	}
	s = strings.TrimSpace(s)
	if len(s) > max {
		return s[:max-20] + "\n…[digest truncated]"
	}
	return s
}

func firstLine(s string) string {
	/*
		 * if i := strings.Index(s, "\n"); i >= 0 {
				return strings.TrimSpace(s[:i])
			}
	*/
	if before, _, ok := strings.Cut(s, "\n"); ok {
		return strings.TrimSpace(before)
	}
	return strings.TrimSpace(s)
}

// addToSummary is the deterministic read→propose loop (plan 11 §3): one tool
// call completes what used to take repeated user nagging. It verifies the
// summary exists, reads the source (fuzzy resolution included), and queues
// up to maxProposals insert proposals — one per claim chunk, each cited.
// Every proposal stays pending in Review; nothing is written.
func (r *Runner) addToSummary(h Host, source, topic string) (string, bool) {
	if strings.TrimSpace(source) == "" {
		return "add-to-summary: source required (workspace-relative path)", false
	}
	sum := h.Dispatch(tools.ToolReadSummary, map[string]string{})
	if !sum.OK {
		return "add-to-summary: " + firstLine(sum.Content) + " — open the target document and choose “Make summary” first", false
	}
	got := h.Dispatch(tools.ToolReadSource, map[string]string{"path": source})
	if !got.OK {
		return "add-to-summary: " + got.Content, false
	}
	text := stripReadHeader(got.Content)
	chunks := chunkClaims(text, 1500, 3)
	if len(chunks) == 0 {
		return fmt.Sprintf("add-to-summary: %q has no readable text", source), false
	}
	var b strings.Builder
	head := fmt.Sprintf("Workflow add-to-summary: %q", source)
	if topic != "" {
		head += fmt.Sprintf(" (topic: %s)", topic)
	}
	fmt.Fprintf(&b, "%s → %d proposal(s) queued for Review:\n", head, len(chunks))
	for i, c := range chunks {
		prop := c + "\n(Source: " + source + ")"
		res := h.Dispatch(tools.ToolProposeSummaryEdit, map[string]string{"op": "insert", "text": prop})
		if !res.OK {
			fmt.Fprintf(&b, "\n%d. FAILED: %s\n", i+1, firstLine(res.Content))
			continue
		}
		fmt.Fprintf(&b, "\n%d. queued insert (%.80s…)\n", i+1, strings.TrimSpace(c))
	}
	b.WriteString("\nNothing written — accept, revise, or reject each proposal in Review.")
	return b.String(), true
}

// stripReadHeader drops the `File "…" (title …):` envelope read_source
// prepends, leaving claim text for chunking.
func stripReadHeader(content string) string {
	if i := strings.Index(content, ":\n"); i >= 0 && i < 300 {
		return strings.TrimSpace(content[i+2:])
	}
	return strings.TrimSpace(content)
}

// chunkClaims splits text into ≤size claim chunks on paragraph boundaries,
// capped at max chunks.
func chunkClaims(text string, size, max int) []string {
	var out []string
	var cur strings.Builder
	flush := func() {
		if t := strings.TrimSpace(cur.String()); t != "" {
			out = append(out, t)
		}
		cur.Reset()
	}
	for _, para := range strings.Split(text, "\n\n") {
		para = strings.TrimSpace(para)
		if para == "" {
			continue
		}
		if cur.Len() > 0 && cur.Len()+len(para)+2 > size {
			flush()
			if len(out) >= max {
				return out
			}
		}
		if cur.Len() > 0 {
			cur.WriteString("\n\n")
		}
		cur.WriteString(para)
	}
	flush()
	if len(out) > max {
		return out[:max]
	}
	return out
}
