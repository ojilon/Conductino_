package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"Conductino/backend/models"
)

const maxToolRounds = 2

// runChatWithTools runs AI_CHAT with an optional tool loop:
// model may emit <tool …/> tags; we execute guarded tools, re-prompt, then finish.
func (g *GeminiService) runChatWithTools(ctx context.Context, req models.AIRequest, host *ToolHost, emit func(models.AIEvent)) {
	prompt, kind, maxTokens := buildPrompt(models.OpChat, req)
	if host != nil && host.FS != nil {
		prompt = prompt + "\n\n" + ToolCatalog()
	}
	if prompt == "" {
		emit(models.AIEvent{Type: "error", Message: "Could not build chat prompt."})
		return
	}

	emit(models.AIEvent{Type: "phase", Phase: 0, Label: "Contacting Gemini"})
	text, err := g.generate(ctx, prompt, maxTokens)
	if err != nil {
		emit(models.AIEvent{Type: "error", Message: err.Error()})
		return
	}

	for round := 0; round < maxToolRounds; round++ {
		calls := ParseToolCalls(text)
		if len(calls) == 0 || host == nil {
			break
		}
		var results []string
		for i, c := range calls {
			if i >= 3 {
				break // hard cap per round
			}
			emit(models.AIEvent{Type: "phase", Phase: 1 + round, Label: fmt.Sprintf("Tool: %s", c.Name)})
			r := host.Dispatch(c.Name, c.Args)
			status := "ok"
			if !r.OK {
				status = "error"
			}
			results = append(results, fmt.Sprintf("### Tool result (%s, %s)\n%s", r.Name, status, r.Content))
		}
		emit(models.AIEvent{Type: "phase", Phase: 2 + round, Label: "Integrating tool results"})
		follow := prompt + "\n\n### Previous model output\n" + text + "\n\n" +
			strings.Join(results, "\n\n") +
			"\n\n### Instruction\nUsing the tool results above, answer the user. Do not invent file contents. " +
			"If you still need a tool, emit at most one more tool tag; otherwise reply in plain prose only."
		text, err = g.generate(ctx, follow, maxTokens)
		if err != nil {
			emit(models.AIEvent{Type: "error", Message: err.Error()})
			return
		}
	}

	clean := StripToolTags(text)
	if clean == "" {
		clean = strings.TrimSpace(text)
	}

	// If propose_summary_edit ran, surface insertion so the UI can propose a DocumentChange.
	if host != nil {
		if prop := strings.TrimSpace(host.LastProposal()); prop != "" {
			emit(models.AIEvent{Type: "done", Payload: encodeChatWithInsertion(clean, prop)})
			return
		}
	}
	emit(models.AIEvent{Type: "done", Payload: encodeResult(kind, clean)})
}

func encodeChatWithInsertion(explanation, insertText string) string {
	res := aiResult{
		Explanation: strings.TrimSpace(explanation),
		Insertion:   &aiInsertion{Text: strings.TrimSpace(insertText), Citation: "(AI draft)"},
	}
	b, err := json.Marshal(res)
	if err != nil {
		return encodeResult(kindExplanation, explanation)
	}
	return string(b)
}
