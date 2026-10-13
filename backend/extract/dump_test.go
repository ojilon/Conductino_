package extract

import (
	"os"
	"path/filepath"
	"testing"

	"Conductino/backend/apppaths"
)

func TestDumpRecreatesMissingDir(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("CONDUCTINO_DATA", tmp)
	t.Setenv("CONDUCTINO_DB", "")

	// Simulate the user deleting the folder by hand: dumps must recreate it.
	dumpDir := apppaths.ExtractDumpDir()
	if err := os.MkdirAll(dumpDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(dumpDir); err != nil {
		t.Fatal(err)
	}

	got, err := DumpExtractJSON("", "notes/my paper.txt", `{"blocks":[]}`)
	if err != nil {
		t.Fatalf("dump must not fail: %v", err)
	}
	if _, err := os.Stat(got); err != nil {
		t.Fatalf("dump file missing: %v", err)
	}
	// Must stay jailed inside the dump dir.
	rel, err := filepath.Rel(dumpDir, got)
	if err != nil || len(rel) >= 3 && (rel == ".." || rel[:3] == ".."+string(filepath.Separator)) {
		t.Fatalf("dump escaped dir: %q", got)
	}
}

func TestDumpSanitizesEvilNames(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("CONDUCTINO_DATA", tmp)
	got, err := DumpExtractJSON("", `../../evil:doc?.txt`, `{"blocks":[]}`)
	if err != nil {
		t.Fatal(err)
	}
	base := filepath.Base(got)
	for _, bad := range []string{"/", "\\", ":"} {
		for _, r := range base {
			if string(r) == bad {
				t.Fatalf("unsanitized char in %q", base)
			}
		}
	}
}
