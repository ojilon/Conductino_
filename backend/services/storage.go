package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// StorageService is the persistence boundary for workspace meta, documents,
// changes, and chat. Phase 6: SQLite (modernc.org/sqlite, pure-Go) is the
// default; InMemoryStorage remains for unit tests.
type StorageService interface {
	Init(ctx context.Context) error
	Engine() string
	Close() error

	Put(key string, value any) error
	Get(key string) (any, bool)

	SetSetting(key, value string) error
	GetSetting(key string) (string, bool)

	UpsertWorkspace(ws WorkspaceRecord) error
	GetWorkspace(id string) (*WorkspaceRecord, error)
	GetWorkspaceByRoot(root string) (*WorkspaceRecord, error)
	ListWorkspaces() ([]WorkspaceRecord, error)

	UpsertDocument(doc DocumentRecord) error
	GetDocument(id string) (*DocumentRecord, error)
	ListDocuments(workspaceID string) ([]DocumentRecord, error)
	GetPrimarySummary(workspaceID string) (*DocumentRecord, error)

	UpsertChange(ch ChangeRecord) error
	ListPendingChanges(documentID string) ([]ChangeRecord, error)

	UpsertThread(t ChatThreadRecord) error
	AppendMessage(msg ChatMessageRecord) error
	GetThread(id string) (*ChatThreadRecord, error)
	ListThreads(workspaceID string) ([]ChatThreadRecord, error)
	ListMessages(threadID string) ([]ChatMessageRecord, error)
}

type WorkspaceRecord struct {
	ID               string `json:"id"`
	RootPath         string `json:"rootPath"`
	PrimarySummaryID string `json:"primarySummaryId,omitempty"`
	Label            string `json:"label,omitempty"`
	UpdatedAt        string `json:"updatedAt,omitempty"`
}

type DocumentRecord struct {
	ID          string `json:"id"`
	WorkspaceID string `json:"workspaceId"`
	Kind        string `json:"kind"` // source | summary
	Title       string `json:"title"`
	Body        string `json:"body"`
	SourcePath  string `json:"sourcePath,omitempty"`
	UpdatedAt   string `json:"updatedAt,omitempty"`
}

type ChangeRecord struct {
	ID         string `json:"id"`
	DocumentID string `json:"documentId"`
	Type       string `json:"type"`
	OldText    string `json:"oldText,omitempty"`
	NewText    string `json:"newText,omitempty"`
	Status     string `json:"status"` // pending | accepted | rejected
	SourceID   string `json:"sourceId,omitempty"`
	UpdatedAt  string `json:"updatedAt,omitempty"`
}

type ChatThreadRecord struct {
	ID          string `json:"id"`
	WorkspaceID string `json:"workspaceId"`
	Title       string `json:"title,omitempty"`
	UpdatedAt   string `json:"updatedAt,omitempty"`
}

type ChatMessageRecord struct {
	ID        string `json:"id"`
	ThreadID  string `json:"threadId"`
	Role      string `json:"role"` // user | assistant | system
	Content   string `json:"content"`
	CreatedAt string `json:"createdAt,omitempty"`
}

/* ------------------------------------------------------------------ */
/* In-memory implementation                                            */
/* ------------------------------------------------------------------ */

type InMemoryStorage struct {
	mu         sync.RWMutex
	data       map[string]any
	settings   map[string]string
	workspaces map[string]WorkspaceRecord
	documents  map[string]DocumentRecord
	changes    map[string]ChangeRecord
	threads    map[string]ChatThreadRecord
	messages   map[string][]ChatMessageRecord // threadID -> msgs
}

func NewStorage() *InMemoryStorage {
	return &InMemoryStorage{
		data:       make(map[string]any),
		settings:   make(map[string]string),
		workspaces: make(map[string]WorkspaceRecord),
		documents:  make(map[string]DocumentRecord),
		changes:    make(map[string]ChangeRecord),
		threads:    make(map[string]ChatThreadRecord),
		messages:   make(map[string][]ChatMessageRecord),
	}
}

func (s *InMemoryStorage) Init(_ context.Context) error { return nil }
func (s *InMemoryStorage) Engine() string               { return "memory" }
func (s *InMemoryStorage) Close() error                 { return nil }

func (s *InMemoryStorage) Put(key string, value any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = value
	return nil
}

func (s *InMemoryStorage) Get(key string) (any, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.data[key]
	return v, ok
}

func (s *InMemoryStorage) SetSetting(key, value string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.settings[key] = value
	return nil
}

