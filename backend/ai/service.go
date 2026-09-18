// Package ai is the network-only model access layer (plan 06).
//
// It depends on backend/tools, backend/extract, backend/usage, and the
// storage interface — never the reverse. Offline work stays testable
// without API keys; verify with:
// go list -deps ./backend/extract ./backend/tools ./backend/usage |
//   findstr /i "net/http wails" (expect no output).
//
// Layout:
//   service.go  — AIService interface, New, Run entry, key resolution
//   backends/   — (plan 05/08) gemini.go, openai_compat.go per-backend files
//   gemini.go   — HTTP client against generateContent
//   prompts.go  — operation → prompt templates and result encoding
//   chat.go     — multi-turn tool loop (calls tools.ToolHost)
//   tools_alias.go — one-release aliases (delete per plan 06 step 6)
package ai

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"Conductino/backend/models"
	"Conductino/backend/usage"
)

// EventSink receives one streaming unit of an AI operation.
type EventSink func(ev models.AIEvent)

// Service is the boundary for all model access on the Go side. The key and
// provider selection belong to the implementation — never to callers, and
// never to the frontend.
type Service interface {
	Run(ctx context.Context, req models.AIRequest, sink EventSink)
	ProviderName() string
	// Configured reports whether a usable API key was found at startup.
	// The UI uses it for the Settings status; Run with no key sinks an
	// "error" event instead of calling the network.
	Configured() bool
	// UsageMeters renders session telemetry (per-provider calls/errors/
	// token estimates/RPM + tool call counts) for the Settings dialog.
	UsageMeters() string
}

// defaultGeminiModel is the model every operation targets. Change it in one
// place here (or extend GeminiService with per-operation models later).
const defaultGeminiModel = "gemini-2.5-flash"

// GeminiService is the real provider: every reader operation is served by
// the Gemini API over stdlib net/http (no extra dependencies — the low-spec
// Windows build story in tasks.md §4 stays intact). AI_SEARCH (browser web
// search) is intentionally NOT served: there is no backing search here, so it
// sinks an "error" event and the UI shows an honest failure instead of a
// fabricated ranking.
type GeminiService struct {
	apiKey string
	model  string
	client *http.Client
	// Optional workspace tools (Phase 5). Nil → chat still works without tools.
	fs   PathResolver
	docs FileOpener
	// Multi-provider (05-multi-provider-apis.md): optional secondary backends.
	secondary []ModelBackend
	mode      string // single | failover | dual | auto
	primary   string // gemini | groq | openrouter (AI_PRIMARY)
}

// New builds the backend AI service, loading the key once at startup.
// Missing key is NOT fatal: Configured() reports false and each Run sinks an
// error event telling the user exactly where to put the key.
func New() *GeminiService {
	g := &GeminiService{
		apiKey:  resolveAPIKey(),
		model:   defaultGeminiModel,
		client:  &http.Client{Timeout: 90 * time.Second},
		mode:    loadAIMode(),
		primary: loadAIPrimary(),
	}
	g.secondary = loadSecondaryBackends()
	return g
}

func loadSecondaryBackends() []ModelBackend {
	var out []ModelBackend
	if k := loadEnvKey("GROQ_API_KEY"); k != "" {
		out = append(out, NewGroqBackend(k))
	}
	if k := loadEnvKey("OPENROUTER_API_KEY"); k != "" {
		out = append(out, NewOpenRouterBackend(k))
	}
	return out
}

// NewWithTools builds Gemini with folder-scoped tools (Resolve + OpenFile).
func NewWithTools(fs PathResolver, docs FileOpener) *GeminiService {
	g := New()
	g.fs = fs
	g.docs = docs
	return g
}

// ProviderName is shown in the Settings dialog.
func (g *GeminiService) ProviderName() string {
	names := []string{}
	if g.ConfiguredPrimary() {
		names = append(names, "Gemini")
	}
	for _, b := range g.secondary {
		if b.Configured() {
			names = append(names, b.Name())
		}
	}
	if len(names) == 0 {
		return "AI (no key)"
	}
	label := strings.Join(names, " + ") + " · mode=" + g.mode
	if g.primary != "" {
		label += " · primary=" + g.primary
	}
	return label
}

// UsageMeters renders session telemetry for the Settings dialog: one line
// per provider plus tool call counts. Empty string when nothing ran yet.
func (g *GeminiService) UsageMeters() string {
	parts := []string{}
	if u := usage.UsageSnapshot(); u != "" {
		parts = append(parts, u)
	}
	if t := toolAuditSummary(); t != "" {
		parts = append(parts, t)
	}
	return strings.Join(parts, "\n")
}

