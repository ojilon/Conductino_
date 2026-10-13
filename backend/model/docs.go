package model

// Library + open-document contract (mirrors TS domain.ts FileTreeNode and
// the OpenFile value in frontend/src/services/backend.ts — keep in sync).

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

// Node is one library tree entry. Path is the opaque token the frontend
// sends back: relative to the workspace root, slash-separated, never
// absolute — so no machine-specific path leaks into UI state. Kind tells
// the UI what a click means: "folder" expands, "file" opens.
type Node struct {
	ID       string `json:"id"`
	Label    string `json:"label"`
	Kind     string `json:"kind"` // "folder" | "file"
	Ext      string `json:"ext,omitempty"`
	Path     string `json:"path,omitempty"` // "" on the root itself
	Children []Node `json:"children,omitempty"`
}

// Doc is one opened workspace file. On the UI path BlocksJSON is empty and
// Dump names the extract_dump file holding the content (read via
// ReadDumpFile — big files never sit in Go memory twice); tools get inline
// BlocksJSON instead. Reason empty = success, else one of the Reason*
// constants with Detail carrying the cause. Root tags the opening folder so
// the UI detects stale tabs after a folder switch.
type Doc struct {
	Title      string `json:"title"`
	BlocksJSON string `json:"blocksJSON"`
	PageCount  int    `json:"pageCount,omitempty"`
	Kind       string `json:"kind,omitempty"`
	Root       string `json:"root,omitempty"`
	Dump       string `json:"dump,omitempty"`
	Reason     string `json:"reason,omitempty"`
	Detail     string `json:"detail,omitempty"`
}
