package list

import (
	"os"
	"path/filepath"
	"testing"
)

func mkTree(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for _, p := range []string{"docs/a.txt", "docs/sub/b.md", "top.pdf"} {
		full := filepath.Join(root, filepath.FromSlash(p))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestTreeShape(t *testing.T) {
	n, f := List(mkTree(t))
	if f != nil {
		t.Fatalf("list failed: %v", f)
	}
	if n.Kind != "folder" || n.Path != "" || n.ID != "." {
		t.Fatalf("bad root: %+v", n)
	}
	if len(n.Children) != 2 { // docs/ + top.pdf (sorted: folder first? no — by name)
		t.Fatalf("want 2 top children, got %d", len(n.Children))
	}
}

func TestTokensAndExt(t *testing.T) {
	n, _ := List(mkTree(t))
	files := Flat(n)
	if len(files) != 3 {
		t.Fatalf("want 3 files, got %d", len(files))
	}
	want := map[string]string{"docs/a.txt": "txt", "docs/sub/b.md": "md", "top.pdf": "pdf"}
	for _, fl := range files {
		ext, ok := want[fl.Path]
		if !ok {
			t.Fatalf("unexpected token %q", fl.Path)
		}
		if fl.Ext != ext {
			t.Fatalf("%s: ext %q want %q", fl.Path, fl.Ext, ext)
		}
		if fl.ID != fl.Path {
			t.Fatalf("%s: id %q must equal path token", fl.Path, fl.ID)
		}
	}
}

func TestMissingRootIsNil(t *testing.T) {
	n, f := List(filepath.Join(t.TempDir(), "nope"))
	if f != nil || n != nil {
		t.Fatalf("missing root must be nil,nil, got %v %v", n, f)
	}
}

func TestEmptyFolder(t *testing.T) {
	root := t.TempDir()
	n, f := List(root)
	if f != nil || n == nil {
		t.Fatalf("empty root must list, got %v %v", n, f)
	}
	if len(n.Children) != 0 {
		t.Fatalf("want 0 children, got %d", len(n.Children))
	}
}
