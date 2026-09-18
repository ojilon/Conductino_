package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"Conductino/backend/models"
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

	// Extract cache (issues 24+26): file extraction results keyed by
	// root-anchored path + mtime + size, so reopening an unchanged file
	// never re-extracts. Bounded (LRU-ish eviction inside the impls).
	GetCachedExtract(path string, mtime, size int64) (*CachedExtract, error)
	PutCachedExtract(e CachedExtract) error
}

// Persistence record aliases (plan 06 step 4). Canonical definitions live in
// backend/models/storage.go so every package shares them; these aliases keep
// storage.go, sqlite_storage.go, workspace.go, backend/main.go, and the Wails
// shell compiling untouched. New code must import backend/models directly.

// CachedExtract is one extraction result for a workspace file.
type CachedExtract = models.CachedExtract

type WorkspaceRecord = models.WorkspaceRecord

type DocumentRecord = models.DocumentRecord

type ChangeRecord = models.ChangeRecord

type ChatThreadRecord = models.ChatThreadRecord

type ChatMessageRecord = models.ChatMessageRecord

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
	extracts   map[string]CachedExtract       // cacheKey -> extract
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
		extracts:   make(map[string]CachedExtract),
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

// inMemoryExtractCap mirrors the SQLite cap for test parity.
const inMemoryExtractCap = 200

func (s *InMemoryStorage) GetCachedExtract(path string, mtime, size int64) (*CachedExtract, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	e, ok := s.extracts[path]
	if !ok || e.Mtime != mtime || e.Size != size {
		return nil, nil
	}
	c := e
	return &c, nil
}

func (s *InMemoryStorage) PutCachedExtract(e CachedExtract) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.extracts[e.Path]; !ok && len(s.extracts) >= inMemoryExtractCap {
		for k := range s.extracts {
			delete(s.extracts, k) // arbitrary oldest stand-in; SQLite uses LRU
			break
		}
	}
	s.extracts[e.Path] = e
	return nil
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
