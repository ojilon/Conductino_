package extract

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// Brand-new 0-byte files must open as blank pages — never errors — so a
// Summary.docx created on disk (or an empty .txt/.md/.pdf) is immediately
// usable by the renderer and the summary edit loop.
func TestEmptyFilesOpenAsBlank(t *testing.T) {
	dir := t.TempDir()
	d := NewDocuments()
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
		opened, err := d.OpenFile(p, c.name)
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", c.name, err)
		}
		if opened.Reason != "" {
			t.Fatalf("%s: unexpected reason %q (%s)", c.name, opened.Reason, opened.Detail)
		}
		if opened.Kind != c.kind {
			t.Fatalf("%s: kind %q, want %q", c.name, opened.Kind, c.kind)
		}
		var wrap struct {
			Blocks []struct {
				ID       string `json:"id"`
				Type     string `json:"type"`
				Segments []struct {
					Text string `json:"text"`
				} `json:"segments"`
			} `json:"blocks"`
		}
		if err := json.Unmarshal([]byte(opened.BlocksJSON), &wrap); err != nil {
			t.Fatalf("%s: bad blocks JSON: %v", c.name, err)
		}
		if len(wrap.Blocks) != 1 || wrap.Blocks[0].Type != "paragraph" {
			t.Fatalf("%s: want one empty paragraph, got %+v", c.name, wrap.Blocks)
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
