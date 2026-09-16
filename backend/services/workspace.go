package services

import (
	"context"

	// NOTE (import fix): was `github.com/lumen/desktop/backend/models`.
	// Fixed to `Conductino/backend/models` to match `module Conductino`
	// in go.mod. `models` provides the Source/FileTreeNode types used below.
	"Conductino/backend/models"
)

// WorkspaceService owns the persistent research workspace: the curated
// library view over the chosen folder, plus sessions, saved sources,
// summary documents and their relationships.
//
// Split from Filesystem on purpose: Filesystem is raw OS capability
// (ReadDir, recursive walk, reveal-in-OS, path resolution). Workspace
// OWNS the library tree the UI shows — root choice, nesting, and later
// persistence of that choice plus file↔document mapping — by composing
// a Filesystem, never the reverse. Disk mechanics stay in exactly one
// place; curation lives here.
//
// FUTURE INTEGRATION: persist via StorageService (SQLite), including the
// chosen library root so a restart reopens the same folder. The mock
// returns an empty set so the frontend falls back to its in-memory
// session state.
type WorkspaceService interface {
	GetSources(ctx context.Context) ([]models.Source, error)
	Save(ctx context.Context) error
	// LibraryTree returns the workspace folder as ONE nested root node
	// (with Children), matching the TS FileTreeNode shape. Nil + nil means
	// "no workspace yet" — empty state, not an error.
	LibraryTree() (*models.FileTreeNode, error)
	// SetLibraryRoot repoints the library at a new directory (absolute path
	// from the folder dialog). Empty paths are ignored (cancelled dialog).
	SetLibraryRoot(path string)
}

type Workspace struct {
	storage StorageService
	fs      *Filesystem
}

func NewWorkspace(fs *Filesystem) *Workspace {
	if fs == nil {
		fs = NewFilesystem("research_workspace")
	}
	return &Workspace{storage: NewStorage(), fs: fs}
}

func (w *Workspace) GetSources(_ context.Context) ([]models.Source, error) {
	return []models.Source{}, nil
}

func (w *Workspace) Save(_ context.Context) error {
	// Mock: record a snapshot marker. Real: serialise sessions/sources/
	// documents/changes into StorageService.
	return w.storage.Put("workspace.snapshot", "mock")
}

// LibraryTree forwards to the composed Filesystem walk.
func (w *Workspace) LibraryTree() (*models.FileTreeNode, error) {
	return w.fs.ListRoot()
}

// SetLibraryRoot repoints the composed Filesystem at the chosen folder.
func (w *Workspace) SetLibraryRoot(path string) {
	w.fs.SetRoot(path)
}
