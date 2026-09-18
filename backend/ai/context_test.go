package ai

import "testing"

func TestTruncatePackUnderBudget(t *testing.T) {
	s := "hello"
	if TruncatePack(s, 100) != s {
		t.Fatal("should not truncate")
	}
}

func TestTruncatePackOverBudget(t *testing.T) {
	s := string(make([]byte, 100))
	for i := range s {
		s = s[:i] + "a" + s[i+1:]
	}
	out := TruncatePack(s, 50)
	if len(out) < 50 {
		t.Fatalf("unexpected short: %d", len(out))
	}
	if !contains(out, "truncated") {
		t.Fatal("expected truncation marker")
	}
}

func TestFormatContextSections(t *testing.T) {
	out := FormatContextSections(map[string]string{
		"Selection": "sel",
		"Outline":   "h1",
		"Empty":     "  ",
	}, []string{"Selection", "Empty", "Outline"})
	if out != "### Selection\nsel\n\n### Outline\nh1" {
		t.Fatalf("got %q", out)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || indexOf(s, sub) >= 0)
}
func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
