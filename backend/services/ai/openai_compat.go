package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// OpenAICompatBackend talks to Groq / OpenRouter / any OpenAI-compatible chat API.
type OpenAICompatBackend struct {
	name   string
	apiKey string
	base   string
	model  string
	client *http.Client
}

func NewGroqBackend(apiKey string) *OpenAICompatBackend {
	model := strings.TrimSpace(loadEnvKey("GROQ_MODEL"))
	if model == "" {
		// llama-3.1-8b-instant was retired by Groq (404s) — gpt-oss-20b is
		// the current fast free-tier default. Set GROQ_MODEL to pin another.
		model = "openai/gpt-oss-20b"
	}
	return &OpenAICompatBackend{
		name:   "groq",
		apiKey: strings.TrimSpace(apiKey),
		base:   "https://api.groq.com/openai/v1",
		model:  model,
		client: &http.Client{Timeout: 90 * time.Second},
	}
}

// groqModelFallbacks are tried in order when the configured model is gone
// (retired IDs 404) or rate-limited (per-model quota buckets on Groq).
var groqModelFallbacks = []string{
	"openai/gpt-oss-20b",
	"qwen/qwen3-32b",
	"meta-llama/llama-4-scout-17b-16e-instruct",
}

func NewOpenRouterBackend(apiKey string) *OpenAICompatBackend {
	model := strings.TrimSpace(loadEnvKey("OPENROUTER_MODEL"))
	if model == "" {
		model = "qwen/qwen3-235b-a22b:free"
	}
	return &OpenAICompatBackend{
		name:   "openrouter",
		apiKey: strings.TrimSpace(apiKey),
		base:   "https://openrouter.ai/api/v1",
		model:  model,
		client: &http.Client{Timeout: 90 * time.Second},
	}
}

// openRouterModelFallbacks: the auto router works with any funded key.
var openRouterModelFallbacks = []string{
	"qwen/qwen3-235b-a22b:free",
	"openrouter/auto",
}

// isModelGone classifies "unknown/retired model" errors worth retrying with
// a fallback model ID. Quota (429) is included: providers bucket limits per
// model, so a sibling model often still has headroom.
func isModelGone(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	for _, needle := range []string{
		"does not exist", "model_not_found", "model not found", "not found",
		"no endpoints", "invalid model", "unknown model", "404",
		"tool choice", // model emitted undeclared calls — sibling may behave
		"429", "rate limit", "rate-limit", "quota", "resource exhausted",
		"limit exceeded",
	} {
		if strings.Contains(s, needle) {
			return true
		}
	}
	return false
}

func (b *OpenAICompatBackend) Name() string      { return b.name }
func (b *OpenAICompatBackend) Configured() bool { return b != nil && b.apiKey != "" }

type oaiChatRequest struct {
	Model       string       `json:"model"`
	Messages    []oaiMessage `json:"messages"`
	Temperature float64      `json:"temperature"`
	MaxTokens   int          `json:"max_tokens,omitempty"`
	// Function tools: declaring them stops tool-trained models (gpt-oss)
	// from emitting undeclared calls, which the API rejects with
	// "Tool choice is none, but model called a tool".
	Tools      []oaiTool `json:"tools,omitempty"`
	ToolChoice string    `json:"tool_choice,omitempty"` // "auto" | "none"
}

type oaiTool struct {
	Type     string      `json:"type"` // "function"
	Function oaiFunction `json:"function"`
}

type oaiFunction struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

type oaiMessage struct {
	Role      string        `json:"role"`
	Content   string        `json:"content"`
	ToolCalls []oaiToolCall `json:"tool_calls,omitempty"`
}

type oaiToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type oaiChatResponse struct {
	Choices []struct {
		Message struct {
			Content   string        `json:"content"`
			ToolCalls []oaiToolCall `json:"tool_calls,omitempty"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Code    any    `json:"code"`
	} `json:"error"`
}

// oaiToolSchemas mirrors the ToolHost registry (tools.go) as JSON Schema.
// Descriptions match ToolCatalog so text-fallback models see the same docs.
func oaiToolSchemas() []oaiTool {
	str := func(desc string) map[string]any { return map[string]any{"type": "string", "description": desc} }
	obj := func(props map[string]any, required ...string) map[string]any {
		// Groq validates parameters against the JSON-Schema metaschema and
		// rejects empty `"required": []` — omit it when there is nothing
		// required (list_workspace, read_summary).
		m := map[string]any{"type": "object", "properties": props, "additionalProperties": false}
		if len(required) > 0 {
			m["required"] = required
		}
		return m
	}
	fn := func(name, desc string, params map[string]any) oaiTool {
		return oaiTool{Type: "function", Function: oaiFunction{Name: name, Description: desc, Parameters: params}}
	}
	return []oaiTool{
		fn(ToolListWorkspace, "List the workspace file tree (relative paths).",
			obj(map[string]any{})),
		fn(ToolReadSource, "Read a workspace source file as text.",
			obj(map[string]any{"path": str("Workspace-relative file path, e.g. notes/a.txt")}, "path")),
		fn(ToolReadSummary, "Read the current research summary snapshot.",
			obj(map[string]any{})),
		fn(ToolProposeSummaryEdit, "Draft one summary change for the user to accept. Never append-only: redefine, restructure, or remove.",
			obj(map[string]any{
				"text":   str("New/replacement text (empty for delete)"),
				"op":     str("insert, modify, or delete"),
				"target": str("Block id or quoted span for modify/delete"),
				"oldtext": str("Span being replaced/removed (modify/delete fallback)"),
			}, "text")),
		fn(ToolSearchWorkspace, "Keyword search over readable workspace files; returns quoted snippets.",
			obj(map[string]any{"query": str("Keyword to find")}, "query")),
	}
}

func (b *OpenAICompatBackend) Generate(ctx context.Context, prompt string, maxTokens int) (string, error) {
	if !b.Configured() {
		return "", fmt.Errorf("%s API key missing.", b.name)
	}
	// Configured model first, then live fallbacks — deduped. Only
	// model-gone/quota errors advance the chain; anything else (auth,
	// network, empty answer) returns immediately so we never burn quota
	// blindly across models.
	candidates := []string{b.model}
	var fallbacks []string
	if b.name == "groq" {
		fallbacks = groqModelFallbacks
	} else {
		fallbacks = openRouterModelFallbacks
	}
	for _, m := range fallbacks {
		if m != b.model {
			candidates = append(candidates, m)
		}
	}
	var firstErr error
	for _, model := range candidates {
		text, err := b.generateWithModel(ctx, prompt, maxTokens, model)
		if err == nil {
			return text, nil
		}
		if firstErr == nil {
			firstErr = err
		}
		if !isModelGone(err) {
			return "", err
		}
	}
	return "", firstErr
}

func (b *OpenAICompatBackend) generateWithModel(ctx context.Context, prompt string, maxTokens int, model string) (string, error) {
	if maxTokens <= 0 {
		maxTokens = 1024
	}
	text, err := b.postChat(ctx, prompt, maxTokens, model, true)
	if err != nil && toolsUnsupported(err) {
		// Model/endpoint without function calling (some free routers):
		// retry once as plain text — the XML-tag fallback in the prompt
		// still applies and ParseToolCalls picks up any tags.
		return b.postChat(ctx, prompt, maxTokens, model, false)
	}
	return text, err
}

// toolsUnsupported detects "this model/endpoint can't do function calling"
// so we can retry as plain text instead of failing the turn.
func toolsUnsupported(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	for _, needle := range []string{
		"tool_choice", "tool choice", "function calling", "function_call",
		"does not support tools", "tools not supported", "unsupported parameter",
		"metaschema", "invalid json schema", "invalid schema",
	} {
		if strings.Contains(s, needle) {
			return true
		}
	}
	return false
}

func (b *OpenAICompatBackend) postChat(ctx context.Context, prompt string, maxTokens int, model string, withTools bool) (string, error) {
	req := oaiChatRequest{
		Model: model,
		Messages: []oaiMessage{
			{Role: "user", Content: prompt},
		},
		Temperature: 0.3,
		MaxTokens:   maxTokens,
	}
	if withTools {
		req.Tools = oaiToolSchemas()
		req.ToolChoice = "auto"
	}
	body, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("AI request failed to encode.")
	}
	url := strings.TrimRight(b.base, "/") + "/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("AI request could not start.")
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+b.apiKey)
	if b.name == "openrouter" {
		httpReq.Header.Set("HTTP-Referer", "https://github.com/ojilon/Conductino_")
		httpReq.Header.Set("X-Title", "Conductino")
	}
	resp, err := b.client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("AI unavailable now (%s).", b.name)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", fmt.Errorf("AI response could not be read.")
	}
	var decoded oaiChatResponse
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return "", fmt.Errorf("AI response was not understood (status %d).", resp.StatusCode)
	}
	if resp.StatusCode == 429 {
		msg := "rate limited"
		if decoded.Error != nil && decoded.Error.Message != "" {
			msg = decoded.Error.Message
		}
		return "", fmt.Errorf("429 rate limit (%s): %s", b.name, msg)
	}
	if resp.StatusCode >= 500 {
		return "", fmt.Errorf("503 unavailable (%s, status %d)", b.name, resp.StatusCode)
	}
	if decoded.Error != nil && decoded.Error.Message != "" {
		return "", fmt.Errorf("%s: %s", b.name, truncateRunes(decoded.Error.Message, 200))
	}
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("%s error (status %d)", b.name, resp.StatusCode)
	}
	if len(decoded.Choices) == 0 {
		return "", fmt.Errorf("%s returned an empty answer.", b.name)
	}
	// Native tool_calls translate back to the internal <tool> tag form so
	// the chat loop parses one shape regardless of how the model invoked.
	// Text-embedded tags (fallback models) pass through untouched.
	msg := decoded.Choices[0].Message
	var sb strings.Builder
	sb.WriteString(strings.TrimSpace(msg.Content))
	for _, tc := range msg.ToolCalls {
		sb.WriteString("\n" + toolCallToTag(tc))
	}
	text := strings.TrimSpace(sb.String())
	if text == "" {
		return "", fmt.Errorf("%s returned an empty answer.", b.name)
	}
	return text, nil
}

// toolCallToTag renders one native function call as an internal tool tag.
// Unknown names pass through — Dispatch rejects them honestly downstream.
func toolCallToTag(tc oaiToolCall) string {
	name := strings.ToLower(strings.TrimSpace(tc.Function.Name))
	var args map[string]string
	if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
		args = map[string]string{}
	}
	var sb strings.Builder
	sb.WriteString(`<tool name="` + name + `"`)
	for k, v := range args {
		k = strings.ToLower(strings.TrimSpace(k))
		if k == "" || k == "name" {
			continue
		}
		sb.WriteString(fmt.Sprintf(` %s="%s"`, k, xmlAttrEscape(v)))
	}
	sb.WriteString(`/>`)
	return sb.String()
}

// xmlAttrEscape escapes a tag attribute value (ParseToolCalls unescapes).
func xmlAttrEscape(s string) string {
	r := strings.NewReplacer(`&`, `&amp;`, `"`, `&quot;`, `<`, `&lt;`, `>`, `&gt;`)
	return r.Replace(s)
}
