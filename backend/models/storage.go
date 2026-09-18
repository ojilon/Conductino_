package models

// Persistence records (SQLite rows / in-memory maps). Owned here so every
// package (services, extract cache writers, future storage impls) shares one
// definition. Timestamps are Unix epoch milliseconds (INTEGER in SQLite);
// zero means "not set" — upserts fill it with time.Now().UnixMilli().

// CachedExtract is one extraction result for a workspace file. Path is the
// root-anchored key (root + rel token), so the same relative file under a
// different folder never collides. (Plan 07 adds md_text + page_map_json
// columns; same key, same LRU.)
type CachedExtract struct {
	Path       string `json:"path"`
	Mtime      int64  `json:"mtime"`
	Size       int64  `json:"size"`
	Kind       string `json:"kind,omitempty"`
	Title      string `json:"title,omitempty"`
	BlocksJSON string `json:"blocksJson,omitempty"`
	PageCount  int    `json:"pageCount,omitempty"`
}

type WorkspaceRecord struct {
	ID               string `json:"id"`
	RootPath         string `json:"rootPath"`
	PrimarySummaryID string `json:"primarySummaryId,omitempty"`
	Label            string `json:"label,omitempty"`
	// UpdatedAt is Unix epoch milliseconds (INTEGER in SQLite).
	// Zero means "not set" — UpsertWorkspace fills it with time.Now().UnixMilli().
	UpdatedAt int64 `json:"updatedAt,omitempty"`
}

type DocumentRecord struct {
	ID          string `json:"id"`
	WorkspaceID string `json:"workspaceId"`
	Kind        string `json:"kind"` // source | summary
	// SourceID links a source document back to its Source row ("" for summaries).
	SourceID string `json:"sourceId,omitempty"`
	Title    string `json:"title"`
	// BlocksJSON is the canonical DocumentBlock[] wire payload (JSON string).
	BlocksJSON string `json:"blocksJson,omitempty"`
	// MetaJSON carries DocumentMetadata (author/venue/format/path/rootPath, ...).
	MetaJSON string `json:"metaJson,omitempty"`
	// UpdatedAt is Unix epoch milliseconds (INTEGER in SQLite).
	UpdatedAt int64 `json:"updatedAt,omitempty"`
}

type ChangeRecord struct {
	ID         string `json:"id"`
	DocumentID string `json:"documentId"`
	// WorkspaceID scopes the proposal to its folder (Phase 2 multi-workspace fix).
	WorkspaceID string `json:"workspaceId,omitempty"`
	Type        string `json:"type"` // insert | modify | delete
	BlockID     string `json:"blockId,omitempty"`
	OldContent  string `json:"oldContent,omitempty"`
	NewContent  string `json:"newContent,omitempty"`
	Status      string `json:"status"` // pending | accepted | rejected
	SourceID   string `json:"sourceId,omitempty"`
	ActivityID string `json:"activityId,omitempty"`
	// CreatedAt is Unix epoch milliseconds (INTEGER in SQLite).
	CreatedAt int64 `json:"createdAt,omitempty"`
}

type ChatThreadRecord struct {
	ID          string `json:"id"`
	WorkspaceID string `json:"workspaceId"`
	// DocumentID optionally ties the thread to the open document ("" = workspace-wide).
	DocumentID string `json:"documentId,omitempty"`
	Title      string `json:"title,omitempty"`
	// CreatedAt/UpdatedAt are Unix epoch milliseconds (INTEGER in SQLite).
	CreatedAt int64 `json:"createdAt,omitempty"`
	UpdatedAt int64 `json:"updatedAt,omitempty"`
}

type ChatMessageRecord struct {
	ID       string `json:"id"`
	ThreadID string `json:"threadId"`
	Role     string `json:"role"` // user | assistant | system
	Content  string `json:"content"`
	// DocumentID records the focused document when the message was sent.
	DocumentID string `json:"documentId,omitempty"`
	// CreatedAt is Unix epoch milliseconds (INTEGER in SQLite).
	CreatedAt int64 `json:"createdAt,omitempty"`
}
