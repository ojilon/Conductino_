package ai

import (
	"context"
	"os"
	"path/filepath"
	"strings"
)

// ModelBackend is one LLM HTTP endpoint. Gemini uses native REST; Groq and
// OpenRouter use the OpenAI-compatible chat completions shape.
// Plan: docs/plans/05-multi-provider-apis.md §4.1
type ModelBackend interface {
	Name() string
	Configured() bool
	Generate(ctx context.Context, prompt string, maxTokens int) (string, error)
}

// loadAIMode reads AI_MODE from env / .ai.env (single | failover | auto).
// Default auto: failover when a secondary key exists, else single.
func loadAIMode() string {
	m := strings.ToLower(strings.TrimSpace(os.Getenv("AI_MODE")))
	if m == "" {
		m = strings.ToLower(strings.TrimSpace(readKeyFile("backend/.ai.env", "AI_MODE")))
	}
	if m == "" {
		if exe, err := os.Executable(); err == nil {
			m = strings.ToLower(strings.TrimSpace(readKeyFile(filepath.Join(filepath.Dir(exe), ".ai.env"), "AI_MODE")))
		}
	}
	switch m {
	case "single", "failover", "dual", "auto":
		return m
	default:
		return "auto"
	}
}

func loadEnvKey(name string) string {
	if k := strings.TrimSpace(os.Getenv(name)); k != "" {
		return k
	}
	if k := readKeyFile("backend/.ai.env", name); k != "" {
		return k
	}
	if exe, err := os.Executable(); err == nil {
		if k := readKeyFile(filepath.Join(filepath.Dir(exe), ".ai.env"), name); k != "" {
			return k
		}
	}
	return ""
}

// isRateLimitOrUnavailable classifies errors that should trigger failover.
func isRateLimitOrUnavailable(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	for _, needle := range []string{
		"429", "rate limit", "rate-limit", "quota", "resource exhausted",
		"503", "unavailable", "overloaded", "capacity",
	} {
		if strings.Contains(s, needle) {
			return true
		}
	}
	return false
}
