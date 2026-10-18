package docpipe

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"Conductino/backend/extract"
)

// TestBuildAndWriteBackRoundTrip walks the whole mirror pipeline on a real
// (tiny) .docx: extract → Build → SaveFor → edit one block → WriteBack →
// re-extract and see the edit, then save again with no changes (no-op).
func TestBuildAndWriteBackRoundTrip(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("CONDUCTINO_DATA", tmp) // mirrors + target marker live under here

	root := filepath.Join(tmp, "ws")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	abs := filepath.Join(root, "notes.docx")

	// 1. A real .docx with two paragraphs, written by the extract writer.
	seed := []extract.DocxBlock{
		{ID: "b1", Type: "paragraph", Segments: []extract.DocxSegment{{Text: "Hello world"}}},
		{ID: "b2", Type: "heading", Level: 2, Segments: []extract.DocxSegment{{Text: "Section"}}},
	}
	if err := extract.WriteDOCX(abs, seed); err != nil {
		t.Fatal(err)
	}

	// 2. Extract it the way OpenFile does.
	docs := extract.NewDocuments()
	opened, err := docs.OpenFile(abs, "notes.docx")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(opened.BlocksJSON, "Hello world") {
		t.Fatalf("extraction lost the paragraph: %s", opened.BlocksJSON)
	}

	// 3. Build + save the mirror pair.
	content, meta, err := Build("docx", "notes", "notes.docx", abs, opened.PageCount, opened.BlocksJSON)
	if err != nil {
		t.Fatal(err)
	}
	if meta.Kind != "docx" || !meta.Editable {
		t.Fatalf("meta wrong: kind=%s editable=%t", meta.Kind, meta.Editable)
	}
	if len(meta.Blocks) != len(content.Blocks) {
		t.Fatalf("meta blocks (%d) != content blocks (%d)", len(meta.Blocks), len(content.Blocks))
	}
	if _, err := SaveFor(root, "notes.docx", content, meta); err != nil {
		t.Fatal(err)
	}
	if _, _, ok := LoadFor(root, "notes.docx"); !ok {
		t.Fatal("LoadFor could not read back the mirror pair")
	}

	// 4. Edit ONE block in the mirror (what the frontend canvas sends).
	for i := range content.Blocks {
		if content.Blocks[i].Type == "paragraph" {
			content.Blocks[i].Segments = []MirrorSegment{{Text: "Hello mirror"}}
			break
		}
	}
	raw, err := json.Marshal(content)
	if err != nil {
		t.Fatal(err)
	}

	// 5. WriteBack verifies + saves; exactly one block changed.
	report, err := WriteBack(root, "notes.docx", abs, string(raw))
	if err != nil {
		t.Fatal(err)
	}
	if !report.Written || len(report.Changed) != 1 || len(report.Added) != 0 {
		t.Fatalf("unexpected report: %+v", report)
	}

	// 6. Re-extract: the .docx on disk carries the edit.
	reopened, err := docs.OpenFile(abs, "notes.docx")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(reopened.BlocksJSON, "Hello mirror") {
		t.Fatalf("write-back did not land: %s", reopened.BlocksJSON)
	}

	// 7. Saving unchanged content again is an honest no-op. Rebuild the
	//    whole mirror from the fresh extraction first (IDs are content-
	//    addressed, so they changed together with the text).
	content2, meta2, err := Build("docx", "notes", "notes.docx", abs, reopened.PageCount, reopened.BlocksJSON)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := SaveFor(root, "notes.docx", content2, meta2); err != nil {
		t.Fatal(err)
	}
	raw2, _ := json.Marshal(content2)
	report2, err := WriteBack(root, "notes.docx", abs, string(raw2))
	if err != nil {
		t.Fatal(err)
	}
	if report2.Written {
		t.Fatalf("expected no-op save, got %+v", report2)
	}
	if report2.UnchangedCount != len(content2.Blocks) {
		t.Fatalf("expected all %d blocks verified unchanged, got %+v", len(content2.Blocks), report2)
	}
}

// TestSummaryTargetRoundTrip checks the workspace summary-target marker.
func TestSummaryTargetRoundTrip(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("CONDUCTINO_DATA", tmp)

	if _, ok := LoadTarget(); ok {
		t.Fatal("expected no target before SaveTarget")
	}
	want := SummaryTarget{Root: "/ws/root", RelPath: "notes/Summary.docx", Title: "Summary"}
	if err := SaveTarget(want); err != nil {
		t.Fatal(err)
	}
	got, ok := LoadTarget()
	if !ok {
		t.Fatal("LoadTarget found nothing after SaveTarget")
	}
	if got.RelPath != want.RelPath || got.Root != want.Root || got.Title != want.Title || got.SetAt == 0 {
		t.Fatalf("target round-trip mismatch: %+v", got)
	}
	ClearTarget()
	if _, ok := LoadTarget(); ok {
		t.Fatal("ClearTarget did not remove the marker")
	}
}

// TestWriteBackRejectsPDF keeps the read-only promise explicit.
func TestWriteBackRejectsPDF(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("CONDUCTINO_DATA", tmp)
	root := filepath.Join(tmp, "ws")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	// Fake a pdf mirror pair (no real pdf needed — WriteBack checks kind
	// before touching any file).
	meta := MirrorMeta{Version: Version, Kind: "pdf", SourcePath: "paper.pdf"}
	content := MirrorContent{Version: Version, Kind: "pdf", Title: "paper"}
	if err := SaveFor(root, "paper.pdf", content, meta); err != nil {
		t.Fatal(err)
	}
	if _, err := WriteBack(root, "paper.pdf", filepath.Join(root, "paper.pdf"), `{"blocks":[]}`); err == nil {
		t.Fatal("expected pdf write-back to be rejected")
	}
}
