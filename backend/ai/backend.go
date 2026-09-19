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

// keySource reports WHERE loadEnvKey found name — never the value itself.
// Startup diagnostic: when a provider shows Configured()==false despite a key
// in backend/.ai.env, this tells whether the process saw a different cwd
// (relative paths miss), a nearer file shadowing it, or a malformed line.
func keySource(name string) string {
	if strings.TrimSpace(os.Getenv(name)) != "" {
		return "process env"
	}
	for _, p := range []string{".ai.env", "backend/.ai.env"} {
		if readKeyFile(p, name) != "" {
			return p
		}
	}
	if exe, err := os.Executable(); err == nil {
		p := filepath.Join(filepath.Dir(exe), ".ai.env")
		if readKeyFile(p, name) != "" {
			return "exe-dir .ai.env"
		}
	}
	return "missing"
}

// keyEnvFor maps a backend name to the env var holding its key (for
// actionable auth-error messages).
func keyEnvFor(backend string) string {
	switch strings.ToLower(strings.TrimSpace(backend)) {
	case "groq":
		return "GROQ_API_KEY"
	case "openrouter":
		return "OPENROUTER_API_KEY"
	default:
		return "GEMINI_API_KEY"
	}
}

// isRateLimitOrUnavailable classifies errors that should trigger failover.
// Canonical implementation lives in backend/usage (telemetry owns the error
// taxonomy so parallel providers share one taxonomy); kept here as a thin alias
// during the migration.
func isRateLimitOrUnavailable(err error) bool {
	return usage.IsRateLimitOrUnavailable(err)
}

// isAuthError classifies invalid-key/auth errors, which must NOT trigger
// silent failover (a bad key never heals by trying another provider — the
// user has to fix config). Thin alias over the usage taxonomy.
func isAuthError(err error) bool {
	return usage.IsAuthError(err)
}
