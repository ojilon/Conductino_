package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"Conductino/backend/models"
)

const (
	// 4 rounds so a full locate → read → retry → propose sequence fits:
	// round 0 discovers (list_workspace), round 1 reads (possibly a
	// did-you-mean miss), round 2 executes the corrected retry, round 3
	// proposes. Budget below caps cost.
	maxToolRounds   = 4
	maxToolsPerTurn = 8 // hard budget across rounds (issue 17)
)

// runChatWithTools runs AI_CHAT with an optional tool loop:
// model may emit <tool …/> tags; we execute guarded tools, re-prompt, then finish.
func (g *GeminiService) runChatWithTools(ctx context.Context, req models.AIRequest, host *ToolHost, emit func(models.AIEvent)) {
	prompt, kind, maxTokens := g.buildPromptFor(models.OpChat, req)
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
	// turnTrace is the chat "thinking" log: one entry per tool dispatch
	// (name + ok + ms). Surfaced in the done payload and persisted on the
	// assistant message, so the user can expand exactly what the loop did.
	var turnTrace []aiToolCall
	// lastResults keeps the latest round's tool outputs past the loop so a
	// proposed-but-unwritten turn can quote the publish failure, if any.
	var lastResults []string
	for round := 0; round < maxToolRounds && toolsUsed < maxToolsPerTurn; round++ {
		calls := ParseToolCalls(text)
		if len(calls) == 0 || host == nil {
			break
		}
		var results []string
		for i, c := range calls {
			if i >= 4 || toolsUsed >= maxToolsPerTurn {
				break // hard cap per round and per turn
			}
			toolsUsed++
			emit(models.AIEvent{Type: "phase", Phase: 1 + round, Label: fmt.Sprintf("Tool: %s", c.Name)})
			started := time.Now()
			r := host.Dispatch(c.Name, c.Args)
			turnTrace = append(turnTrace, aiToolCall{Tool: c.Name, OK: r.OK, Ms: time.Since(started).Milliseconds()})
			auditTool(c.Name, c.Args, r.OK) // issue 17: name + path only, never content
			status := "ok"
			if !r.OK {
				status = "error"
			}
			results = append(results, fmt.Sprintf("### Tool result (%s, %s)\n%s", r.Name, status, r.Content))
		}
		lastResults = results
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

	// Surface EVERY proposal this turn so the UI can materialize one
	// DocumentChange per claim for review (plan 11 §3). Drops empties
	// (e.g. a delete with no text carries op only — still reviewable).
	// Published summaries ride along so the UI reloads each file and
	// decorates the write-through diffs in-file.
	if host != nil {
		var props []proposal
		for _, p := range host.Proposals() {
			if strings.TrimSpace(p.NewText) != "" || p.Op == "delete" {
				props = append(props, p)
			}
		}
		// Single-path guarantee (plan 11 §3): the summary is openly editable
		// by default, so a turn that proposed but never published writes
		// through automatically — review happens in-file, never in cards.
		// Only when the host can actually publish (mapped summary + path).
		published := host.PublishedDocs()
		autoErr := ""
		if len(props) > 0 && len(published) == 0 && host.CanPublish() {
			emit(models.AIEvent{Type: "phase", Phase: 3, Label: "Publishing summary"})
			started := time.Now()
			r := host.Dispatch(ToolPublishSummary, map[string]string{})
			turnTrace = append(turnTrace, aiToolCall{Tool: ToolPublishSummary, OK: r.OK, Ms: time.Since(started).Milliseconds()})
			auditTool(ToolPublishSummary, map[string]string{}, r.OK)
			if r.OK {
				published = host.PublishedDocs()
			} else {
				autoErr = strings.TrimSpace(r.Content)
			}
		}
		var publishedOut []aiPublishedSummary
		for _, pub := range published {
			var diffs []aiPublishedDiff
			for _, d := range pub.Diffs {
				diffs = append(diffs, aiPublishedDiff{Op: d.Op, Target: d.Target, Text: d.Text, OldText: d.OldText, Note: d.Note})
			}
			publishedOut = append(publishedOut, aiPublishedSummary{SummaryID: pub.SummaryID, Path: pub.Path, Diffs: diffs})
		}
		if len(props) > 0 || len(publishedOut) > 0 {
			emit(models.AIEvent{Type: "done", Payload: encodeChatWithProposals(clean, props, turnTrace, publishedOut)})
			return
		}
		// Proposed but nothing written: tell the UI exactly why (no mapped
		// summary, no file path, or a failed publish) so it toasts the fix
		// instead of leaving the user staring at an unchanged file.
		if len(props) > 0 {
			emit(models.AIEvent{Type: "done", Payload: encodeChatWithProposals(clean, props, turnTrace, nil, publishFailure(host, lastResults, autoErr))})
			return
		}
	}
	emit(models.AIEvent{Type: "done", Payload: encodeResult(kind, clean)})
}

func encodeChatWithProposals(explanation string, props []proposal, trace []aiToolCall, published []aiPublishedSummary, publishErr ...string) string {
	res := aiResult{Explanation: strings.TrimSpace(explanation), ToolTrace: trace, Published: published}
	if len(published) == 0 && len(props) > 0 && len(publishErr) > 0 && strings.TrimSpace(publishErr[0]) != "" {
		res.PublishError = strings.TrimSpace(publishErr[0])
	}
	for _, prop := range props {
		op := prop.Op
		if op == "" {
			op = "insert"
		}
		res.Proposals = append(res.Proposals, aiProposal{
			Op:                op,
			TargetBlockID:     strings.TrimSpace(prop.Target),
			OldContent:        strings.TrimSpace(prop.OldText),
			NewContent:        strings.TrimSpace(prop.NewText),
			HighlightFragment: strings.TrimSpace(prop.Highlight),
		})
		// Legacy insertion mirror for older frontends (first insert only).
		if op == "insert" && res.Insertion == nil {
			res.Insertion = &aiInsertion{Text: strings.TrimSpace(prop.NewText), Citation: "(AI draft)"}
		}
	}
	b, err := json.Marshal(res)
	if err != nil {
		return encodeResult(kindExplanation, explanation)
	}
	return string(b)
}

func encodeChatWithProposal(explanation string, prop proposal) string {
	return encodeChatWithProposals(explanation, []proposal{prop}, nil, nil)
}

// publishFailure explains a proposed-but-unwritten turn for the user toast.
func publishFailure(host *ToolHost, results []string, autoErr string) string {
	if host == nil {
		return "publishing unavailable in this turn"
	}
	if strings.TrimSpace(host.PrimarySummaryID) == "" {
		return "No workspace summary — open the target document and choose “Make summary”, then ask again."
	}
	if strings.TrimSpace(host.SummaryPath) == "" {
		return "Summary has no file path — save it as a .docx under the workspace (Save DOCX), then ask again."
	}
	if autoErr != "" {
		return "Publish failed: " + autoErr
	}
	for _, r := range results {
		const prefix = "### Tool result (publish_summary, error)\n"
		if strings.HasPrefix(r, prefix) {
			if msg := strings.TrimSpace(strings.TrimPrefix(r, prefix)); msg != "" {
				return "Publish failed: " + msg
			}
			return "Publish failed without detail — retry once, then report."
		}
	}
	return "Edits proposed but not written — ask again or publish explicitly."
}
