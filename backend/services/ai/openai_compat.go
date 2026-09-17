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
		model = "llama-3.1-8b-instant"
	}
	return &OpenAICompatBackend{
		name:   "groq",
		apiKey: strings.TrimSpace(apiKey),
		base:   "https://api.groq.com/openai/v1",
		model:  model,
		client: &http.Client{Timeout: 90 * time.Second},
	}
}

func NewOpenRouterBackend(apiKey string) *OpenAICompatBackend {
	model := strings.TrimSpace(loadEnvKey("OPENROUTER_MODEL"))
	if model == "" {
		model = "openrouter/auto"
	}
	return &OpenAICompatBackend{
		name:   "openrouter",
		apiKey: strings.TrimSpace(apiKey),
		base:   "https://openrouter.ai/api/v1",
		model:  model,
		client: &http.Client{Timeout: 90 * time.Second},
	}
}

func (b *OpenAICompatBackend) Name() string      { return b.name }
func (b *OpenAICompatBackend) Configured() bool { return b != nil && b.apiKey != "" }

type oaiChatRequest struct {
	Model       string       `json:"model"`
	Messages    []oaiMessage `json:"messages"`
	Temperature float64      `json:"temperature"`
	MaxTokens   int          `json:"max_tokens,omitempty"`
}

type oaiMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type oaiChatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Code    any    `json:"code"`
	} `json:"error"`
}

func (b *OpenAICompatBackend) Generate(ctx context.Context, prompt string, maxTokens int) (string, error) {
	if !b.Configured() {
		return "", fmt.Errorf("%s API key missing.", b.name)
	}
	if maxTokens <= 0 {
		maxTokens = 1024
	}
	body, err := json.Marshal(oaiChatRequest{
		Model: b.model,
		Messages: []oaiMessage{
			{Role: "user", Content: prompt},
		},
		Temperature: 0.3,
		MaxTokens:   maxTokens,
	})
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
	if len(decoded.Choices) == 0 || strings.TrimSpace(decoded.Choices[0].Message.Content) == "" {
		return "", fmt.Errorf("%s returned an empty answer.", b.name)
	}
	return strings.TrimSpace(decoded.Choices[0].Message.Content), nil
}
