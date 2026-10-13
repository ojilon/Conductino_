package save

import (
	"os"
	"path/filepath"
	"testing"

	"Conductino/backend/model"
	"Conductino/backend/read"
)

func TestRoundTrip(t *testing.T) {
	lib := t.TempDir()
	data := t.TempDir()
	wire, _ := read.ParseText("n/a.md", []byte("# Hello\n\nworld one"))
	abs, f := SaveText(lib, "n/a.md", wire)
	if f != nil {
		t.Fatalf("save failed: %v", f)
	}
	if abs != filepath.Join(lib, "n", "a.md") {
		t.Fatalf("abs %q", abs)
	}
	d, f2 := read.OpenFile(lib, data, "n/a.md")
	if f2 != nil {
		t.Fatalf("reopen failed: %v", f2)
	}
	if d.BlocksJSON != wire {
		t.Fatalf("round trip changed blocks:\n%s\nvs\n%s", d.BlocksJSON, wire)
	}
}

func TestRendersListAndSkipsPage(t *testing.T) {
	lib := t.TempDir()
	wire := `{"blocks":[
		{"id":"b-1","type":"heading","level":2,"segments":[{"text":"T"}]},
		{"id":"b-2","type":"list","segments":[],"listItems":[[{"text":"a"}],[{"text":"b"}]]},
		{"id":"b-3","type":"page","segments":[{"text":""}]}
	]}`
	abs, f := SaveText(lib, "l.md", wire)
	if f != nil {
		t.Fatalf("save failed: %v", f)
	}
	raw, err := os.ReadFile(abs)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != "## T\n\n- a\n- b" {
		t.Fatalf("rendered %q", string(raw))
	}
}

func TestEscapeRejected(t *testing.T) {
	lib := t.TempDir()
	wire := `{"blocks":[]}`
	if _, f := SaveText(lib, "../x.md", wire); f == nil {
		t.Fatal(".. escape must fail")
	}
	outside := filepath.Join(string(filepath.Separator), "x.md")
	if _, f := SaveText(lib, outside, wire); f == nil {
		t.Fatal("absolute outsider must fail")
	}
}

func TestUnsupportedExt(t *testing.T) {
	if _, f := SaveText(t.TempDir(), "a.pdf", `{"blocks":[]}`); f == nil || f.Reason != model.ReasonUnsupported {
		t.Fatalf("want unsupported, got %v", f)
	}
}

func TestBadJSON(t *testing.T) {
	if _, f := SaveText(t.TempDir(), "a.md", "not json"); f == nil || f.Reason != model.ReasonParseError {
		t.Fatalf("want parse_error, got %v", f)
	}
}