func (s *InMemoryStorage) GetSetting(key string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.settings[key]
	return v, ok
}

func (s *InMemoryStorage) UpsertWorkspace(ws WorkspaceRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.workspaces[ws.ID] = ws
	return nil
}

func (s *InMemoryStorage) GetWorkspace(id string) (*WorkspaceRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	w, ok := s.workspaces[id]
	if !ok {
		return nil, fmt.Errorf("workspace %s not found", id)
	}
	return &w, nil
}

func (s *InMemoryStorage) GetWorkspaceByRoot(root string) (*WorkspaceRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, w := range s.workspaces {
		if w.RootPath == root {
			ww := w
			return &ww, nil
		}
	}
	return nil, fmt.Errorf("workspace for root %s not found", root)
}

func (s *InMemoryStorage) ListWorkspaces() ([]WorkspaceRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]WorkspaceRecord, 0, len(s.workspaces))
	for _, w := range s.workspaces {
		out = append(out, w)
	}
	return out, nil
}

func (s *InMemoryStorage) UpsertDocument(doc DocumentRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.documents[doc.ID] = doc
	return nil
}

func (s *InMemoryStorage) GetDocument(id string) (*DocumentRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	d, ok := s.documents[id]
	if !ok {
		return nil, fmt.Errorf("document %s not found", id)
	}
	return &d, nil
}

func (s *InMemoryStorage) ListDocuments(workspaceID string) ([]DocumentRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]DocumentRecord, 0)
	for _, d := range s.documents {
		if d.WorkspaceID == workspaceID {
			out = append(out, d)
		}
	}
	return out, nil
}

func (s *InMemoryStorage) GetPrimarySummary(workspaceID string) (*DocumentRecord, error) {
	ws, err := s.GetWorkspace(workspaceID)
	if err != nil || ws == nil || ws.PrimarySummaryID == "" {
		return nil, err
	}
	return s.GetDocument(ws.PrimarySummaryID)
}

func (s *InMemoryStorage) UpsertChange(ch ChangeRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.changes[ch.ID] = ch
	return nil
}

func (s *InMemoryStorage) ListPendingChanges(documentID string) ([]ChangeRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]ChangeRecord, 0)
	for _, c := range s.changes {
		if c.DocumentID == documentID && c.Status == "pending" {
			out = append(out, c)
		}
	}
	return out, nil
}

func (s *InMemoryStorage) UpsertThread(t ChatThreadRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.threads[t.ID] = t
	return nil
}

func (s *InMemoryStorage) AppendMessage(msg ChatMessageRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.messages[msg.ThreadID] = append(s.messages[msg.ThreadID], msg)
	return nil
}

func (s *InMemoryStorage) GetThread(id string) (*ChatThreadRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.threads[id]
	if !ok {
		return nil, fmt.Errorf("thread %s not found", id)
	}
	return &t, nil
}

func (s *InMemoryStorage) ListThreads(workspaceID string) ([]ChatThreadRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]ChatThreadRecord, 0)
	for _, t := range s.threads {
		if t.WorkspaceID == workspaceID {
			out = append(out, t)
		}
	}
	return out, nil
}

func (s *InMemoryStorage) ListMessages(threadID string) ([]ChatMessageRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	msgs := s.messages[threadID]
	out := make([]ChatMessageRecord, len(msgs))
	copy(out, msgs)
	return out, nil
}

/* ------------------------------------------------------------------ */
/* Open helpers                                                        */
/* ------------------------------------------------------------------ */

// DefaultDBPath returns the app-data path for conductino.db.
// Override with CONDUCTINO_DB.
func DefaultDBPath() string {
	if p := os.Getenv("CONDUCTINO_DB"); p != "" {
		return p
	}
	dir, err := os.UserConfigDir()
	if err != nil || dir == "" {
		dir = os.TempDir()
	}
	return filepath.Join(dir, "Conductino", "conductino.db")
}

// OpenDefaultStorage prefers SQLite; falls back to in-memory on open failure.
func OpenDefaultStorage(ctx context.Context) StorageService {
	path := DefaultDBPath()
	store, err := NewSQLiteStorage(path)
	if err != nil {
		mem := NewStorage()
		_ = mem.Init(ctx)
		return mem
	}
	if err := store.Init(ctx); err != nil {
		_ = store.Close()
		mem := NewStorage()
		_ = mem.Init(ctx)
		return mem
	}
	return store
}