// Name implements ModelBackend for the primary Gemini endpoint.
func (g *GeminiService) Name() string { return "gemini" }

// ConfiguredPrimary is true when a Gemini key is present.
func (g *GeminiService) ConfiguredPrimary() bool { return strings.TrimSpace(g.apiKey) != "" }

// Configured reports whether any usable API key was found.
func (g *GeminiService) Configured() bool {
	if g.ConfiguredPrimary() {
		return true
	}
	for _, b := range g.secondary {
		if b.Configured() {
			return true
		}
	}
	return false
}

// Generate implements ModelBackend by delegating to the Gemini HTTP client.
func (g *GeminiService) Generate(ctx context.Context, prompt string, maxTokens int) (string, error) {
	return g.generate(ctx, prompt, maxTokens)
}

// generateWithFailover tries the preferred backend, then others on failure.
// emitPhase may be nil. It returns the winning backend name for usage
// telemetry ("" when nothing was attempted, e.g. missing keys).
//
// Selection rules (05-multi-provider-apis.md):
//   AI_PRIMARY=groq|openrouter → that secondary first (skip Gemini until it fails).
//   AI_PRIMARY=gemini or empty → Gemini first when keyed.
//   AI_MODE=auto with a secondary key → treat as failover.
//   In failover/auto: ANY primary error tries the next backend (not only 429),
//   so a dead Gemini free tier does not block Groq.
func (g *GeminiService) generateWithFailover(
	ctx context.Context,
	prompt string,
	maxTokens int,
	emitPhase func(label string),
) (text, backend string, err error) {
	mode := g.mode
	if mode == "auto" {
		mode = "single"
		for _, b := range g.secondary {
			if b.Configured() {
				mode = "failover"
				break
			}
		}
	}

	preferSecondary := g.primary == "groq" || g.primary == "openrouter"

	var firstErr error
	tryGemini := func() (string, error) {
		if !g.ConfiguredPrimary() {
			return "", fmt.Errorf("gemini key missing")
		}
		if emitPhase != nil {
			emitPhase("Using gemini…")
		}
		return g.generate(ctx, prompt, maxTokens)
	}
	trySecondary := func(onlyName string) (string, string, error) {
		var localFirst error
		for _, b := range g.secondary {
			if !b.Configured() {
				continue
			}
			if onlyName != "" && b.Name() != onlyName {
				continue
			}
			if emitPhase != nil {
				emitPhase("Using " + b.Name() + "…")
			}
			text, err := b.Generate(ctx, prompt, maxTokens)
			if err == nil {
				return text, b.Name(), nil
			}
			if localFirst == nil {
				localFirst = err
			} else {
				localFirst = fmt.Errorf("%v; %s: %v", localFirst, b.Name(), err)
			}
		}
		if localFirst != nil {
			return "", "", localFirst
		}
		return "", "", fmt.Errorf("no secondary backend configured")
	}

	// Preferred secondary first when AI_PRIMARY says so.
	if preferSecondary {
		text, name, err := trySecondary(g.primary)
		if err == nil {
			return text, name, nil
		}
		firstErr = err
		// Fall through: try other secondaries, then Gemini, unless single-mode.
		if mode != "single" {
			text, name, err = trySecondary("")
			if err == nil {
				return text, name, nil
			}
			if firstErr == nil {
				firstErr = err
			}
			if g.ConfiguredPrimary() {
				if emitPhase != nil {
					emitPhase("Secondary failed — trying gemini…")
				}
				text, err = tryGemini()
				if err == nil {
					return text, "gemini", nil
				}
				firstErr = fmt.Errorf("%v; gemini: %v", firstErr, err)
			}
		}
		if firstErr != nil {
			return "", "", firstErr
		}
		return "", "", fmt.Errorf("AI key missing — add GROQ_API_KEY or GEMINI_API_KEY to .ai.env / backend/.ai.env and restart.")
	}

	// Default: Gemini first, then secondaries.
	if g.ConfiguredPrimary() {
		text, err := tryGemini()
		if err == nil {
			return text, "gemini", nil
		}
		firstErr = err
	// Failover/auto: try secondary on any failure when a secondary exists.
	// Single mode: only continue on rate-limit/unavailable class errors.
	if mode == "single" && !isRateLimitOrUnavailable(err) {
		return "", "gemini", err
	}
		if mode == "single" {
			hasSec := false
			for _, b := range g.secondary {
				if b.Configured() {
					hasSec = true
					break
				}
			}
			if !hasSec {
				return "", "gemini", err
			}
		}
		if emitPhase != nil {
			emitPhase("Primary failed — trying secondary…")
		}
	}

	text, name, err := trySecondary("")
	if err == nil {
		return text, name, nil
	}
	if firstErr == nil {
		firstErr = err
	} else {
		firstErr = fmt.Errorf("%v; %v", firstErr, err)
	}

	if firstErr != nil {
		return "", "", firstErr
	}
	return "", "", fmt.Errorf("AI key missing — add GEMINI_API_KEY or GROQ_API_KEY to .ai.env / backend/.ai.env and restart.")
}

