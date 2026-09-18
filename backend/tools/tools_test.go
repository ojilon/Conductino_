package tools

import (
	"path/filepath"
	"strings"
	"testing"

	"Conductino/backend/models"
)

func TestParseToolCalls(t *testing.T) {
	text := `Let me look.
<tool name="list_workspace"/>
<tool name="read_source" path="notes/a.txt"/>
Done.`
	calls := ParseToolCalls(text)
	if len(calls) != 2 {
		t.Fatalf("want 2 calls, got %d", len(calls))
	}
	if calls[0].Name != ToolListWorkspace {
		t.Errorf("first name: %s", calls[0].Name)
	}
	if calls[1].Args["path"] != "notes/a.txt" {
		t.Errorf("path: %v", calls[1].Args)
	}
}

func TestStripToolTags(t *testing.T) {
	in := "Answer <tool name=\"list_workspace\"/> here."
	out := StripToolTags(in)
	if strings.Contains(out, "<tool") {
		t.Fatalf("tags remain: %q", out)
	}
}

type fakeFS struct {
	root string
}

func (f *fakeFS) Root() string { return f.root }
func (f *fakeFS) Resolve(path string) (string, error) {
	if strings.Contains(path, "..") || strings.HasPrefix(path, "/") {
		return "", errOutside
	}
	return f.root + "/" + path, nil
}
func (f *fakeFS) ListRoot() (*models.FileTreeNode, error) {
	return &models.FileTreeNode{
		ID: ".", Kind: "folder", Label: "ws",
		Children: []models.FileTreeNode{
			{ID: "a.txt", Kind: "file", Label: "a.txt", Ext: "txt", Path: "a.txt"},
		},
	}, nil
}

var errOutside = errString("path escapes workspace")

type errString string

func (e errString) Error() string { return string(e) }

type fakeDocs struct{}

func (fakeDocs) OpenFile(absPath, relKey string) (models.OpenedDocument, error) {
	return models.OpenedDocument{
		Title:      "a",
		BlocksJSON: `{"blocks":[{"id":"1","type":"paragraph","segments":[{"text":"hello tool"}]}]}`,
		Kind:       "text",
	}, nil
}

func TestReadSourceRejectsEscape(t *testing.T) {
	h := &ToolHost{FS: &fakeFS{root: "/tmp/ws"}, Docs: fakeDocs{}}
	r := h.Dispatch(ToolReadSource, map[string]string{"path": "../etc/passwd"})
	if r.OK {
		t.Fatal("expected rejection")
	}
}

func TestListWorkspace(t *testing.T) {
	h := &ToolHost{FS: &fakeFS{root: "/tmp/ws"}}
	r := h.Dispatch(ToolListWorkspace, nil)
	if !r.OK || !strings.Contains(r.Content, "a.txt") {
		t.Fatalf("list: %+v", r)
	}
}

func TestProposeSummaryEdit(t *testing.T) {	h := &ToolHost{}
	r := h.Dispatch(ToolProposeSummaryEdit, map[string]string{"text": "ATP is driven by proton motive force."})
	if !r.OK || h.LastProposal().NewText == "" {
		t.Fatalf("propose: %+v", r)
	}
	if got := h.LastProposal().Op; got != "insert" {
		t.Fatalf("default op = %q, want insert", got)
	}
}

func TestSearchWorkspace(t *testing.T) {
	h := &ToolHost{FS: &fakeFS{root: "/tmp/ws"}, Docs: fakeDocs{}}
	r := h.Dispatch(ToolSearchWorkspace, map[string]string{"query": "hello"})
	if !r.OK || !strings.Contains(r.Content, "a.txt") {
		t.Fatalf("search hit: %+v", r)
	}
	r = h.Dispatch(ToolSearchWorkspace, map[string]string{"query": "zzz-no-such-word"})
	if !r.OK || !strings.Contains(r.Content, "no matches") {
		t.Fatalf("search miss: %+v", r)
	}
	if r := h.Dispatch(ToolSearchWorkspace, map[string]string{"query": "  "}); r.OK {
		t.Fatalf("empty query should fail: %+v", r)
	}
}

func TestProposeSummaryEditOps(t *testing.T) {	h := &ToolHost{}
	// modify without a target must fail (the UI could never anchor it).
	if r := h.Dispatch(ToolProposeSummaryEdit, map[string]string{"op": "modify", "text": "new"}); r.OK {
		t.Fatalf("modify without target should fail: %+v", r)
	}
	if r := h.Dispatch(ToolProposeSummaryEdit, map[string]string{
		"op": "modify", "target": "b1", "text": "better definition",
	}); !r.OK || h.LastProposal().Op != "modify" {
		t.Fatalf("modify: %+v", r)
	}
	if r := h.Dispatch(ToolProposeSummaryEdit, map[string]string{
		"op": "delete", "target": "b2",
	}); !r.OK || h.LastProposal().Op != "delete" {
		t.Fatalf("delete: %+v", r)
	}
	if r := h.Dispatch(ToolProposeSummaryEdit, map[string]string{"op": "rename", "text": "x"}); r.OK {
		t.Fatalf("unknown op should fail: %+v", r)
	}
}

// suggestFS gates Resolve to listed files, so near-misses exercise the
// did-you-mean path instead of the permissive fakeFS.
type suggestFS struct {
	root  string
	files []string
}

func (f *suggestFS) Root() string { return f.root }
func (f *suggestFS) Resolve(path string) (string, error) {
	for _, x := range f.files {
		if x == path {
			return f.root + "/" + path, nil
		}
	}
	return "", errOutside
}
func (f *suggestFS) ListRoot() (*models.FileTreeNode, error) {
	root := &models.FileTreeNode{ID: ".", Kind: "folder", Label: "ws"}
	for _, x := range f.files {
		root.Children = append(root.Children, models.FileTreeNode{
			ID: x, Kind: "file", Label: filepath.Base(x),
			Ext: strings.TrimPrefix(filepath.Ext(x), "."), Path: x,
		})
	}
	return root, nil
}

func TestReadSourceMissingExtension(t *testing.T) {
	h := &ToolHost{FS: &suggestFS{root: "/tmp/ws", files: []string{"Summary.docx"}}, Docs: fakeDocs{}}
	r := h.Dispatch(ToolReadSource, map[string]string{"path": "Summary"})
	if !r.OK || !strings.Contains(r.Content, "Summary.docx") {
		t.Fatalf("stem resolve: %+v", r)
	}
}

func TestReadSourceTypoSuggests(t *testing.T) {
	h := &ToolHost{FS: &suggestFS{
		root:  "/tmp/ws",
		files: []string{"plant_bioenergetics_and_metabolism.docx", "Summary.docx"},
	}, Docs: fakeDocs{}}
	r := h.Dispatch(ToolReadSource, map[string]string{"path": "plant_bioeneergetics_and_metabolism"})
	if r.OK {
		t.Fatalf("typo should not succeed: %+v", r)
	}
	if !strings.Contains(r.Content, "did you mean") ||
		!strings.Contains(r.Content, "plant_bioenergetics_and_metabolism.docx") {
		t.Fatalf("no suggestion: %+v", r)
	}
}
