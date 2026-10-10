package services

import (
	"os"
	"path/filepath"
	"testing"
)

// Both txt paths must agree on the same corpus file (split safety net).
func TestTextPathsAgree(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "notes.md")
	raw := "# Title\n\nBody para.\n\nBody para.\n"
	if err := os.WriteFile(p, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	a, err := openTextFile(p, "notes.md")
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
