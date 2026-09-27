package workflows

import (
	"strings"
	"testing"

	"Conductino/backend/tools"
)

type fakeHost struct {
	files []string
	calls int
	// summary behaves like read_summary; sourceText feeds read_source.
	summary    string
	sourceText string
	proposed   []string
}

func (f *fakeHost) Dispatch(name string, args map[string]string) tools.ToolResult {
	f.calls++
	switch name {
	case tools.ToolReadSummary:
		if f.summary == "" {
			return tools.ToolResult{Name: name, OK: false, Content: "no summary snapshot available"}
		}
		return tools.ToolResult{Name: name, OK: true, Content: f.summary}
	case tools.ToolReadSource:
		return tools.ToolResult{Name: name, OK: true, Content: "File \"x\":\n" + f.sourceText}
	case tools.ToolProposeSummaryEdit:
		f.proposed = append(f.proposed, args["text"])
		return tools.ToolResult{Name: name, OK: true, Content: "Queued"}
	}
	return tools.ToolResult{Name: name, OK: false, Content: "unknown"}
}

func (f *fakeHost) Files() []string { return f.files }

func TestSummarizeFolderDigests(t *testing.T) {
	r := NewRunner()
	h := &fakeHost{files: []string{"a.txt", "b.docx", "c.pdf", "d.md", "e.txt"}}
	out, ok := r.summarizeFolder(h)
	if !ok {
		t.Fatalf("should run: %s", out)
	}
	if h.calls != 3 {
		t.Fatalf("budget: want 3 reads, got %d", h.calls)
	}
	for _, want := range []string{"a.txt", "b.docx", "c.pdf", "propose_summary_edit"} {
		if !strings.Contains(out, want) {
			t.Fatalf("report missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "d.md") {
		t.Fatalf("report must stop at MaxSources:\n%s", out)
	}
}

func TestRunUnknownAndEmpty(t *testing.T) {
	r := NewRunner()
	if _, ok := r.Run("nope", nil, nil); ok {
		t.Fatal("unknown workflow must fail")
	}
	h := &fakeHost{}
	if _, ok := r.Run("summarize-folder", nil, h); ok {
		t.Fatal("empty workspace must fail")
	}
	var nilRunner *Runner
	if _, ok := nilRunner.Run("summarize-folder", nil, h); ok {
		t.Fatal("nil runner must fail")
	}
}

func TestAddToSummaryLoop(t *testing.T) {
	r := NewRunner()
	h := &fakeHost{
		summary:    "Existing summary.",
		sourceText: "Claim A is important.\n\nClaim B follows from A.\n\nClaim C concludes.",
	}
	out, ok := r.Run("add-to-summary", map[string]string{"source": "paper.docx", "topic": "intro"}, h)
	if !ok {
		t.Fatalf("should run: %s", out)
	}
	if len(h.proposed) != 1 {
		// 3 short paragraphs fit one 1500-char chunk.
		t.Fatalf("want 1 proposal, got %d: %v", len(h.proposed), h.proposed)
	}
	if !strings.Contains(h.proposed[0], "Claim A") || !strings.Contains(h.proposed[0], "(Source: paper.docx)") {
		t.Fatalf("proposal content: %q", h.proposed[0])
	}
	if !strings.Contains(out, "queued for Review") || !strings.Contains(out, "Nothing written") {
		t.Fatalf("report:\n%s", out)
	}
}

func TestAddToSummaryPreconditions(t *testing.T) {
	r := NewRunner()
	// No summary → honest error, nothing proposed.
	h := &fakeHost{sourceText: "text"}
	if out, ok := r.Run("add-to-summary", map[string]string{"source": "x.docx"}, h); ok {
		t.Fatalf("must fail without summary: %s", out)
	} else if len(h.proposed) != 0 {
		t.Fatal("must not propose without summary")
	}
	// No source → honest error.
	h2 := &fakeHost{summary: "s"}
	if out, ok := r.Run("add-to-summary", map[string]string{}, h2); ok {
		t.Fatalf("must fail without source: %s", out)
	}
}

func TestChunkClaims(t *testing.T) {
	got := chunkClaims("a\n\nb\n\nc", 3, 2)
	if len(got) != 2 {
		t.Fatalf("chunks: %q", got)
	}
	if len(chunkClaims("", 10, 3)) != 0 {
		t.Fatal("empty text → no chunks")
	}
}

func TestRunnerSatisfiesToolInterface(t *testing.T) {
	// Compile-time proof that *Runner plugs into tools.ToolHost.Workflow.
	var _ tools.WorkflowRunner = NewRunner()
}
