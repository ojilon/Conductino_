package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// geminiEndpoint is the generateContent REST path; %s is the model name.
// Auth goes in the x-goog-api-key header, never in the URL (URLs end up in
// logs; headers don't).
const geminiEndpoint = "https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent"

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
func (g *GeminiService) generate(ctx context.Context, prompt string, maxTokens int) (string, error) {
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
	if resp.StatusCode == 429 {
		detail := "rate limited"
		if decoded.Error != nil && strings.TrimSpace(decoded.Error.Message) != "" {
			detail = strings.TrimSpace(decoded.Error.Message)
		}
		return "", fmt.Errorf("429 rate limit (gemini): %s", truncateRunes(detail, 200))
	}
	if resp.StatusCode >= 500 {
		return "", fmt.Errorf("503 unavailable (gemini, status %d)", resp.StatusCode)
	}
	if decoded.Error != nil {
		detail := strings.TrimSpace(decoded.Error.Message)
		if detail == "" {
			detail = fmt.Sprintf("status %d", decoded.Error.Code)
		}
		low := strings.ToLower(detail)
		if strings.Contains(low, "quota") || strings.Contains(low, "rate") || strings.Contains(low, "resource exhausted") {
			return "", fmt.Errorf("429 rate limit (gemini): %s", truncateRunes(detail, 200))
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
