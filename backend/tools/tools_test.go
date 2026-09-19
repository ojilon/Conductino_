package tools

import (
	"path/filepath"
	"strings"
	"testing"

	"Conductino/backend/extract"
	"Conductino/backend/mirror"
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

func TestResolveBasenameOmitsFolder(t *testing.T) {
	// Users name the file, not the folder: "Summary" must resolve into a
	// subfolder, and bare names must not match across folders ambiguously.
	h := &ToolHost{FS: &suggestFS{
		root: "/tmp/ws",
		files: []string{
			"plant_physiology/Plant_Bioenergetics_and_Metabolism.docx",
			"plant_physiology/Summary.docx",
		},
	}, Docs: fakeDocs{}}
	if got, _ := h.resolveSourcePath("Summary"); got != "plant_physiology/Summary.docx" {
		t.Fatalf("basename stem: %q", got)
	}
	if got, _ := h.resolveSourcePath("Plant_Bioenergetics_and_Metabolism.docx"); got != "plant_physiology/Plant_Bioenergetics_and_Metabolism.docx" {
		t.Fatalf("basename exact: %q", got)
	}
	// Ambiguous basenames suggest instead of guessing.
	h2 := &ToolHost{FS: &suggestFS{
		root:  "/tmp/ws",
		files: []string{"a/Notes.docx", "b/Notes.docx"},
	}, Docs: fakeDocs{}}
	if got, sug := h2.resolveSourcePath("Notes"); got != "" || len(sug) != 2 {
		t.Fatalf("ambiguous: %q %v", got, sug)
	}
	// A path naming a folder still matches in full.
	if got, _ := h.resolveSourcePath("plant_physiology/Summary.docx"); got != "plant_physiology/Summary.docx" {
		t.Fatalf("full path: %q", got)
	}
}

func TestCanPublishGate(t *testing.T) {
	full := &ToolHost{
		FS: &fakeFS{root: "/tmp/ws"}, PrimarySummaryID: "s",
		SummaryPath: "Summary.docx", Mirror: mirror.New(t.TempDir()),
	}
	if !full.CanPublish() {
		t.Fatal("complete host must publish")
	}
	for name, h := range map[string]*ToolHost{
		"nil mirror":   {FS: &fakeFS{root: "/tmp/ws"}, PrimarySummaryID: "s", SummaryPath: "Summary.docx"},
		"no id":        {FS: &fakeFS{root: "/tmp/ws"}, SummaryPath: "Summary.docx", Mirror: mirror.New(t.TempDir())},
		"no path":      {FS: &fakeFS{root: "/tmp/ws"}, PrimarySummaryID: "s", Mirror: mirror.New(t.TempDir())},
		"no fs":        {PrimarySummaryID: "s", SummaryPath: "Summary.docx", Mirror: mirror.New(t.TempDir())},
		"escape path":  {FS: &fakeFS{root: "/tmp/ws"}, PrimarySummaryID: "s", SummaryPath: "../evil.docx", Mirror: mirror.New(t.TempDir())},
	} {
		if h.CanPublish() {
			t.Fatalf("%s must not publish", name)
		}
	}
	var nilHost *ToolHost
	if nilHost.CanPublish() {
		t.Fatal("nil host must not publish")
	}
}

func TestReadSummaryMirrorWorkingCopy(t *testing.T) {
	dir := t.TempDir()
	h := &ToolHost{SummaryText: "Alpha\n\nBeta", PrimarySummaryID: "sum1", Mirror: mirror.New(dir)}
	r := h.Dispatch(ToolReadSummary, nil)
	if !r.OK || !strings.Contains(r.Content, "Alpha") {
		t.Fatalf("snapshot read: %+v", r)
	}
	// Propose mirrors into the working copy…
	p := h.Dispatch(ToolProposeSummaryEdit, map[string]string{"op": "insert", "text": "Gamma claim"})
	if !p.OK || !strings.Contains(p.Content, "Mirror working copy updated") {
		t.Fatalf("propose mirror: %+v", p)
	}
	// …so the next read composes snapshot + applied edit.
	r2 := h.Dispatch(ToolReadSummary, nil)
	if !r2.OK || !strings.Contains(r2.Content, "Gamma claim") || !strings.Contains(r2.Content, "working copy") {
		t.Fatalf("mirror read: %+v", r2)
	}
	if !strings.Contains(r2.Content, "edits applied this session") {
		t.Fatalf("op log missing: %+v", r2)
	}
	// User saves (snapshot moves) → mirror re-syncs, edit log kept.
	h.SummaryText = "Alpha\n\nBeta\n\nUser rewrite"
	r3 := h.Dispatch(ToolReadSummary, nil)
	if !r3.OK || !strings.Contains(r3.Content, "User rewrite") || strings.Contains(r3.Content, "Gamma claim") {
		t.Fatalf("resync: %+v", r3)
	}
}

func TestProposalsAccumulateAcrossCalls(t *testing.T) {
	h := &ToolHost{PrimarySummaryID: "s"}
	for _, text := range []string{"claim one", "claim two", "claim three"} {
		if r := h.Dispatch(ToolProposeSummaryEdit, map[string]string{"op": "insert", "text": text}); !r.OK {
			t.Fatalf("propose: %+v", r)
		}
	}
	got := h.Proposals()
	if len(got) != 3 || got[0].NewText != "claim one" || got[2].NewText != "claim three" {
		t.Fatalf("accumulate: %+v", got)
	}
	if h.LastProposal().NewText != "claim three" {
		t.Fatal("LastProposal must stay the latest")
	}
	var nilHost *ToolHost
	if nilHost.Proposals() != nil {
		t.Fatal("nil host must yield nil")
	}
}

func TestPublishSummaryWritesDocx(t *testing.T) {
	dir := t.TempDir()
	mdir := t.TempDir()
	h := &ToolHost{
		FS: &fakeFS{root: dir}, Docs: fakeDocs{},
		SummaryText: "Alpha claim.", PrimarySummaryID: "sum1",
		SummaryPath: "Summary.docx", Mirror: mirror.New(mdir),
	}
	if r := h.Dispatch(ToolProposeSummaryEdit, map[string]string{"op": "insert", "text": "Beta claim"}); !r.OK {
		t.Fatalf("propose: %+v", r)
	}
	r := h.Dispatch(ToolPublishSummary, nil)
	if !r.OK {
		t.Fatalf("publish: %+v", r)
	}
	if !strings.Contains(r.Content, "Summary.docx") {
		t.Fatalf("report: %+v", r)
	}
	// The .docx on disk re-extracts with snapshot + proposal content.
	d := extract.NewDocuments()
	opened, err := d.OpenFile(filepath.Join(dir, "Summary.docx"), "Summary.docx")
	if err != nil || opened.Reason != "" {
		t.Fatalf("reopen: %+v %v", opened, err)
	}
	if plain := blocksJSONToPlain(opened.BlocksJSON); !strings.Contains(plain, "Alpha") || !strings.Contains(plain, "Beta") {
		t.Fatalf("round trip: %q", plain)
	}
	// Diffs recorded for in-file decoration.
	pubs := h.PublishedDocs()
	if len(pubs) != 1 || pubs[0].Path != "Summary.docx" || len(pubs[0].Diffs) != 1 {
		t.Fatalf("published: %+v", pubs)
	}
	if pubs[0].Diffs[0].Text == "" || pubs[0].Diffs[0].Op != "insert" {
		t.Fatalf("diff: %+v", pubs[0].Diffs)
	}
	// No path → honest error, nothing written.
	h2 := &ToolHost{Mirror: mirror.New(t.TempDir()), PrimarySummaryID: "s", SummaryText: "x"}
	if r := h2.Dispatch(ToolPublishSummary, nil); r.OK {
		t.Fatalf("pathless publish must fail: %+v", r)
	}
	// Second publish with no new ops → still OK, zero diffs.
	r2 := h.Dispatch(ToolPublishSummary, nil)
	if !r2.OK {
		t.Fatalf("republish: %+v", r2)
	}
	if got := h.PublishedDocs(); len(got) != 2 || len(got[1].Diffs) != 0 {
		t.Fatalf("republish diffs: %+v", got)
	}
}

func TestReadSummaryEmptyIsValid(t *testing.T) {
	dir := t.TempDir()
	h := &ToolHost{PrimarySummaryID: "blank", Mirror: mirror.New(dir)}
	r := h.Dispatch(ToolReadSummary, nil)
	if !r.OK || !strings.Contains(r.Content, "empty") {
		t.Fatalf("empty summary must be OK: %+v", r)
	}
	// No id and no snapshot → honest error (nothing to read).
	h2 := &ToolHost{}
	if r := h2.Dispatch(ToolReadSummary, nil); r.OK {
		t.Fatalf("id-less blank should fail: %+v", r)
	}
}
