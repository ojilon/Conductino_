package services

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

// SQLiteStorage is the Phase 6 system of record (pure-Go driver, no CGO).
type SQLiteStorage struct {
	db   *sql.DB
	path string
}

// NewSQLiteStorage opens (or creates) the database file at path.
func NewSQLiteStorage(path string) (*SQLiteStorage, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create db dir: %w", err)
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(`PRAGMA foreign_keys = ON; PRAGMA journal_mode = WAL;`); err != nil {
		_ = db.Close()
		return nil, err
	}
	return &SQLiteStorage{db: db, path: path}, nil
}

func (s *SQLiteStorage) Engine() string { return "sqlite" }

func (s *SQLiteStorage) Close() error {
	if s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *SQLiteStorage) Init(_ context.Context) error {
	schema := `
CREATE TABLE IF NOT EXISTS settings (
  key   TEXT PRIMARY KEY,
  value TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS workspaces (
  id                  TEXT PRIMARY KEY,
  root_path           TEXT NOT NULL UNIQUE,
  primary_summary_id  TEXT NOT NULL DEFAULT '',
  label               TEXT NOT NULL DEFAULT '',
  updated_at          INTEGER NOT NULL
);
CREATE TABLE IF NOT EXISTS documents (
  id            TEXT PRIMARY KEY,
  workspace_id  TEXT NOT NULL,
  kind          TEXT NOT NULL,
  source_id     TEXT NOT NULL DEFAULT '',
  title         TEXT NOT NULL DEFAULT '',
  blocks_json   TEXT NOT NULL DEFAULT '',
  meta_json     TEXT NOT NULL DEFAULT '',
  updated_at    INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_documents_ws ON documents(workspace_id);
CREATE TABLE IF NOT EXISTS document_changes (
  id            TEXT PRIMARY KEY,
  document_id   TEXT NOT NULL,
  workspace_id  TEXT NOT NULL DEFAULT '',
  type          TEXT NOT NULL,
  block_id      TEXT NOT NULL DEFAULT '',
  old_content   TEXT NOT NULL DEFAULT '',
  new_content   TEXT NOT NULL DEFAULT '',
  status        TEXT NOT NULL DEFAULT 'pending',
  source_id     TEXT NOT NULL DEFAULT '',
  activity_id   TEXT NOT NULL DEFAULT '',
  created_at    INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_changes_doc ON document_changes(document_id, status);
CREATE TABLE IF NOT EXISTS chat_threads (
  id            TEXT PRIMARY KEY,
  workspace_id  TEXT NOT NULL,
  document_id   TEXT NOT NULL DEFAULT '',
  title         TEXT NOT NULL DEFAULT '',
  created_at    INTEGER NOT NULL,
  updated_at    INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_threads_ws ON chat_threads(workspace_id);
CREATE TABLE IF NOT EXISTS chat_messages (
  id            TEXT PRIMARY KEY,
  thread_id     TEXT NOT NULL,
  role          TEXT NOT NULL,
  content       TEXT NOT NULL,
  document_id   TEXT NOT NULL DEFAULT '',
  created_at    INTEGER NOT NULL,
  FOREIGN KEY (thread_id) REFERENCES chat_threads(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_messages_thread ON chat_messages(thread_id, created_at);
`
	_, err := s.db.Exec(schema)
	return err
}

func (s *SQLiteStorage) Put(key string, value any) error {
	return s.SetSetting(key, fmt.Sprint(value))
}

func (s *SQLiteStorage) Get(key string) (any, bool) {
	v, ok := s.GetSetting(key)
	if !ok {
		return nil, false
	}
	return v, true
}

func (s *SQLiteStorage) SetSetting(key, value string) error {
	_, err := s.db.Exec(
		`INSERT INTO settings(key, value) VALUES(?, ?)
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value`,
		key, value,
	)
	return err
}

func (s *SQLiteStorage) GetSetting(key string) (string, bool) {
	var v string
	err := s.db.QueryRow(`SELECT value FROM settings WHERE key = ?`, key).Scan(&v)
	if err == sql.ErrNoRows {
		return "", false
	}
	if err != nil {
		return "", false
	}
	return v, true
}

