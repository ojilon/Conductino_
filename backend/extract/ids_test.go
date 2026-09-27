package extract

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func blockIDs(t *testing.T, wire string) []string {
	t.Helper()
	var wrap struct {
		Blocks []textBlock `json:"blocks"`
	}
	if err := json.Unmarshal([]byte(wire), &wrap); err != nil {
		t.Fatal(err)
	}
	ids := make([]string, 0, len(wrap.Blocks))
	for _, b := range wrap.Blocks {
		ids = append(ids, b.ID)
	}
	return ids
}

// Reopening an unchanged file must yield identical IDs (issues 7+27:
// highlights, pending changes, and chat anchors survive reopening).
func TestStableBlockIDsAcrossReextract(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "notes.txt")
	body := "# Title\n\nFirst para.\n\nSecond para.\n"
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	a, err := openTextFile(p, "notes.txt")
	if err != nil {
		t.Fatal(err)
	}
	b, err := openTextFile(p, "notes.txt")
	if err != nil {
		t.Fatal(err)
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
	c, err := openTextFile(p, "notes.txt")
	if err != nil {
		t.Fatal(err)
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
	a, err := openTextFile(p, "dup.txt")
	if err != nil {
		t.Fatal(err)
	}
	ids := blockIDs(t, a.BlocksJSON)
	if len(ids) != 2 || ids[0] == ids[1] {
		t.Fatalf("duplicate paragraphs need unique ids: %v", ids)
	}
	b, err := openTextFile(p, "dup.txt")
	if err != nil {
		t.Fatal(err)
	}
	ids2 := blockIDs(t, b.BlocksJSON)
	if ids[0] != ids2[0] || ids[1] != ids2[1] {
		t.Fatalf("unstable dup ids: %v vs %v", ids, ids2)
	}
}

// NOTE: TestExtractCacheRoundTrip lives in services/storage_test.go —
// the cache belongs to storage, not extraction.
