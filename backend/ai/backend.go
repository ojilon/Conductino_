package ai

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"Conductino/backend/usage"
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
	m := strings.ToLower(strings.TrimSpace(loadEnvKey("AI_MODE")))
	switch m {
	case "single", "failover", "dual", "auto":
		return m
	default:
		return "auto"
	}
}

// loadAIPrimary reads AI_PRIMARY (gemini | groq | openrouter).
// Empty → gemini when Gemini key exists, else first configured secondary.
func loadAIPrimary() string {
	p := strings.ToLower(strings.TrimSpace(loadEnvKey("AI_PRIMARY")))
	switch p {
	case "gemini", "groq", "openrouter":
		return p
	default:
		return ""
	}
}

// loadEnvKey looks up a KEY from process env, then dotenv files.
// Paths match resolveAPIKey so GROQ_API_KEY in repo-root .ai.env works
// the same way GEMINI_API_KEY does (wails dev runs from the repo root).
func loadEnvKey(name string) string {
	if k := strings.TrimSpace(os.Getenv(name)); k != "" {
		return k
	}
	for _, p := range []string{".ai.env", "backend/.ai.env"} {
		if k := readKeyFile(p, name); k != "" {
			return k
		}
	}
	if exe, err := os.Executable(); err == nil {
		if k := readKeyFile(filepath.Join(filepath.Dir(exe), ".ai.env"), name); k != "" {
			return k
		}
	}
	return ""
}

// isRateLimitOrUnavailable classifies errors that should trigger failover.
// Canonical implementation lives in backend/usage (telemetry owns the error
// taxonomy so parallel providers share it); kept here as a thin alias
// during the migration.
func isRateLimitOrUnavailable(err error) bool {
	return usage.IsRateLimitOrUnavailable(err)
}
