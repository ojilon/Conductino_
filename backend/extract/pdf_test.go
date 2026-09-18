package services

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"Conductino/backend/models"
)

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

func TestOpenPdfFile(t *testing.T) {
	p := makeTestPDF(t, []string{"Hello PDF world", "Second page here"})
	opened, err := openPdfFile(p, "test.pdf")
	if err != nil {
		t.Fatal(err)
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
	again, err := openPdfFile(p, "test.pdf")
	if err != nil {
		t.Fatal(err)
	}
	if opened.BlocksJSON != again.BlocksJSON {
		t.Fatalf("unstable re-extract:\n%s\nvs\n%s", opened.BlocksJSON, again.BlocksJSON)
	}
}

func TestOpenPdfFileMissing(t *testing.T) {
	_, err := openPdfFile(filepath.Join(t.TempDir(), "nope.pdf"), "nope.pdf")
	if err == nil {
		t.Fatal("expected error")
	}
	if reason, _ := ReasonOf(err); reason != models.ReasonNotFound {
		t.Fatalf("reason=%s", reason)
	}
}
