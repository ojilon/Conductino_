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
	opened, err := openDocxFile(path, "summary.docx")
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

// List blocks must marshal segments:[] (never null): a nil slice encodes
// to JSON null, which crashed ReaderMode openFile (`b.segments.map` on
// null) as an uncaught promise rejection with no toast.
func TestListBlocksNeverNullSegments(t *testing.T) {
	xml := `<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body>` +
		`<w:p><w:pPr><w:numPr/></w:pPr><w:r><w:t>Only item</w:t></w:r></w:p>` +
		`</w:body></w:document>`
	blocks, err := parseDocumentXML([]byte(xml), "list.docx")
	if err != nil {
		t.Fatal(err)
	}
	if len(blocks) != 1 || blocks[0].Type != "list" {
		t.Fatalf("blocks=%+v", blocks)
	}
	wire, err := BlocksToJSON(blocks)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(wire, `"segments":null`) {
		t.Fatalf("null segments in wire: %s", wire)
	}
}

func TestParseListsAndTables(t *testing.T) {	xml := `<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body>` +
		`<w:p><w:pPr><w:numPr/></w:pPr><w:r><w:t>First item</w:t></w:r></w:p>` +
		`<w:p><w:pPr><w:numPr/></w:pPr><w:r><w:t>Second item</w:t></w:r></w:p>` +
		`<w:p><w:r><w:t>After list.</w:t></w:r></w:p>` +
		`<w:tbl><w:tr><w:tc><w:p><w:r><w:t>A1</w:t></w:r></w:p></w:tc>` +
		`<w:tc><w:p><w:r><w:t>B1</w:t></w:r></w:p></w:tc></w:tr></w:tbl>` +
		`</w:body></w:document>`
	blocks, err := parseDocumentXML([]byte(xml), "test.docx")
	if err != nil {
		t.Fatal(err)
	}
	if len(blocks) != 3 {
		t.Fatalf("want 3 blocks (list, para, table row), got %d: %+v", len(blocks), blocks)
	}
	if blocks[0].Type != "list" || len(blocks[0].ListItems) != 2 {
		t.Fatalf("list block: %+v", blocks[0])
	}
	if blocks[1].Type != "paragraph" {
		t.Fatalf("para block: %+v", blocks[1])
	}
	joined := ""
	for _, s := range blocks[2].Segments {
		joined += s.Text
	}
	if joined != "A1 | B1" {
		t.Fatalf("table row = %q, want %q", joined, "A1 | B1")
	}
	// Stable IDs: same input twice → same IDs.
	again, err := parseDocumentXML([]byte(xml), "test.docx")
	if err != nil {
		t.Fatal(err)
	}
	for i := range blocks {
		if blocks[i].ID != again[i].ID {
			t.Fatalf("unstable id at %d: %s vs %s", i, blocks[i].ID, again[i].ID)
		}
	}
}
