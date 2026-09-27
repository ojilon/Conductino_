package ai

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// Duplicate keys: last occurrence wins (the user's .ai.env had two
// AI_PRIMARY lines; first-wins silently pinned them to Gemini).
func TestReadKeyFileLastWins(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, ".ai.env")
	body := "AI_PRIMARY=gemini\nAI_PRIMARY=groq\n"
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := readKeyFile(p, "AI_PRIMARY"); got != "groq" {
		t.Fatalf("got %q, want groq", got)
	}
}

// Trailing " # comment" must not leak into the value (AI_MODE=auto # ...).
func TestReadKeyFileStripsInlineComment(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, ".ai.env")
	body := "AI_MODE=auto            # single|dual|failover|auto\n"
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := readKeyFile(p, "AI_MODE"); got != "auto" {
		t.Fatalf("got %q, want auto", got)
	}
}

func TestIsModelGone(t *testing.T) {
	gone := []string{
		"groq: The model `llama-3.1-8b-instant` does not exist",
		"model_not_found",
		"429 rate limit (groq): quota exceeded",
	}
	for _, m := range gone {
		if !isModelGone(errors.New(m)) {
			t.Fatalf("should be gone: %s", m)
		}
	}
	stay := []string{
		"groq API key missing.",
		"AI unavailable now (groq).",
		"groq: invalid API key",
	}
	for _, m := range stay {
		if isModelGone(errors.New(m)) {
			t.Fatalf("should not advance chain: %s", m)
		}
	}
}
