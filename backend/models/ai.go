package models

// AI wire contract (mirrors TS AIRequest / AIEvent in
// frontend/src/types/domain.ts; translated by the WailsAIProvider in
// frontend/src/services/ai.ts). Every event echoes RequestID so concurrent
// requests never cross-talk on the shared "ai://event" channel.

type AIOperation string

const (
	OpSearch    AIOperation = "AI_SEARCH"
	OpSummarize AIOperation = "AI_SUMMARIZE"
	OpExplain   AIOperation = "AI_EXPLAIN"
	OpExpand    AIOperation = "AI_EXPAND"
	OpMerge     AIOperation = "AI_MERGE"
	OpRewrite   AIOperation = "AI_REWRITE"
	OpVerify    AIOperation = "AI_VERIFY"
	OpChat      AIOperation = "AI_CHAT"
)

// ChatTurn is one prior message in a multi-turn chat request.
type ChatTurn struct {
	Role    string `json:"role"` // user | assistant | system
	Content string `json:"content"`
}

// AIRequest is the wire shape of an AI operation. TS `selection` is an object
// `{ blockId, text }`; it arrives here flattened into SelectionText/BlockID
// (Selection keeps the plain text for backward compatibility).
type AIRequest struct {
	Operation     string `json:"operation"`
	Query         string `json:"query,omitempty"`
	DocumentID    string `json:"documentId,omitempty"`
	SourceID      string `json:"sourceId,omitempty"`
	ChangeID      string `json:"changeId,omitempty"`
	Selection     string `json:"selection,omitempty"`
	SelectionText string `json:"selectionText,omitempty"`
	BlockID       string `json:"blockId,omitempty"`
	// Phase 2/3 workspace + oneshot context
	WorkspaceID            string `json:"workspaceId,omitempty"`
	CustomPrompt           string `json:"customPrompt,omitempty"`
	IncludeDocumentContext *bool  `json:"includeDocumentContext,omitempty"`
	// ContextPack is a budgeted string (selection window + outline) built
	// by the frontend while document blocks live in AppState. The AI
	// service appends it to the model prompt.
	ContextPack string `json:"contextPack,omitempty"`
	// Phase 4 multi-turn history (prior turns; current query is Query/Selection).
	MessageHistory []ChatTurn `json:"messageHistory,omitempty"`
	Mode           string     `json:"mode,omitempty"` // "oneshot" | "chat"
	// Phase 5: summary snapshot for read_summary / propose_summary_edit tools.
	// Frontend sends a truncated plain-text view until Phase 6 storage owns it.
	SummaryContent   string `json:"summaryContent,omitempty"`
	PrimarySummaryID string `json:"primarySummaryId,omitempty"`
	// Resolved @doc mentions: document ids named in the chat composer.
	// The raw @token stays in Query; these ids tell the harness exactly
	// which documents were meant. A mentioned summary overrides the
	// workspace primary as the proposal target.
	MentionIDs []string `json:"mentionIds,omitempty"`
	// Focused pending proposal for chat-targeted revise. The model revises
	// it by emitting a modify/delete proposal against the same block.
	FocusedChange *FocusedChange `json:"focusedChange,omitempty"`
	// RequestID correlates events back to the originating call: every AIEvent
	// the backend emits for this request echoes it, so concurrent requests
	// never cross-talk on the shared "ai://event" channel.
	RequestID string `json:"requestId,omitempty"`
}

// FocusedChange is the pending proposal a chat turn likely refers to
// ("shorten this proposal"). Mirrors the TS AIRequest.focusedChange.
type FocusedChange struct {
	ID         string `json:"id,omitempty"`
	Op         string `json:"op,omitempty"` // insert | modify | delete
	BlockID    string `json:"blockId,omitempty"`
	OldContent string `json:"oldContent,omitempty"`
	NewContent string `json:"newContent,omitempty"`
}

// AIEvent is one streaming unit emitted to the frontend over "ai://event".
type AIEvent struct {
	Type      string   `json:"type"` // "phase" | "sources" | "done" | "error"
	RequestID string   `json:"requestId,omitempty"`
	Phase     int      `json:"phase,omitempty"`
	Label     string   `json:"label,omitempty"`
	SourceIDs []string `json:"sourceIds,omitempty"`
	// Payload carries the JSON-encoded result on "done" (keys: explanation,
	// insertion {text, citation}, revision, relatedSources[]).
	Payload string `json:"payload,omitempty"`
	// Message carries the human-readable failure reason on "error".
	Message string `json:"message,omitempty"`
}
