// Package ai is the Go-side model access layer behind the Wails boundary.
//
// The Gemini API key is loaded from the process environment or the git-ignored
// backend/.ai.env file (see resolveAPIKey), so it never crosses the Wails
// boundary into JS. The React UI talks to this package through a thin TS
// provider (frontend/src/services/ai.ts → App.StreamAIRequest → "ai://event").
//
// Layout (Phase 1 split of the former services/ai.go):
//   service.go  — AIService interface, New, Run entry, key resolution
//   gemini.go   — HTTP client against generateContent
//   prompts.go  — operation → prompt templates and result encoding
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
}

// New builds the backend AI service, loading the key once at startup.
// Missing key is NOT fatal: Configured() reports false and each Run sinks an
// error event telling the user exactly where to put the key.
func New() *GeminiService {
	return &GeminiService{
		apiKey: resolveAPIKey(),
		model:  defaultGeminiModel,
		client: &http.Client{Timeout: 90 * time.Second},
	}
}

// ProviderName is shown in the Settings dialog.
func (g *GeminiService) ProviderName() string { return "Gemini (Go backend)" }

// Configured reports whether an API key was found (env or backend/.ai.env).
func (g *GeminiService) Configured() bool { return strings.TrimSpace(g.apiKey) != "" }

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
func readKeyFile(path, name string) string {
	raw, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
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
		val = strings.Trim(val, `"'`)
		return strings.TrimSpace(val)
	}
	return ""
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
		emit(models.AIEvent{Type: "error", Message: "AI key missing — add GEMINI_API_KEY to backend/.ai.env and restart the app."})
		return
	}
	op := models.AIOperation(strings.ToUpper(strings.TrimSpace(req.Operation)))
	if op == models.OpSearch {
		// Browser web search has no backing service behind this boundary;
		// report it instead of fabricating a ranking.
		emit(models.AIEvent{Type: "error", Message: "Web search is not connected — AI Browse cannot run yet."})
		return
	}
	prompt, kind, maxTokens := buildPrompt(op, req)
	if prompt == "" {
		emit(models.AIEvent{Type: "error", Message: fmt.Sprintf("Unknown AI operation %q.", req.Operation)})
		return
	}
	emit(models.AIEvent{Type: "phase", Phase: 0, Label: "Contacting Gemini"})
	text, err := g.generate(ctx, prompt, maxTokens)
	if err != nil {
		emit(models.AIEvent{Type: "error", Message: err.Error()})
		return
	}
	emit(models.AIEvent{Type: "done", Payload: encodeResult(kind, text)})
}
