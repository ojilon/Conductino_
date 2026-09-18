package skills

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseStarterSkill(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("summarize-source.md"))
	if err != nil {
		t.Fatal(err)
	}
	s, err := Parse(raw, "summarize-source.md")
	if err != nil {
		t.Fatal(err)
	}
	if s.Name != "summarize-source" || s.Version != 1 {
		t.Fatalf("identity: %+v", s)
	}
	if len(s.Tools) != 4 || s.Tools[0] != "list_workspace" {
		t.Fatalf("tools: %v", s.Tools)
	}
	if s.Budgets["maxToolCalls"] != 6 || s.Budgets["maxChars"] != 6000 {
		t.Fatalf("budgets: %v", s.Budgets)
	}
	if !strings.Contains(s.Body, "never claim success") {
		t.Fatalf("body lost verification rule")
	}
}

func TestParseRejects(t *testing.T) {
	for name, data := range map[string]string{
		"no frontmatter":   "# just a body\n",
		"unclosed":         "---\nskill: x\n",
		"missing name":     "---\nversion: 1\n---\nbody\n",
		"bad version":      "---\nskill: x\nversion: one\n---\nbody\n",
		"empty body":       "---\nskill: x\nversion: 1\n---\n",
		"bad budgets":      "---\nskill: x\nbudgets: [1]\n---\nbody\n",
	} {
		if _, err := Parse([]byte(data), name); err == nil {
			t.Fatalf("%s: expected error", name)
		}
	}
}

func TestLoadDirAndMatch(t *testing.T) {
	dir := t.TempDir()
	body := "---\nskill: a\nwhen: [AI_MERGE]\ntools: [read_source]\n---\nDo A.\n"
	if err := os.WriteFile(filepath.Join(dir, "a.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "note.txt"), []byte("skip me"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := LoadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Name != "a" {
		t.Fatalf("load: %+v", got)
	}
	if m := Match(got, "ai_merge", ""); len(m) != 1 {
		t.Fatalf("match op: %+v", m)
	}
	if m := Match(got, "AI_CHAT", ""); len(m) != 0 {
		t.Fatalf("match negative: %+v", m)
	}
	ex := Excerpt(got[0], 4)
	if !strings.HasPrefix(ex, "## Skill: a\n") || !strings.HasSuffix(ex, "…[truncated]") {
		t.Fatalf("excerpt: %q", ex)
	}
}
