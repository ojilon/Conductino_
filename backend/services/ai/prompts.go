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
