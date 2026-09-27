package usage

import (
	"errors"
	"testing"
)

// Auth failures must classify as auth (not rate-limit): a bad key never
// heals by failing over, and conflating the two hides a config fault.
func TestAuthVsRateLimitSplit(t *testing.T) {
	auth := []string{
		"groq: invalid API key — check GROQ_API_KEY.",
		"openrouter: invalid API key — check OPENROUTER_API_KEY.",
		"gemini: invalid API key — check GEMINI_API_KEY.",
		"Incorrect API key provided",
		"401 Unauthorized",
		"API key not valid. Please pass a valid API key.",
		"permission denied",
	}
	for _, m := range auth {
		if !IsAuthError(errors.New(m)) {
			t.Fatalf("should be auth: %s", m)
		}
		if IsRateLimitOrUnavailable(errors.New(m)) {
			t.Fatalf("auth must not trigger failover class: %s", m)
		}
	}
	transient := []string{
		"429 rate limit (groq).",
		"429 rate limit (gemini).",
		"503 unavailable (openrouter, status 503)",
		"AI unavailable now.",
		"quota exceeded",
	}
	for _, m := range transient {
		if IsAuthError(errors.New(m)) {
			t.Fatalf("transient must not classify as auth: %s", m)
		}
		if !IsRateLimitOrUnavailable(errors.New(m)) {
			t.Fatalf("should trigger failover class: %s", m)
		}
	}
}
