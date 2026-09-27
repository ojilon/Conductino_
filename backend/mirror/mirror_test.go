package mirror

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestReadSyncsFromSnapshot(t *testing.T) {
	s := New(filepath.Join(t.TempDir(), "summaries"))
	md, log, err := s.Read("sum1", "Hello\n\nWorld")
	if err != nil {
		t.Fatal(err)
	}
	if md != "Hello\n\nWorld" || len(log) != 0 {
		t.Fatalf("sync: %q %v", md, log)
	}
	// Same snapshot → working copy served.
	md2, _, err := s.Read("sum1", "Hello\n\nWorld")
	if err != nil || md2 != md {
		t.Fatalf("stable: %q %v", md2, err)
	}
	// Snapshot moved (user saved) → re-sync.
	md3, _, err := s.Read("sum1", "Hello\n\nBrave World")
	if err != nil || !strings.Contains(md3, "Brave") {
		t.Fatalf("resync: %q %v", md3, err)
	}
	// Empty snapshot keeps the working copy.
	md4, _, err := s.Read("sum1", "")
	if err != nil || md4 != md3 {
		t.Fatalf("empty keeps copy: %q %v", md4, err)
	}
	// Unknown summary, empty snapshot → empty, no error.
	md5, _, err := s.Read("nobody", "")
	if err != nil || md5 != "" {
		t.Fatalf("blank: %q %v", md5, err)
	}
}

func TestApplyOps(t *testing.T) {
	s := New(filepath.Join(t.TempDir(), "summaries"))
	if _, _, err := s.Read("s", "Alpha\n\nBeta"); err != nil {
		t.Fatal(err)
	}
	if ok, _, err := s.Apply("s", "insert", "", "", "Gamma"); err != nil || !ok {
		t.Fatalf("insert: %v %v", ok, err)
	}
	if ok, _, err := s.Apply("s", "modify", "", "Beta", "Beta prime"); err != nil || !ok {
		t.Fatalf("modify: %v %v", ok, err)
	}
	if ok, _, err := s.Apply("s", "delete", "", "Alpha", ""); err != nil || !ok {
		t.Fatalf("delete: %v %v", ok, err)
	}
	md, log, err := s.Read("s", "")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(md, "Alpha") || !strings.Contains(md, "Beta prime") || !strings.Contains(md, "Gamma") {
		t.Fatalf("mirror text:\n%s", md)
	}
	if len(log) != 3 {
		t.Fatalf("log: %+v", log)
	}
	// Modify log keeps the pre-image for reject-restore.
	if log[1].OldText != "Beta" || log[1].Text != "Beta prime" {
		t.Fatalf("modify log: %+v", log[1])
	}
	// Missing span: no failure, honest note, content preserved as insert.
	ok, note, err := s.Apply("s", "modify", "nope", "missing span", "New claim")
	if err != nil || ok {
		t.Fatalf("miss should not apply: %v %q %v", ok, note, err)
	}
	md2, _, _ := s.Read("s", "")
	if !strings.Contains(md2, "New claim") {
		t.Fatalf("miss fallback:\n%s", md2)
	}
	// Unknown op errors.
	if _, _, err := s.Apply("s", "rewrite", "", "", "x"); err == nil {
		t.Fatal("expected unknown-op error")
	}
	// Nil store never panics.
	var nilStore *Store
	if _, _, err := nilStore.Read("s", "x"); err == nil {
		t.Fatal("nil store should error")
	}
}

func TestSanitizeAndSweep(t *testing.T) {
	if sanitize("../../etc/x") != "etcx" {
		t.Fatalf("sanitize: %q", sanitize("../../etc/x"))
	}
	dir := filepath.Join(t.TempDir(), "summaries")
	s := New(dir)
	if _, _, err := s.Read("old", "text"); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-8 * 24 * time.Hour)
	for _, n := range []string{"old.md", "old.json"} {
		_ = os.Chtimes(filepath.Join(dir, n), old, old)
	}
	if err := s.Sweep(7 * 24 * time.Hour); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "old.md")); !os.IsNotExist(err) {
		t.Fatal("sweep should delete old mirrors")
	}
}
