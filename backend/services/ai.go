// Package services — Go-side services behind the Wails boundary.
//
// AI lives here, not in the frontend: the Gemini API key is loaded from the
// process environment or the git-ignored backend/.ai.env file (see
// resolveAPIKey), so it never crosses the Wails boundary into JS. The React
// UI talks to this service through a thin TS provider
// (frontend/src/services/ai.ts → App.StreamAIRequest → "ai://event").
package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	// NOTE (import fix): the module is named `Conductino` in go.mod, so every
	// internal import must start with `Conductino/...`. `models` holds the
	// shared Go mirror of the frontend domain types
	// (see backend/models/models.go).
	"Conductino/backend/models"
)

// AIEventSink receives one streaming unit of an AI operation.
type AIEventSink func(ev models.AIEvent)

// AIService is the boundary for all model access on the Go side. The key and
// provider selection belong to the implementation — never to callers, and
// never to the frontend.
type AIService interface {
	Run(ctx context.Context, req models.AIRequest, sink AIEventSink)
	ProviderName() string
	// Configured reports whether a usable API key was found at startup.
	// The UI uses it for the Settings status; Run with no key sinks an
	// "error" event instead of calling the network.
	Configured() bool
}

// defaultGeminiModel is the model every operation targets. Change it in one
// place here (or extend GeminiAIService with per-operation models later).
const defaultGeminiModel = "gemini-2.5-flash"

// geminiEndpoint is the generateContent REST path; %s is the model name.
// Auth goes in the x-goog-api-key header, never in the URL (URLs end up in
// logs; headers don't).
const geminiEndpoint = "https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent"

// GeminiAIService is the real provider: every reader operation is served by
// the Gemini API over stdlib net/http (no extra dependencies — the low-spec
// Windows build story in tasks.md §4 stays intact). AI_SEARCH (browser web
// search) is intentionally NOT served: there is no backing search here, so it
// sinks an "error" event and the UI shows an honest failure instead of a
// fabricated ranking.
type GeminiAIService struct {
	apiKey string
	model  string
	client *http.Client
}

// NewAI builds the backend AI service, loading the key once at startup.
// Missing key is NOT fatal: Configured() reports false and each Run sinks an
// error event telling the user exactly where to put the key.
func NewAI() *GeminiAIService {
	return &GeminiAIService{
		apiKey: resolveAPIKey(),
		model:  defaultGeminiModel,
		client: &http.Client{Timeout: 90 * time.Second},
	}
}

// ProviderName is shown in the Settings dialog.
func (g *GeminiAIService) ProviderName() string { return "Gemini (Go backend)" }

// Configured reports whether an API key was found (env or backend/.ai.env).
func (g *GeminiAIService) Configured() bool { return strings.TrimSpace(g.apiKey) != "" }

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

// aiResult is the JSON shape sunk in a "done" event's Payload. Keys mirror
// the TS AIResult (frontend/src/types/domain.ts): exactly one of them is set
// per operation; relatedSources is provider-supplied only (empty here — the
// model has no access to the user's library, and the UI renders an explicit
// "no related sources" state instead of invented papers).
type aiResult struct {
	Explanation    string       `json:"explanation,omitempty"`
	Insertion      *aiInsertion `json:"insertion,omitempty"`
	Revision       string       `json:"revision,omitempty"`
	RelatedSources []aiRelated  `json:"relatedSources,omitempty"`
}

type aiInsertion struct {
	Text     string `json:"text"`
	Citation string `json:"citation"`
}

type aiRelated struct {
	Title string `json:"title"`
	Meta  string `json:"meta"`
}

