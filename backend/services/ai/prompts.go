package ai

import (
	"encoding/json"
	"strings"

	"Conductino/backend/models"
)

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
		return withContext(base, req), kindRevision, tokenBudgetShort
	case models.OpChat:
		// Multi-turn: history + current user message (Query or CustomPrompt).
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
		b.WriteString("Always finish your final sentence.\n\n")
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
