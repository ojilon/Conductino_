package services

import (
	"context"
	"path/filepath"
	"time"

	"Conductino/backend/models"
)

// WorkspaceService owns the persistent research workspace: the curated
// library view over the chosen folder, plus sessions, saved sources,
// summary documents and their relationships.
type WorkspaceService interface {
	GetSources(ctx context.Context) ([]models.Source, error)
	Save(ctx context.Context) error
	LibraryTree() (*models.FileTreeNode, error)
	SetLibraryRoot(path string)
	LibraryRoot() string
}

type Workspace struct {
	storage     StorageService
	fs          *Filesystem
	workspaceID string
}

func NewWorkspace(fs *Filesystem, storage StorageService) *Workspace {
	if fs == nil {
		fs = NewFilesystem("research_workspace")
	}
	if storage == nil {
		storage = NewStorage()
	}
	return &Workspace{storage: storage, fs: fs, workspaceID: "ws-default"}
}

func (w *Workspace) GetSources(_ context.Context) ([]models.Source, error) {
	return []models.Source{}, nil
}

func (w *Workspace) Save(_ context.Context) error {
	root := w.fs.Root()
	_ = w.storage.SetSetting("last_library_root", root)
	rec := WorkspaceRecord{
		ID:        w.workspaceID,
		RootPath:  root,
		Label:     filepath.Base(root),
		UpdatedAt: time.Now().UnixMilli(),
	}
	if existing, _ := w.storage.GetWorkspace(w.workspaceID); existing != nil {
		rec.PrimarySummaryID = existing.PrimarySummaryID
		if existing.Label != "" {
			rec.Label = existing.Label
		}
	}
	return w.storage.UpsertWorkspace(rec)
}

func (w *Workspace) LibraryTree() (*models.FileTreeNode, error) {
	return w.fs.ListRoot()
}

func (w *Workspace) SetLibraryRoot(path string) {
	w.fs.SetRoot(path)
	if path == "" {
		return
	}
	if existing, _ := w.storage.GetWorkspaceByRoot(path); existing != nil {
		w.workspaceID = existing.ID
		_ = w.storage.SetSetting("last_library_root", path)
		return
	}
	id := "ws-" + filepath.Base(path)
	if existing, _ := w.storage.GetWorkspace(id); existing != nil && existing.RootPath != path {
		id = "ws-" + filepath.Base(path) + "-" + time.Now().Format("150405")
	}
	w.workspaceID = id
	_ = w.storage.UpsertWorkspace(WorkspaceRecord{
		ID:        id,
		RootPath:  path,
		Label:     filepath.Base(path),
		UpdatedAt: time.Now().UnixMilli(),
	})
	_ = w.storage.SetSetting("last_library_root", path)
}

func (w *Workspace) LibraryRoot() string {
	return w.fs.Root()
}

func (w *Workspace) WorkspaceID() string {
	return w.workspaceID
}

func (w *Workspace) SetPrimarySummary(summaryID string) error {
	ws, err := w.storage.GetWorkspace(w.workspaceID)
	if err != nil {
		return err
	}
	if ws == nil {
		ws = &WorkspaceRecord{ID: w.workspaceID, RootPath: w.fs.Root()}
	}
	ws.PrimarySummaryID = summaryID
	ws.UpdatedAt = time.Now().UnixMilli()
	return w.storage.UpsertWorkspace(*ws)
}
