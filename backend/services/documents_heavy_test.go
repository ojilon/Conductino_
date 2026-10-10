package services

import (
	"os"
	"path/filepath"
	"testing"

	"Conductino/backend/extract"
)

// Both txt paths must agree on the same corpus file (split safety net).
// Legacy side goes through the canonical public entry (extract.Documents.OpenFile
// dispatch), not an internal helper — so the test guards the real serving path.
func TestTextPathsAgree(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "notes.md")
	raw := "# Title\n\nBody para.\n\nBody para.\n"
	if err := os.WriteFile(p, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	a, err := extract.NewDocuments().OpenFile(p, "notes.md")
	if err != nil {
		t.Fatalf("legacy path: %v", err)
	}
	b, err := openTextViaHeavy(p, "notes.md")
	if err != nil {
		t.Fatalf("heavy path: %v", err)
	}
	if a.BlocksJSON != b.BlocksJSON || a.Title != b.Title || a.Kind != b.Kind {
		t.Fatalf("paths disagree:\nlegacy=%s\nheavy =%s", a.BlocksJSON, b.BlocksJSON)
	}
}
