package ai

import (
	"encoding/json"
	"fmt"
	"strings"

	"Conductino/backend/models"
	"Conductino/backend/skills"
)

// aiResult is the JSON shape sunk in a "done" event's Payload. Keys mirror
// the TS AIResult (frontend/src/types/domain.ts): explanation always rides
// along; proposals carry structured summary edits (insert/modify/delete)
// while legacy insertion mirrors the insert path for older frontends.
type aiResult struct {
	Explanation    string       `json:"explanation,omitempty"`
	Insertion      *aiInsertion `json:"insertion,omitempty"`
	Revision       string       `json:"revision,omitempty"`
	RelatedSources []aiRelated  `json:"relatedSources,omitempty"`
	Proposals      []aiProposal `json:"proposals,omitempty"`
	// ToolTrace carries the turn's tool loop for the chat "thinking" view.
	ToolTrace []aiToolCall `json:"toolTrace,omitempty"`
	// Published carries summary files written this turn (publish_summary):
	// the UI reloads each and decorates the diffs in-file.
	Published []aiPublishedSummary `json:"publishedSummaries,omitempty"`
	// PublishError explains proposed-but-unwritten turns for the user toast.
	PublishError string `json:"publishError,omitempty"`
}

// aiProposal mirrors TS AIProposal: full rewrite power, not append-only.
type aiProposal struct {
	Op                string `json:"op"` // insert | modify | delete
	TargetBlockID     string `json:"targetBlockId,omitempty"`
	OldContent        string `json:"oldContent,omitempty"`
	NewContent        string `json:"newContent,omitempty"`
	HighlightFragment string `json:"highlightFragment,omitempty"`
}

type aiInsertion struct {
	Text     string `json:"text"`
	Citation string `json:"citation"`
}

// aiToolCall is one entry of the chat "thinking" log: what the tool loop
// ran, whether it worked, and how long it took. Names + redacted status
// only — never file content, prompts, or proposal text (plan 02 §5).
type aiToolCall struct {
	Tool string `json:"tool"`
	OK   bool   `json:"ok"`
	Ms   int64  `json:"ms"`
}

// aiPublishedDiff is one write-through edit for in-file diff decoration.
type aiPublishedDiff struct {
	Op      string `json:"op"`
	Target  string `json:"target,omitempty"`
	Text    string `json:"text,omitempty"`
	OldText string `json:"oldText,omitempty"`
	Note    string `json:"note,omitempty"`
}

// aiPublishedSummary tells the UI a summary file changed on disk this turn:
// reload it and decorate the diffs in-file.
type aiPublishedSummary struct {
	SummaryID string            `json:"summaryId"`
	Path      string            `json:"path"`
	Diffs     []aiPublishedDiff `json:"diffs,omitempty"`
}

type aiRelated struct {
	Title string `json:"title"`
	Meta  string `json:"meta"`
}

// promptKind selects which aiResult key the model text lands in.
type promptKind string

const (
	kindExplanation promptKind = "explanation"
	kindInsertion   promptKind = "insertion"
	kindRevision    promptKind = "revision"
)

const (
	tokenBudgetExplanation = 2048
	tokenBudgetShort       = 1024
	tokenBudgetChat        = 2048
)

func selectionText(req models.AIRequest) string {
	sel := strings.TrimSpace(req.SelectionText)
	if sel == "" {
		sel = strings.TrimSpace(req.Selection)
	}
	return sel
}

// withContext appends custom prompt + document context pack + summary snapshot.
func withContext(base string, req models.AIRequest) string {
	var b strings.Builder
	b.WriteString(base)
	if cp := strings.TrimSpace(req.CustomPrompt); cp != "" {
		b.WriteString("\n\n### User instruction\n")
		b.WriteString(cp)
	}
	if pack := strings.TrimSpace(req.ContextPack); pack != "" {
		b.WriteString("\n\n### Document context\n")
		b.WriteString(pack)
	}
	if sum := strings.TrimSpace(req.SummaryContent); sum != "" {
		if len(sum) > 6000 {
			sum = sum[:6000] + "\n…[summary truncated]"
		}
		b.WriteString("\n\n### Current research summary (target document)\n")
		b.WriteString(sum)
	}
	return b.String()
}

