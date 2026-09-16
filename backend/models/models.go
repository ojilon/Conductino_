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
	ID         string         `json:"id"`
	Label      string         `json:"label"`
	Kind       string         `json:"kind"` // "folder" | "file"
	Ext        string         `json:"ext,omitempty"`
	Children   []FileTreeNode `json:"children,omitempty"`
	DocumentID string         `json:"documentId,omitempty"`
}

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

// AIRequest is the wire shape of an AI operation (mirrors types/domain.ts).
type AIRequest struct {
	Operation  string `json:"operation"`
	Query      string `json:"query,omitempty"`
	DocumentID string `json:"documentId,omitempty"`
	SourceID   string `json:"sourceId,omitempty"`
	ChangeID   string `json:"changeId,omitempty"`
	Selection  string `json:"selection,omitempty"`
}

// AIEvent is one streaming unit emitted to the frontend.
type AIEvent struct {
	Type      string   `json:"type"` // "phase" | "sources" | "done" | "error"
	Phase     int      `json:"phase,omitempty"`
	Label     string   `json:"label,omitempty"`
	SourceIDs []string `json:"sourceIds,omitempty"`
	Payload   string   `json:"payload,omitempty"`
}
