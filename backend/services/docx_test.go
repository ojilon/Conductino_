package services

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteAndReadDOCXRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "summary.docx")
	blocks := []docxBlock{
		{ID: "h1", Type: "heading", Level: 1, Segments: []docxSegment{{Text: "Chemiosmosis"}}},
		{ID: "p1", Type: "paragraph", Segments: []docxSegment{
			{Text: "Mitchell proposed "},
			{Text: "chemiosmotic", Em: true},
			{Text: " coupling with "},
			{Text: "strong evidence", Strong: true},
			{Text: "."},
		}},
	}
	if err := WriteDOCX(path, blocks); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil || info.Size() < 100 {
		t.Fatalf("file missing or tiny: %v size=%v", err, info)
	}
	opened, err := openDocxFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if opened.Kind != "docx" {
		t.Fatalf("kind=%s", opened.Kind)
	}
	if !strings.Contains(opened.BlocksJSON, "Chemiosmosis") {
		t.Fatalf("missing title in %s", opened.BlocksJSON)
	}
	if !strings.Contains(opened.BlocksJSON, "chemiosmotic") {
		t.Fatalf("missing italic run in %s", opened.BlocksJSON)
	}
}

func TestWriteSummaryDOCXPathContainment(t *testing.T) {
	dir := t.TempDir()
	blocksJSON := `{"blocks":[{"id":"1","type":"paragraph","segments":[{"text":"hello"}]}]}`
	_, err := WriteSummaryDOCXPath(dir, "../escape.docx", blocksJSON)
	if err == nil {
		t.Fatal("expected escape rejection")
	}
	abs, err := WriteSummaryDOCXPath(dir, "summaries/ok.docx", blocksJSON)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(abs, "summaries") {
		t.Fatalf("abs=%s", abs)
	}
}

func TestDefaultSummaryDOCXName(t *testing.T) {
	n := DefaultSummaryDOCXName("My Paper: 2024!")
	if !strings.HasSuffix(n, ".docx") || !strings.Contains(n, "summaries") {
		t.Fatalf("name=%s", n)
	}
}