// Run executes one AI operation, emitting coarse progress ("phase") events
// and exactly one terminal event: "done" with the JSON result, or "error"
// with a human-readable message. Every event echoes req.RequestID so the
// frontend can route concurrent requests on the shared "ai://event" channel.
// Cancellation via ctx stops the wait/stream promptly.
func (g *GeminiAIService) Run(ctx context.Context, req models.AIRequest, sink AIEventSink) {
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

// promptKind selects which aiResult key the model text lands in.
type promptKind string

const (
	kindExplanation promptKind = "explanation"
	kindInsertion   promptKind = "insertion"
	kindRevision    promptKind = "revision"
)

// tokenBudgetExplanation/tokenBudgetShort size the per-call output ceiling.
// Explanations and insertions need headroom for a full paragraph; revisions
// restate the input and stay small. Thinking is disabled (see
// geminiGenerationConfig), so the whole budget is visible text — these
// ceilings are generous, not tight.
const (
	tokenBudgetExplanation = 2048
	tokenBudgetShort       = 1024
)

// buildPrompt translates an AI operation into a model prompt, the result kind
// its answer belongs to, and its token ceiling. It returns "" for operations
// this service does not serve (AI_SEARCH is rejected earlier with its own
// honest message).
func buildPrompt(op models.AIOperation, req models.AIRequest) (string, promptKind, int) {
	sel := strings.TrimSpace(req.SelectionText)
	if sel == "" {
		sel = strings.TrimSpace(req.Selection)
	}
	query := strings.TrimSpace(req.Query)
	switch op {
	case models.OpExplain:
		return "Explain the following passage from a research document concisely, in 2-4 sentences, for a researcher. Always finish your final sentence:\n\n" + sel, kindExplanation, tokenBudgetExplanation
	case models.OpVerify:
		return "Assess the following claim from a research document in 2-4 sentences: is it well-supported on its face, and what caveat (if any) should a researcher keep in mind? Always finish your final sentence. Claim:\n\n" + sel, kindExplanation, tokenBudgetExplanation
	case models.OpExpand:
		return "Go deeper on the following passage from a research document: unpack the mechanism, add relevant quantitative or contextual detail, in one short paragraph. Always finish your final sentence:\n\n" + sel, kindExplanation, tokenBudgetExplanation
	case models.OpSummarize:
		text := sel
		if text == "" {
			text = query
		}
		return "Summarize the following text in 3-5 sentences for a research summary. Always finish your final sentence:\n\n" + text, kindExplanation, tokenBudgetExplanation
	case models.OpMerge:
		return "Draft a 1-2 sentence insertion for a research summary based on the following selected passage. Keep it factual and self-contained. Always finish your final sentence:\n\n" + sel, kindInsertion, tokenBudgetExplanation
	case models.OpRewrite:
		draft := query
		if draft == "" {
			draft = sel
		}
		return "Revise the following draft sentence(s) for clarity and academic tone. Return ONLY the revised text, no commentary:\n\n" + draft, kindRevision, tokenBudgetShort
	default:
		return "", "", 0
	}
}

// encodeResult packs raw model text into the "done" payload JSON the TS
// provider parses into AIResult. Insertions carry an "(AI draft)" citation
// because the model cannot cite the user's library.
func encodeResult(kind promptKind, text string) string {
	res := aiResult{}
	switch kind {
	case kindInsertion:
		res.Insertion = &aiInsertion{Text: strings.TrimSpace(text), Citation: "(AI draft)"}
	case kindRevision:
		res.Revision = strings.TrimSpace(text)
	default:
		res.Explanation = strings.TrimSpace(text)
	}
	wire, err := json.Marshal(res)
	if err != nil {
		fallback, _ := json.Marshal(aiResult{Explanation: text})
		return string(fallback)
	}
	return string(wire)
}

// geminiRequest/geminiResponse mirror the generateContent REST shapes used
// here (contents + generationConfig in, candidates/error out).
type geminiRequest struct {
	Contents         []geminiContent        `json:"contents"`
	GenerationConfig geminiGenerationConfig `json:"generationConfig,omitempty"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiGenerationConfig struct {
	Temperature     float32 `json:"temperature,omitempty"`
	MaxOutputTokens int     `json:"maxOutputTokens,omitempty"`
	// ThinkingConfig disables model reasoning output. Our prompts ask for
	// short factual answers where hidden thinking buys nothing — and thinking
	// tokens come out of the SAME maxOutputTokens budget, so leaving it on
	// is what used to cut answers off mid-sentence (finishReason MAX_TOKENS
	// with no visible cause). Budget 0 = the whole ceiling is answer text.
	ThinkingConfig geminiThinkingConfig `json:"thinkingConfig,omitempty"`
}

type geminiThinkingConfig struct {
	ThinkingBudget int `json:"thinkingBudget,omitempty"`
}

type geminiResponse struct {
	Candidates []geminiCandidate `json:"candidates"`
	Error      *geminiAPIError   `json:"error,omitempty"`
}

type geminiCandidate struct {
	Content geminiContent `json:"content"`
	// FinishReason reports why the model stopped: "STOP" (complete) vs
	// "MAX_TOKENS" (cut off by the ceiling) and others. Captured so a
	// truncated answer is never silently served as a finished one.
	FinishReason string `json:"finishReason,omitempty"`
}

type geminiAPIError struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
}

// generate performs one blocking generateContent call and returns the joined
// response text. Failures are returned as short, key-free errors safe to show
// in the UI (the key travels only in the request header, never in messages).
// A MAX_TOKENS stop is treated as a failure — not a partial success — so the
// UI says the answer was cut off instead of showing a sentence that just ends.
func (g *GeminiAIService) generate(ctx context.Context, prompt string, maxTokens int) (string, error) {
	wire, err := json.Marshal(geminiRequest{
		Contents: []geminiContent{{Parts: []geminiPart{{Text: prompt}}}},
		GenerationConfig: geminiGenerationConfig{
			Temperature:     0.3,
			MaxOutputTokens: maxTokens,
			ThinkingConfig:  geminiThinkingConfig{ThinkingBudget: 0},
		},
	})
	if err != nil {
		return "", fmt.Errorf("AI request failed to encode.")
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf(geminiEndpoint, g.model), bytes.NewReader(wire))
	if err != nil {
		return "", fmt.Errorf("AI request could not start.")
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-goog-api-key", g.apiKey)
	resp, err := g.client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("AI unavailable now.")
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", fmt.Errorf("AI response could not be read.")
	}
	var decoded geminiResponse
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return "", fmt.Errorf("AI response was not understood (status %d).", resp.StatusCode)
	}
	if decoded.Error != nil {
		detail := strings.TrimSpace(decoded.Error.Message)
		if detail == "" {
			detail = fmt.Sprintf("status %d", decoded.Error.Code)
		}
		return "", fmt.Errorf("AI unavailable now: %s", truncateRunes(detail, 220))
	}
	var sb strings.Builder
	cutOff := false
	for _, c := range decoded.Candidates {
		if strings.EqualFold(strings.TrimSpace(c.FinishReason), "MAX_TOKENS") {
			cutOff = true
		}
		for _, p := range c.Content.Parts {
			sb.WriteString(p.Text)
		}
	}
	text := strings.TrimSpace(sb.String())
	if text == "" {
		return "", fmt.Errorf("AI returned an empty response.")
	}
	if cutOff {
		return "", fmt.Errorf("AI response was cut off — try a shorter selection.")
	}
	return text, nil
}

// truncateRunes shortens s to at most n runes so upstream error bodies can be
// surfaced without flooding the UI.
func truncateRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}
