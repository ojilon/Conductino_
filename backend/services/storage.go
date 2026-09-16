package services

import (
	"context"
	"fmt"
	"sync"
)

// StorageService is the persistence boundary.
//
// Today: InMemoryStorage (map-backed, survives the session only).
// Future: SQLiteStorage using modernc.org/sqlite (pure-Go, no cgo —
// keeps the binary lightweight on low-spec Windows machines).
//
// Persistent concepts (schema is intentionally small for now):
//
//	sessions(id, title, created_at)
//	sources(id, kind, title, origin, url, saved)
//	documents(id, source_id, kind, title, body_json)
//	summary_sources(summary_document_id, source_id)
//	ai_activities(id, operation, status, started_at, finished_at)
//	document_changes(id, document_id, type, old, new, status, source_id)
//	bookmarks(source_id, created_at)
type StorageService interface {
	Init(ctx context.Context) error
	Engine() string
	Put(key string, value any) error
	Get(key string) (any, bool)
}

/* ------------------------------------------------------------------ */
/* In-memory implementation (default)                                  */
/* ------------------------------------------------------------------ */

type InMemoryStorage struct {
	mu   sync.RWMutex
	data map[string]any
}

func NewStorage() *InMemoryStorage {
	return &InMemoryStorage{data: make(map[string]any)}
}

func (s *InMemoryStorage) Init(_ context.Context) error { return nil }
func (s *InMemoryStorage) Engine() string               { return "memory" }

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

/* ------------------------------------------------------------------ */
/* SQLite implementation sketch — enable when persistence is wanted.   */
/*                                                                     */
/*   import _ "modernc.org/sqlite"                                    */
/*   db, err := sql.Open("sqlite", "lumen.db")                        */
/*   db.Exec(schema) // create tables from the list above             */
/*                                                                     */
/* Keep the StorageService interface stable; the frontend is          */
/* agnostic to the engine (its StorageService.engine field flips      */
/* from "memory" to "sqlite").                                        */
/* ------------------------------------------------------------------ */

var _ = fmt.Sprintf // keep fmt import for the sketch above
