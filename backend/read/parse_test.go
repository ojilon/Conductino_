package read

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"Conductino/backend/model"
)

func blockIDs(t *testing.T, wire string) []string {
	t.Helper()
	blocks, err := model.ParseBlocks(wire)
	if err != nil {
		t.Fatal(err)
	}
	ids := make([]string, 0, len(blocks))
	for _, b := range blocks {
		ids = append(ids, b.ID)
	}
	return ids
}

// Reopening an unchanged file must yield identical IDs: highlights, pending
// changes, and chat anchors survive reopening.
func TestStableBlockIDsAcrossReextract(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "notes.txt")
	body := "# Title\n\nFirst para.\n\nSecond para.\n"
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	a, fail := OpenText(p, "notes.txt")
	if fail != nil {
		t.Fatal(fail)
	}
	b, fail := OpenText(p, "notes.txt")
	if fail != nil {
		t.Fatal(fail)
	}
	ida, idb := blockIDs(t, a.BlocksJSON), blockIDs(t, b.BlocksJSON)
	if len(ida) != 3 || len(idb) != 3 {
		t.Fatalf("want 3 blocks, got %v / %v", ida, idb)
	}
	for i := range ida {
		if ida[i] != idb[i] {
			t.Fatalf("unstable id at %d: %s vs %s", i, ida[i], idb[i])
		}
	}
	// Edit the middle paragraph: untouched blocks keep IDs, edited gets new.
	body2 := "# Title\n\nFirst para CHANGED.\n\nSecond para.\n"
	if err := os.WriteFile(p, []byte(body2), 0o644); err != nil {
		t.Fatal(err)
	}
	c, fail := OpenText(p, "notes.txt")
	if fail != nil {
		t.Fatal(fail)
	}
	idc := blockIDs(t, c.BlocksJSON)
	if idc[0] != ida[0] || idc[2] != ida[2] {
		t.Fatalf("untouched blocks moved: %v vs %v", ida, idc)
	}
	if idc[1] == ida[1] {
		t.Fatalf("edited block kept stale id %s", idc[1])
	}
}

