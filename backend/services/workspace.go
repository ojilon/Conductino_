package services

import (
	"context"

	"github.com/lumen/desktop/backend/models"
)

// WorkspaceService owns the persistent research workspace:
// sessions, saved sources, summary documents and their relationships.
//
// FUTURE INTEGRATION: persist via StorageService (SQLite). The mock
// returns an empty set so the frontend falls back to its in-memory
// session state.
type WorkspaceService interface {
	GetSources(ctx context.Context) ([]models.Source, error)
	Save(ctx context.Context) error
}

type Workspace struct {
	storage StorageService
}

func NewWorkspace() *Workspace {
	return &Workspace{storage: NewStorage()}
}

func (w *Workspace) GetSources(_ context.Context) ([]models.Source, error) {
	return []models.Source{}, nil
}

func (w *Workspace) Save(_ context.Context) error {
	// Mock: record a snapshot marker. Real: serialise sessions/sources/
	// documents/changes into StorageService.
	return w.storage.Put("workspace.snapshot", "mock")
}