// resolveAPIKey finds the Gemini key without ever hard-coding it. Precedence:
//  1. GEMINI_API_KEY environment variable (CI / shell exports win).
//  2. .ai.env in the working directory, then backend/.ai.env (wails dev runs
//     from the repo root, so the second path is the normal one).
//  3. .ai.env next to the built executable (installed-binary fallback).
func resolveAPIKey() string {
	if k := strings.TrimSpace(os.Getenv("GEMINI_API_KEY")); k != "" {
		return k
	}
	for _, p := range []string{".ai.env", "backend/.ai.env"} {
		if k := readKeyFile(p, "GEMINI_API_KEY"); k != "" {
			return k
		}
	}
	if exe, err := os.Executable(); err == nil {
		if k := readKeyFile(filepath.Join(filepath.Dir(exe), ".ai.env"), "GEMINI_API_KEY"); k != "" {
			return k
		}
	}
	return ""
}

// readKeyFile parses a KEY=VALUE dotenv file and returns the value for name.
// Blank lines and # comments are skipped; surrounding quotes are stripped.
// A missing or unreadable file yields "" — the caller tries the next source.
// Last occurrence wins (standard dotenv behavior), so a user can override an
// earlier line by appending below. Trailing " # comment" is stripped —
// without this, `AI_MODE=auto # comment` would parse as a garbage mode.
func readKeyFile(path, name string) string {
	raw, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	found := ""
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok || strings.TrimSpace(key) != name {
			continue
		}
		val = strings.TrimSpace(val)
		if i := strings.Index(val, " #"); i >= 0 {
			val = strings.TrimSpace(val[:i])
		}
		val = strings.Trim(val, `"'`)
		found = strings.TrimSpace(val)
	}
	return found
}

// Run executes one AI operation, emitting coarse progress ("phase") events
// and exactly one terminal event: "done" with the JSON result, or "error"
// with a human-readable message. Every event echoes req.RequestID so the
// frontend can route concurrent requests on the shared "ai://event" channel.
// Cancellation via ctx stops the wait/stream promptly.
func (g *GeminiService) Run(ctx context.Context, req models.AIRequest, sink EventSink) {
	// emit stamps the request correlation ID on every outgoing event.
	emit := func(ev models.AIEvent) {
		ev.RequestID = req.RequestID
		sink(ev)
	}
	if !g.Configured() {
		emit(models.AIEvent{Type: "error", Message: "AI key missing — add GEMINI_API_KEY or GROQ_API_KEY to backend/.ai.env and restart the app."})
		return
	}
	op := models.AIOperation(strings.ToUpper(strings.TrimSpace(req.Operation)))
	if op == models.OpSearch {
		// Browser web search has no backing service behind this boundary;
		// report it instead of fabricating a ranking.
		emit(models.AIEvent{Type: "error", Message: "Web search is not connected — AI Browse cannot run yet."})
		return
	}

	// Phase 5: chat uses the tool loop when filesystem tools are wired.
	if op == models.OpChat {
		host := &ToolHost{
			FS:               g.fs,
			Docs:             g.docs,
			SummaryText:      req.SummaryContent,
			PrimarySummaryID: req.PrimarySummaryID,
		}
		g.runChatWithTools(ctx, req, host, emit)
		return
	}

	prompt, kind, maxTokens := buildPrompt(op, req)
	if prompt == "" {
		emit(models.AIEvent{Type: "error", Message: fmt.Sprintf("Unknown AI operation %q.", req.Operation)})
		return
	}
	// Repeatable one-shots skip the network on a fresh cache hit (plan 05
	// §3.4). Chat and merges are never cached — they must stay fresh.
	if cached, ok := usage.ExplainCacheGet(op, req); ok {
		emit(models.AIEvent{Type: "done", Payload: encodeResult(kind, cached)})
		return
	}
	emit(models.AIEvent{Type: "phase", Phase: 0, Label: "Contacting AI"})
	text, err := g.generateMetered(ctx, op, prompt, maxTokens, func(label string) {
		emit(models.AIEvent{Type: "phase", Label: label})
	})
	if err != nil {
		emit(models.AIEvent{Type: "error", Message: err.Error()})
		return
	}
	usage.ExplainCachePut(op, req, text)
	emit(models.AIEvent{Type: "done", Payload: encodeResult(kind, text)})
}