// formatChatHistory turns prior turns into a readable transcript block.
// Roles are lowercased for the model; empty content is skipped.
func formatChatHistory(turns []models.ChatTurn) string {
	if len(turns) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("### Conversation so far\n")
	for _, t := range turns {
		role := strings.ToLower(strings.TrimSpace(t.Role))
		content := strings.TrimSpace(t.Content)
		if content == "" {
			continue
		}
		switch role {
		case "user", "assistant", "system":
		default:
			role = "user"
		}
		b.WriteString(role)
		b.WriteString(": ")
		b.WriteString(content)
		b.WriteString("\n\n")
	}
	return b.String()
}

// buildPrompt translates an AI operation into a model prompt, the result kind
// its answer belongs to, and its token ceiling. It returns "" for operations
// this service does not serve (AI_SEARCH is rejected earlier with its own
// honest message).
func buildPrompt(op models.AIOperation, req models.AIRequest) (string, promptKind, int) {
	sel := selectionText(req)
	query := strings.TrimSpace(req.Query)
	switch op {
	case models.OpExplain:
		base := "Explain the following passage from a research document concisely, in 2-4 sentences, for a researcher. Use the document context when it clarifies the passage. Always finish your final sentence:\n\n" + sel
		return withContext(base, req), kindExplanation, tokenBudgetExplanation
	case models.OpVerify:
		base := "Assess the following claim from a research document in 2-4 sentences: is it well-supported on its face, and what caveat (if any) should a researcher keep in mind? Use surrounding context when provided. Always finish your final sentence. Claim:\n\n" + sel
		return withContext(base, req), kindExplanation, tokenBudgetExplanation
	case models.OpExpand:
		base := "Go deeper on the following passage from a research document: unpack the mechanism, add relevant quantitative or contextual detail, in one short paragraph. Prefer details supported by the document context. Always finish your final sentence:\n\n" + sel
		return withContext(base, req), kindExplanation, tokenBudgetExplanation
	case models.OpSummarize:
		text := sel
		if text == "" {
			text = query
		}
		base := "Summarize the following text in 3-5 sentences for a research summary. Always finish your final sentence:\n\n" + text
		return withContext(base, req), kindExplanation, tokenBudgetExplanation
	case models.OpMerge:
		base := "Draft a 1-2 sentence insertion for a research summary based on the following selected passage. Keep it factual and self-contained. Prefer not to repeat claims already present in the current research summary when that context is provided. Always finish your final sentence:\n\n" + sel
		return withContext(base, req), kindInsertion, tokenBudgetExplanation
	case models.OpRewrite:
		draft := query
		if draft == "" {
			draft = sel
		}
		base := "Revise the following draft sentence(s) for clarity and academic tone. Return ONLY the revised text, no commentary:\n\n" + draft
		// Revise-with-context (plan 11 §3): when the UI names the pending
		// proposal, the model revises THAT span against the summary snapshot
		// (appended by withContext) — not a bare sentence in a vacuum.
		if fb := focusedProposalBlock(req.FocusedChange); fb != "" {
			base += "\n\n" + fb
		}
		return withContext(base, req), kindRevision, tokenBudgetShort
	case models.OpChat:
		// Multi-turn: history + current user message (Query or CustomPrompt).
		// The summary snapshot (withContext) is the harmonization baseline:
		// the model may redefine, restructure, or remove — never append-only.
		// @doc mentions arrive as MentionIDs; the selection (if any) scopes
		// the region. The frontend sends both explicitly so nothing is guessed.
		current := query
		if current == "" {
			current = strings.TrimSpace(req.CustomPrompt)
		}
		if current == "" {
			current = sel
		}
		var b strings.Builder
		b.WriteString("You are a research assistant helping a scholar read and reason about documents. ")
		b.WriteString("Answer clearly and concisely. Prefer evidence from the document context when provided. ")
		b.WriteString("The research summary is a living document: when new material arrives, harmonize it with what is already there — ")
		b.WriteString("redefine a stale definition, restructure a section, or drop what a new source disproves. Do not restrict yourself to appending. ")
		b.WriteString("Always finish your final sentence.\n\n")
		if len(req.MentionIDs) > 0 {
			b.WriteString("### Mentioned documents (explicit @doc targets, do not guess others)\n")
			b.WriteString(strings.Join(req.MentionIDs, ", "))
			b.WriteString("\n\n")
		}
		if sel != "" {
			b.WriteString("### Anchored region (the user selected this span; scope edits here when asked to change text)\n")
			b.WriteString(sel)
			if strings.TrimSpace(req.BlockID) != "" {
				b.WriteString("\n(block " + strings.TrimSpace(req.BlockID) + ")")
			}
			b.WriteString("\n\n")
		}
		if fc := req.FocusedChange; fc != nil && (strings.TrimSpace(fc.NewContent) != "" || strings.TrimSpace(fc.OldContent) != "") {
			b.WriteString(focusedProposalBlock(req.FocusedChange))
		}
		if hist := formatChatHistory(req.MessageHistory); hist != "" {
			b.WriteString(hist)
		}
		b.WriteString("### Current user message\n")
		b.WriteString(current)
		return withContext(b.String(), req), kindExplanation, tokenBudgetChat
	default:
		return "", "", 0
	}
}