// Duplicate identical paragraphs must still get unique, deterministic IDs.
func TestStableBlockIDsDuplicateParagraphs(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "dup.txt")
	if err := os.WriteFile(p, []byte("Same.\n\nSame.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	a, fail := OpenText(p, "dup.txt")
	if fail != nil {
		t.Fatal(fail)
	}
	ids := blockIDs(t, a.BlocksJSON)
	if len(ids) != 2 || ids[0] == ids[1] {
		t.Fatalf("duplicate paragraphs need unique ids: %v", ids)
	}
	b, fail := OpenText(p, "dup.txt")
	if fail != nil {
		t.Fatal(fail)
	}
	ids2 := blockIDs(t, b.BlocksJSON)
	if ids[0] != ids2[0] || ids[1] != ids2[1] {
		t.Fatalf("unstable dup ids: %v vs %v", ids, ids2)
	}
}

func TestBuildContextPack(t *testing.T) {
	wire := `{"blocks":[
		{"id":"h1","type":"heading","level":2,"segments":[{"text":"Overview"}]},
		{"id":"p1","type":"paragraph","segments":[{"text":"Before text."}]},
		{"id":"p2","type":"paragraph","segments":[{"text":"Selected passage here."}]},
		{"id":"p3","type":"paragraph","segments":[{"text":"After text."}]}
	]}`
	pack, err := BuildContextPack(wire, PackOpts{
		Title: "Doc", BlockID: "p2", SelText: "Selected passage here.",
		RangeStart: 0, RangeEnd: 8, WindowBlocks: 1, IncludeOutline: true, Budget: 6000,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"### Selection (block p2, chars 0–8)\nSelected passage here.",
		"### Document\nDoc",
		"### Before selection\nBefore text.",
		"### After selection\nAfter text.",
		"### Outline\n## Overview",
	} {
		if !strings.Contains(pack, want) {
			t.Fatalf("pack missing %q:\n%s", want, pack)
		}
	}
	// Unknown block → no window sections, outline still present.
	pack2, err := BuildContextPack(wire, PackOpts{Title: "Doc", BlockID: "nope", SelText: "x", IncludeOutline: true})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(pack2, "Before selection") || !strings.Contains(pack2, "## Overview") {
		t.Fatalf("window handling:\n%s", pack2)
	}
	// Budget truncation keeps the machine-readable marker.
	small, err := BuildContextPack(wire, PackOpts{Title: "Doc", SelText: strings.Repeat("z", 200), Budget: 100})
	if err != nil {
		t.Fatal(err)
	}
	if len(small) > 120 || !strings.Contains(small, "[…truncated…]") {
		t.Fatalf("budget: %q", small)
	}
	// Bad JSON errors.
	if _, err := BuildContextPack("{bad", PackOpts{}); err == nil {
		t.Fatal("expected JSON error")
	}
}

func TestBlocksFromMarkdownRoundTrip(t *testing.T) {
	md := "# Title\n\nFirst paragraph.\n\n- item one\n- item two\n\n--- page 2 ---\n\nLast paragraph."
	blocks := BlocksFromMarkdown(md, "Summary.docx")
	wire := model.MarshalBlocks(blocks)
	back, _, err := NormalizeBlocksJSON(wire)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"# Title", "First paragraph.", "- item one", "- item two", "Last paragraph."} {
		if !strings.Contains(back, want) {
			t.Fatalf("round trip missing %q:\n%s", want, back)
		}
	}
	if strings.Contains(back, "--- page") {
		t.Fatal("page markers must not survive into blocks")
	}
	// Stable IDs: same input twice → identical ids (anchors survive publish).
	again := BlocksFromMarkdown(md, "Summary.docx")
	if len(again) != len(blocks) {
		t.Fatalf("block counts differ: %d vs %d", len(blocks), len(again))
	}
	for i := range blocks {
		if blocks[i].ID != again[i].ID || blocks[i].ID == "" {
			t.Fatalf("unstable id at %d: %q vs %q", i, blocks[i].ID, again[i].ID)
		}
	}
	if empty := BlocksFromMarkdown("", "x"); len(empty) != 1 {
		t.Fatalf("blank md must yield one empty block, got %d", len(empty))
	}
}

func TestNormalizeBlocksJSON(t *testing.T) {
	wire := `{"blocks":[
		{"id":"h1","type":"heading","level":1,"segments":[{"text":"Chemiosmosis"}]},
		{"id":"p1","type":"paragraph","segments":[{"text":"Mitchell proposed coupling."}]},
		{"id":"pg2","type":"page","segments":[{"text":""}]},
		{"id":"p2","type":"paragraph","segments":[{"text":"Later work confirmed it."}]},
		{"id":"l1","type":"list","segments":[],"listItems":[[{"text":"a"}],[{"text":"b"}]]}
	]}`
	md, pmJSON, err := NormalizeBlocksJSON(wire)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"# Chemiosmosis", "Mitchell proposed coupling.", "--- page 2 ---", "Later work confirmed it.", "- a\n- b"} {
		if !strings.Contains(md, want) {
			t.Fatalf("md missing %q:\n%s", want, md)
		}
	}
	var pm []struct {
		Page    int    `json:"page"`
		BlockID string `json:"blockId"`
	}
	if err := json.Unmarshal([]byte(pmJSON), &pm); err != nil {
		t.Fatal(err)
	}
	if len(pm) != 2 || pm[0].BlockID != "h1" || pm[1].BlockID != "p2" {
		t.Fatalf("page map: %+v", pm)
	}
	if _, _, err := NormalizeBlocksJSON("{bad"); err == nil {
		t.Fatal("expected JSON error")
	}
	if got := PlainToMarkdown("a\n\n\nb\r\n\r\n"); got != "a\n\nb" {
		t.Fatalf("plain: %q", got)
	}
}

// Brand-new 0-byte files must open as blank pages — never failures — so a
// Summary.docx created on disk (or an empty .txt/.md/.pdf) is immediately
// usable by the renderer and the summary edit loop.
func TestEmptyFilesOpenAsBlank(t *testing.T) {
	dir := t.TempDir()
	data := t.TempDir()
	cases := []struct{ name, kind string }{
		{"Summary.docx", "docx"},
		{"notes.txt", "text"},
		{"notes.md", "text"},
		{"scan.pdf", "pdf"},
	}
	for _, c := range cases {
		p := filepath.Join(dir, c.name)
		if err := os.WriteFile(p, []byte{}, 0o644); err != nil {
			t.Fatal(err)
		}
		opened, fail := OpenFile(dir, data, c.name)
		if fail != nil {
			t.Fatalf("%s: unexpected failure: %v", c.name, fail)
		}
		if opened.Kind != c.kind {
			t.Fatalf("%s: kind %q, want %q", c.name, opened.Kind, c.kind)
		}
		blocks, err := model.ParseBlocks(opened.BlocksJSON)
		if err != nil {
			t.Fatalf("%s: bad blocks JSON: %v", c.name, err)
		}
		if len(blocks) != 1 || blocks[0].Type != "paragraph" {
			t.Fatalf("%s: want one empty paragraph, got %+v", c.name, blocks)
		}
		want := "Summary"
		if c.name != "Summary.docx" {
			want = c.name[:len(c.name)-len(filepath.Ext(c.name))]
		}
		if opened.Title != want {
			t.Fatalf("%s: title %q, want %q", c.name, opened.Title, want)
		}
	}
}

