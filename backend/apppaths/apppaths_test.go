package apppaths

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDataRootPrefersEnv(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("CONDUCTINO_DATA", tmp)
	t.Setenv("CONDUCTINO_DB", "")
	if got := DataRoot(); got != mustAbs(t, tmp) {
		t.Fatalf("CONDUCTINO_DATA not preferred: %q", got)
	}
	if got := DBPath(); got != filepath.Join(mustAbs(t, tmp), "conductino.db") {
		t.Fatalf("DBPath should live under CONDUCTINO_DATA: %q", got)
	}
}

func TestDBOverrideWinsOverDataRoot(t *testing.T) {
	data := t.TempDir()
	db := filepath.Join(t.TempDir(), "custom.db")
	t.Setenv("CONDUCTINO_DATA", data)
	t.Setenv("CONDUCTINO_DB", db)
	if got := DBPath(); got != mustAbs(t, db) {
		t.Fatalf("CONDUCTINO_DB should win: %q", got)
	}
}

func TestEnsureDataRootCreatesTree(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("CONDUCTINO_DATA", tmp)
	t.Setenv("CONDUCTINO_DB", "")
	if err := EnsureDataRoot(); err != nil {
		t.Fatal(err)
	}
	for _, d := range []string{DataRoot(), WorkDir(), SummariesDir(), SkillsDir(), CacheDir()} {
		if st, err := os.Stat(d); err != nil || !st.IsDir() {
			t.Fatalf("missing dir %s: %v", d, err)
		}
	}
}

func TestSubdirLayout(t *testing.T) {
	tmp := mustAbs(t, t.TempDir())
	t.Setenv("CONDUCTINO_DATA", tmp)
	t.Setenv("CONDUCTINO_DB", "")
	cases := map[string]string{
		"work":       WorkDir(),
		"summaries":  SummariesDir(),
		"skills":     SkillsDir(),
		"cache":      CacheDir(),
		"config":     ConfigPath(),
		"conductino": DBPath(),
	}
	for name, p := range cases {
		if filepath.Dir(p) == "" || p == "" {
			t.Fatalf("%s resolved empty", name)
		}
		rel, err := filepath.Rel(tmp, p)
		if err != nil || rel == ".." || len(rel) >= 3 && rel[:3] == ".."+string(filepath.Separator) {
			t.Fatalf("%s escapes data root: %q", name, p)
		}
	}
}

func mustAbs(t *testing.T, p string) string {
	t.Helper()
	abs, err := filepath.Abs(p)
	if err != nil {
		t.Fatal(err)
	}
	return abs
}