// buildPromptFor is the service entry: pure prompt plus matched skill
// excerpts (≤2, ~1500 chars total), so tool discipline travels with the
// request instead of one ever-growing system prompt. No skills loaded →
// output identical to buildPrompt.
func (g *GeminiService) buildPromptFor(op models.AIOperation, req models.AIRequest) (string, promptKind, int) {
	prompt, kind, budget := buildPrompt(op, req)
	if prompt == "" {
		return prompt, kind, budget
	}
	return g.attachSkills(prompt, op, req), kind, budget
}

// skillExcerpt is one injectable skill (JSON-serializable for MatchSkills).
type skillExcerpt struct {
	Name    string `json:"name"`
	Excerpt string `json:"excerpt"`
}

// matchedSkillExcerpts resolves operation+intent to ≤2 capped excerpts.
func (g *GeminiService) matchedSkillExcerpts(op models.AIOperation, intent string) []skillExcerpt {
	if g == nil || len(g.skills) == 0 {
		return nil
	}
	matched := skills.Match(g.skills, string(op), intent)
	var out []skillExcerpt
	total := 0
	for i, s := range matched {
		if i >= 2 {
			break
		}
		cap := 1500
		if n, ok := s.Budgets["maxChars"]; ok && n > 0 && n < cap {
			cap = n
		}
		ex := skills.Excerpt(s, cap)
		if i > 0 && total+len(ex) > 1600 {
			break
		}
		out = append(out, skillExcerpt{Name: s.Name, Excerpt: ex})
		total += len(ex)
	}
	return out
}

// attachSkills appends matched skill excerpts (plan 11 §1). Matching is by
// operation + user text against each skill's `when`; injection is capped so
// prompts stay budgeted.
func (g *GeminiService) attachSkills(base string, op models.AIOperation, req models.AIRequest) string {
	matched := g.matchedSkillExcerpts(op, strings.TrimSpace(req.Query+" "+req.CustomPrompt))
	if len(matched) == 0 {
		return base
	}
	var b strings.Builder
	b.WriteString(base)
	b.WriteString("\n\n### Working skills (follow the steps; verify each action from its tool result)\n")
	for _, e := range matched {
		b.WriteString(e.Excerpt)
		b.WriteString("\n")
	}
	return b.String()
}

// focusedProposalBlock renders the pending proposal a turn likely refers to
// ("shorten this proposal"). Shared by chat (revise-by-reference) and
// rewrite (Review revise button). Empty when nothing is focused.
func focusedProposalBlock(fc *models.FocusedChange) string {
	if fc == nil || (strings.TrimSpace(fc.NewContent) == "" && strings.TrimSpace(fc.OldContent) == "") {
		return ""
	}
	var b strings.Builder
	b.WriteString("### Focused pending proposal (the user likely means THIS when they say \"this proposal\", \"shorten it\", \"expand it\")\n")
	fmt.Fprintf(&b, "(change %s, %s, block %s)\n",
		strings.TrimSpace(fc.ID), strings.TrimSpace(fc.Op), strings.TrimSpace(fc.BlockID))
	if strings.TrimSpace(fc.OldContent) != "" {
		b.WriteString("Before:\n" + strings.TrimSpace(fc.OldContent) + "\n")
	}
	b.WriteString("Proposed:\n" + strings.TrimSpace(fc.NewContent) + "\n")
	b.WriteString("To revise it, emit propose_summary_edit with op=\"modify\" (or \"delete\") targeting that block and the FULL revised text. ")
	b.WriteString("Do not touch it unless the user asks.\n\n")
	return b.String()
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
	b, err := json.Marshal(res)
	if err != nil {
		return `{"explanation":""}`
	}
	return string(b)
}