// makeTestPDF builds a minimal valid PDF with correct xref offsets: one
// Helvetica text line per page. Hand-rolled so tests need no writer dep.
func makeTestPDF(t *testing.T, pages []string) string {
	t.Helper()
	var body strings.Builder
	offsets := []int{}
	header := "%PDF-1.4\n"
	addObj := func(n int, content string) {
		// xref offsets are absolute from file start — include the header.
		offsets = append(offsets, len(header)+body.Len())
		fmt.Fprintf(&body, "%d 0 obj\n%s\nendobj\n", n, content)
	}
	kids := ""
	for i := range pages {
		kids += fmt.Sprintf("%d 0 R ", 3+i*2)
	}
	fontObj := 3 + len(pages)*2
	addObj(1, "<< /Type /Catalog /Pages 2 0 R >>")
	addObj(2, fmt.Sprintf("<< /Type /Pages /Kids [%s] /Count %d >>", kids, len(pages)))
	for i, text := range pages {
		text = strings.ReplaceAll(strings.ReplaceAll(text, "\\", "\\\\"), "(", "\\(")
		text = strings.ReplaceAll(text, ")", "\\)")
		stream := fmt.Sprintf("BT /F1 24 Tf 100 700 Td (%s) Tj ET", text)
		addObj(3+i*2, fmt.Sprintf("<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Contents %d 0 R /Resources << /Font << /F1 %d 0 R >> >> >>", 4+i*2, fontObj))
		addObj(4+i*2, fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(stream), stream))
	}
	addObj(fontObj, "<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>")
	// startxref is absolute from file start — include the header length.
	xrefPos := len(header) + body.Len()
	nObjs := 3 + len(pages)*2
	var xref strings.Builder
	fmt.Fprintf(&xref, "xref\n0 %d\n", nObjs+1)
	xref.WriteString("0000000000 65535 f \n")
	for _, off := range offsets {
		fmt.Fprintf(&xref, "%010d 00000 n \n", off)
	}
	var out strings.Builder
	out.WriteString(header)
	out.WriteString(body.String())
	out.WriteString(xref.String())
	fmt.Fprintf(&out, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF", nObjs+1, xrefPos)
	p := filepath.Join(t.TempDir(), "test.pdf")
	if err := os.WriteFile(p, []byte(out.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestOpenPDFFile(t *testing.T) {
	p := makeTestPDF(t, []string{"Hello PDF world", "Second page here"})
	opened, fail := OpenPDF(p, "test.pdf")
	if fail != nil {
		t.Fatal(fail)
	}
	if opened.Kind != "pdf" {
		t.Fatalf("kind=%s", opened.Kind)
	}
	if opened.PageCount != 2 {
		t.Fatalf("pages=%d", opened.PageCount)
	}
	if !strings.Contains(opened.BlocksJSON, "Hello PDF world") {
		t.Fatalf("missing p1 text in %s", opened.BlocksJSON)
	}
	if !strings.Contains(opened.BlocksJSON, "Second page here") {
		t.Fatalf("missing p2 text in %s", opened.BlocksJSON)
	}
	if !strings.Contains(opened.BlocksJSON, `"type":"page"`) {
		t.Fatalf("missing page-break blocks in %s", opened.BlocksJSON)
	}
	// Stable IDs across re-extract.
	again, fail := OpenPDF(p, "test.pdf")
	if fail != nil {
		t.Fatal(fail)
	}
	if opened.BlocksJSON != again.BlocksJSON {
		t.Fatalf("unstable re-extract:\n%s\nvs\n%s", opened.BlocksJSON, again.BlocksJSON)
	}
}

func TestOpenPDFFileMissing(t *testing.T) {
	_, fail := OpenPDF(filepath.Join(t.TempDir(), "nope.pdf"), "nope.pdf")
	if fail == nil {
		t.Fatal("expected failure")
	}
	if fail.Reason != model.ReasonNotFound {
		t.Fatalf("reason=%s", fail.Reason)
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
	wire := model.MarshalBlocks(blocks)
	if strings.Contains(wire, `"segments":null`) {
		t.Fatalf("null segments in wire: %s", wire)
	}
}

func TestParseListsAndTables(t *testing.T) {
	xml := `<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body>` +
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
	if joined := blocks[2].Text(); joined != "A1 | B1" {
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
