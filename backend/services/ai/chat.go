package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"Conductino/backend/models"
)

const (
	maxToolRounds  = 2
	maxToolsPerTurn = 6 // hard budget across rounds (issue 17)
)

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

	emit(models.AIEvent{Type: "phase", Phase: 0, Label: "Contacting AI"})
	text, err := g.generateMetered(ctx, models.OpChat, prompt, maxTokens, func(label string) {
		emit(models.AIEvent{Type: "phase", Label: label})
	})
	if err != nil {
		emit(models.AIEvent{Type: "error", Message: err.Error()})
		return
	}

	toolsUsed := 0
	for round := 0; round < maxToolRounds && toolsUsed < maxToolsPerTurn; round++ {
		calls := ParseToolCalls(text)
		if len(calls) == 0 || host == nil {
			break
		}
		var results []string
		for i, c := range calls {
			if i >= 3 || toolsUsed >= maxToolsPerTurn {
				break // hard cap per round and per turn
			}
			toolsUsed++
			emit(models.AIEvent{Type: "phase", Phase: 1 + round, Label: fmt.Sprintf("Tool: %s", c.Name)})
			r := host.Dispatch(c.Name, c.Args)
			auditTool(c.Name, c.Args, r.OK) // issue 17: name + path only, never content
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
		text, err = g.generateMetered(ctx, models.OpChat, follow, maxTokens, func(label string) {
			emit(models.AIEvent{Type: "phase", Label: label})
		})
		if err != nil {
			emit(models.AIEvent{Type: "error", Message: err.Error()})
			return
		}
	}

	clean := StripToolTags(text)
	if clean == "" {
		clean = strings.TrimSpace(text)
	}

	// If propose_summary_edit ran, surface the structured proposal so the UI
	// can materialize a DocumentChange (insert/modify/delete) for review.
	if host != nil {
		if prop := host.LastProposal(); strings.TrimSpace(prop.NewText) != "" || prop.Op == "delete" {
			emit(models.AIEvent{Type: "done", Payload: encodeChatWithProposal(clean, prop)})
			return
		}
	}
	emit(models.AIEvent{Type: "done", Payload: encodeResult(kind, clean)})
}

func encodeChatWithProposal(explanation string, prop proposal) string {
	op := prop.Op
	if op == "" {
		op = "insert"
	}
	res := aiResult{
		Explanation: strings.TrimSpace(explanation),
		Proposals: []aiProposal{{
			Op:                op,
			TargetBlockID:     strings.TrimSpace(prop.Target),
			OldContent:        strings.TrimSpace(prop.OldText),
			NewContent:        strings.TrimSpace(prop.NewText),
			HighlightFragment: strings.TrimSpace(prop.Highlight),
		}},
	}
	// Legacy insertion mirror for older frontends (insert path only).
	if op == "insert" {
		res.Insertion = &aiInsertion{Text: strings.TrimSpace(prop.NewText), Citation: "(AI draft)"}
	}
	b, err := json.Marshal(res)
	if err != nil {
		return encodeResult(kindExplanation, explanation)
	}
	return string(b)
}
