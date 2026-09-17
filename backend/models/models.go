// Package models mirrors the frontend domain types (src/types/domain.ts).
// Keep the two in sync manually — or generate one from the other — so the
// Wails JSON boundary stays unambiguous.
package models

type SourceKind string

const (
	SourceWeb     SourceKind = "web"
	SourcePDF     SourceKind = "pdf"
	SourceDOCX    SourceKind = "docx"
	SourceHTML    SourceKind = "html"
	SourceText    SourceKind = "text"
	SourceSummary SourceKind = "summary"
)

type Source struct {
	ID          string     `json:"id"`
	Kind        SourceKind `json:"kind"`
	Title       string     `json:"title"`
	Origin      string     `json:"origin"`
	URL         string     `json:"url,omitempty"`
	TypeLabel   string     `json:"typeLabel"`
	Abstract    string     `json:"abstract"`
	Rank        int        `json:"rank,omitempty"`
	Relevance   string     `json:"relevance,omitempty"`
	Saved       bool       `json:"saved"`
	ReaderDocID string     `json:"readerDocId,omitempty"`
}

type FileTreeNode struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Kind  string `json:"kind"` // "folder" | "file"
	Ext   string `json:"ext,omitempty"`
	// Path locates the node on disk, relative to the workspace root the
	// backend holds privately (Filesystem.root, set from the folder dialog).
	// It is relative — never absolute — so no machine-specific path leaks
	// into UI state or persistence. The frontend treats it as an opaque
	// token: display Label, send Path back for open/extract/reveal, never
	// build paths itself. Mirrors `path?` on the TS FileTreeNode.
	Path       string         `json:"path,omitempty"`
	Children   []FileTreeNode `json:"children,omitempty"`
	DocumentID string         `json:"documentId,omitempty"`
}

// OpenedDocument is the result of opening one workspace file: real extracted
// content for supported types. BlocksJSON decodes to DocumentBlock[] on the
// TS side (frontend/src/types/domain.ts). Produced by Documents.OpenFile,
// carried over the boundary by App.OpenFile.
//
// Typed failure (tasks.md 1.2): classified failures are returned as a VALUE
// with Reason set and BlocksJSON empty — not as a Go error — so the UI can
// tell "unsupported type" (the only case that may use a placeholder) apart
// from real read failures without string-matching error text. A Go error
// return is reserved for truly unexpected bridge failures.
type OpenedDocument struct {
	Title      string `json:"title"`
	BlocksJSON string `json:"blocksJSON"`
	PageCount  int    `json:"pageCount,omitempty"`
	Kind       string `json:"kind,omitempty"` // "text" for the plain-text arm
	// Root is the absolute workspace root the file was resolved against.
	// Tags the document with its opening folder (tasks.md 1.1 option b)
	// so the UI can detect stale tabs after a folder switch.
	Root string `json:"root,omitempty"`
	// Reason is empty on success; otherwise one of the OpenFailureReason
	// values below with Detail carrying the human-readable cause.
	Reason string `json:"reason,omitempty"`
	Detail string `json:"detail,omitempty"`
}

// OpenFailureReason classifies why a file could not be opened. Mirrors the
// TS OpenFailureReason in frontend/src/services/backend.ts — keep in sync.
type OpenFailureReason string

const (
	ReasonUnsupported      OpenFailureReason = "unsupported"
	ReasonNotFound         OpenFailureReason = "not_found"
	ReasonPermissionDenied OpenFailureReason = "permission_denied"
	ReasonTooLarge         OpenFailureReason = "too_large"
	ReasonParseError       OpenFailureReason = "parse_error"
)

type AIOperation string

const (
	OpSearch    AIOperation = "AI_SEARCH"
	OpSummarize AIOperation = "AI_SUMMARIZE"
	OpExplain   AIOperation = "AI_EXPLAIN"
	OpExpand    AIOperation = "AI_EXPAND"
	OpMerge     AIOperation = "AI_MERGE"
	OpRewrite   AIOperation = "AI_REWRITE"
	OpVerify    AIOperation = "AI_VERIFY"
)

// AIRequest is the wire shape of an AI operation (mirrors TS AIRequest in
// frontend/src/types/domain.ts, translated by the WailsAIProvider in
// frontend/src/services/ai.ts). TS `selection` is an object
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
	// RequestID correlates events back to the originating call: every AIEvent
	// the backend emits for this request echoes it, so concurrent requests
	// never cross-talk on the shared "ai://event" channel.
	RequestID string `json:"requestId,omitempty"`
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