func (s *SQLiteStorage) UpsertWorkspace(ws WorkspaceRecord) error {
	if ws.UpdatedAt == 0 {
		ws.UpdatedAt = time.Now().UnixMilli()
	}
	_, err := s.db.Exec(
		`INSERT INTO workspaces(id, root_path, primary_summary_id, label, updated_at)
		 VALUES(?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET
		   root_path = excluded.root_path,
		   primary_summary_id = excluded.primary_summary_id,
		   label = excluded.label,
		   updated_at = excluded.updated_at`,
		ws.ID, ws.RootPath, ws.PrimarySummaryID, ws.Label, ws.UpdatedAt,
	)
	return err
}

func (s *SQLiteStorage) GetWorkspace(id string) (*WorkspaceRecord, error) {
	var ws WorkspaceRecord
	err := s.db.QueryRow(
		`SELECT id, root_path, primary_summary_id, label, updated_at FROM workspaces WHERE id = ?`, id,
	).Scan(&ws.ID, &ws.RootPath, &ws.PrimarySummaryID, &ws.Label, &ws.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &ws, nil
}

func (s *SQLiteStorage) GetWorkspaceByRoot(root string) (*WorkspaceRecord, error) {
	var ws WorkspaceRecord
	err := s.db.QueryRow(
		`SELECT id, root_path, primary_summary_id, label, updated_at FROM workspaces WHERE root_path = ?`, root,
	).Scan(&ws.ID, &ws.RootPath, &ws.PrimarySummaryID, &ws.Label, &ws.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &ws, nil
}

func (s *SQLiteStorage) ListWorkspaces() ([]WorkspaceRecord, error) {
	rows, err := s.db.Query(`SELECT id, root_path, primary_summary_id, label, updated_at FROM workspaces ORDER BY updated_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []WorkspaceRecord
	for rows.Next() {
		var ws WorkspaceRecord
		if err := rows.Scan(&ws.ID, &ws.RootPath, &ws.PrimarySummaryID, &ws.Label, &ws.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, ws)
	}
	return out, rows.Err()
}

func (s *SQLiteStorage) UpsertDocument(doc DocumentRecord) error {
	if doc.UpdatedAt == 0 {
		doc.UpdatedAt = time.Now().UnixMilli()
	}
	_, err := s.db.Exec(
		`INSERT INTO documents(id, workspace_id, kind, source_id, title, blocks_json, meta_json, updated_at)
		 VALUES(?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET
		   workspace_id = excluded.workspace_id,
		   kind = excluded.kind,
		   source_id = excluded.source_id,
		   title = excluded.title,
		   blocks_json = excluded.blocks_json,
		   meta_json = excluded.meta_json,
		   updated_at = excluded.updated_at`,
		doc.ID, doc.WorkspaceID, doc.Kind, doc.SourceID, doc.Title, doc.BlocksJSON, doc.MetaJSON, doc.UpdatedAt,
	)
	return err
}

func (s *SQLiteStorage) GetDocument(id string) (*DocumentRecord, error) {
	var d DocumentRecord
	err := s.db.QueryRow(
		`SELECT id, workspace_id, kind, source_id, title, blocks_json, meta_json, updated_at FROM documents WHERE id = ?`, id,
	).Scan(&d.ID, &d.WorkspaceID, &d.Kind, &d.SourceID, &d.Title, &d.BlocksJSON, &d.MetaJSON, &d.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (s *SQLiteStorage) ListDocuments(workspaceID string) ([]DocumentRecord, error) {
	q := `SELECT id, workspace_id, kind, source_id, title, blocks_json, meta_json, updated_at FROM documents`
	var rows *sql.Rows
	var err error
	if workspaceID != "" {
		rows, err = s.db.Query(q+` WHERE workspace_id = ? ORDER BY updated_at DESC`, workspaceID)
	} else {
		rows, err = s.db.Query(q + ` ORDER BY updated_at DESC`)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []DocumentRecord
	for rows.Next() {
		var d DocumentRecord
		if err := rows.Scan(&d.ID, &d.WorkspaceID, &d.Kind, &d.SourceID, &d.Title, &d.BlocksJSON, &d.MetaJSON, &d.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (s *SQLiteStorage) GetPrimarySummary(workspaceID string) (*DocumentRecord, error) {
	ws, err := s.GetWorkspace(workspaceID)
	if err != nil || ws == nil || ws.PrimarySummaryID == "" {
		return nil, err
	}
	return s.GetDocument(ws.PrimarySummaryID)
}

func (s *SQLiteStorage) UpsertChange(ch ChangeRecord) error {
	if ch.CreatedAt == 0 {
		ch.CreatedAt = time.Now().UnixMilli()
	}
	_, err := s.db.Exec(
		`INSERT INTO document_changes(id, document_id, workspace_id, type, block_id, old_content, new_content, status, source_id, activity_id, created_at)
		 VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET
		   status = excluded.status,
		   new_content = excluded.new_content,
		   old_content = excluded.old_content`,
		ch.ID, ch.DocumentID, ch.WorkspaceID, ch.Type, ch.BlockID, ch.OldContent, ch.NewContent, ch.Status, ch.SourceID, ch.ActivityID, ch.CreatedAt,
	)
	return err
}

func (s *SQLiteStorage) ListPendingChanges(documentID string) ([]ChangeRecord, error) {
	rows, err := s.db.Query(
		`SELECT id, document_id, workspace_id, type, block_id, old_content, new_content, status, source_id, activity_id, created_at
		 FROM document_changes WHERE document_id = ? AND status = 'pending' ORDER BY created_at`,
		documentID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ChangeRecord
	for rows.Next() {
		var c ChangeRecord
		if err := rows.Scan(&c.ID, &c.DocumentID, &c.WorkspaceID, &c.Type, &c.BlockID, &c.OldContent, &c.NewContent, &c.Status, &c.SourceID, &c.ActivityID, &c.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *SQLiteStorage) UpsertThread(t ChatThreadRecord) error {
	now := time.Now().UnixMilli()
	if t.CreatedAt == 0 {
		t.CreatedAt = now
	}
	if t.UpdatedAt == 0 {
		t.UpdatedAt = now
	}
	_, err := s.db.Exec(
		`INSERT INTO chat_threads(id, workspace_id, document_id, title, created_at, updated_at)
		 VALUES(?, ?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET
		   title = excluded.title,
		   document_id = excluded.document_id,
		   updated_at = excluded.updated_at`,
		t.ID, t.WorkspaceID, t.DocumentID, t.Title, t.CreatedAt, t.UpdatedAt,
	)
	return err
}

func (s *SQLiteStorage) AppendMessage(msg ChatMessageRecord) error {
	if msg.CreatedAt == 0 {
		msg.CreatedAt = time.Now().UnixMilli()
	}
	_, err := s.db.Exec(
		`INSERT INTO chat_messages(id, thread_id, role, content, document_id, created_at) VALUES(?, ?, ?, ?, ?, ?)`,
		msg.ID, msg.ThreadID, msg.Role, msg.Content, msg.DocumentID, msg.CreatedAt,
	)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`UPDATE chat_threads SET updated_at = ? WHERE id = ?`, msg.CreatedAt, msg.ThreadID)
	return err
}

func (s *SQLiteStorage) GetThread(id string) (*ChatThreadRecord, error) {
	var t ChatThreadRecord
	err := s.db.QueryRow(
		`SELECT id, workspace_id, document_id, title, created_at, updated_at FROM chat_threads WHERE id = ?`, id,
	).Scan(&t.ID, &t.WorkspaceID, &t.DocumentID, &t.Title, &t.CreatedAt, &t.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (s *SQLiteStorage) ListThreads(workspaceID string) ([]ChatThreadRecord, error) {
	rows, err := s.db.Query(
		`SELECT id, workspace_id, document_id, title, created_at, updated_at FROM chat_threads
		 WHERE workspace_id = ? OR ? = '' ORDER BY updated_at DESC`,
		workspaceID, workspaceID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ChatThreadRecord
	for rows.Next() {
		var t ChatThreadRecord
		if err := rows.Scan(&t.ID, &t.WorkspaceID, &t.DocumentID, &t.Title, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *SQLiteStorage) ListMessages(threadID string) ([]ChatMessageRecord, error) {
	rows, err := s.db.Query(
		`SELECT id, thread_id, role, content, document_id, created_at FROM chat_messages
		 WHERE thread_id = ? ORDER BY created_at ASC`,
		threadID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ChatMessageRecord
	for rows.Next() {
		var m ChatMessageRecord
		if err := rows.Scan(&m.ID, &m.ThreadID, &m.Role, &m.Content, &m.DocumentID, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

var _ StorageService = (*SQLiteStorage)(nil)
