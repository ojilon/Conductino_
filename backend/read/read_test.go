package read

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestJailRejectsEscape(t *testing.T) {
	root := t.TempDir()
	if _, f := Resolve(root, "../secret.txt"); f == nil {
		t.Fatal(".. escape must fail")
	}
	outside := filepath.Join(string(filepath.Separator), "outside.txt")
	if _, f := Resolve(root, outside); f == nil {
		t.Fatal("absolute outsider must fail")
	}
}

func TestResolveHappyPath(t *testing.T) {
	root := t.TempDir()
	p := filepath.Join(root, "notes", "a.txt")
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, f := Resolve(root, "notes/a.txt")
	if f != nil {
		t.Fatalf("resolve failed: %v", f)
	}
	if got != p {
		t.Fatalf("got %q want %q", got, p)
	}
}

func TestParseHeadings(t *testing.T) {
	wire, title := ParseText("notes/a.md", []byte("# Hello\n\nworld"))
	if title != "a" {
		t.Fatalf("title %q", title)
	}
	if !strings.Contains(wire, `"heading"`) || !strings.Contains(wire, "Hello") {
		t.Fatalf("wire missing heading: %s", wire)
	}
}

func TestOpenDumpsJSON(t *testing.T) {
	lib := t.TempDir()
	data := t.TempDir()
	if err := os.WriteFile(filepath.Join(lib, "a.txt"), []byte("# T\n\nbody"), 0o644); err != nil {
		t.Fatal(err)
	}
	d, f := OpenFile(lib, data, "a.txt")
	if f != nil {
		t.Fatalf("open failed: %v", f)
	}
	raw, err := os.ReadFile(filepath.Join(data, "temporary", "extract_dump", "a.txt.json"))
	if err != nil {
		t.Fatalf("dump missing: %v", err)
	}
	if string(raw) != d.Blocks {
		t.Fatal("dump must equal returned blocks")
	}
}

func TestUnsupportedType(t *testing.T) {
	lib := t.TempDir()
	if err := os.WriteFile(filepath.Join(lib, "a.xyz"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, f := OpenFile(lib, t.TempDir(), "a.xyz"); f == nil || f.Reason != ReasonUnsupported {
		t.Fatalf("want unsupported, got %v", f)
	}
}

func TestTooLarge(t *testing.T) {
	lib := t.TempDir()
	big := make([]byte, MaxTextBytes+10)
	for i := range big {
		big[i] = 'a'
	}
	if err := os.WriteFile(filepath.Join(lib, "big.txt"), big, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, f := OpenFile(lib, t.TempDir(), "big.txt"); f == nil || f.Reason != ReasonTooLarge {
		t.Fatalf("want too_large, got %v", f)
	}
}
