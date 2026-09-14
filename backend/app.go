package main

import (
	"context"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/lumen/desktop/backend/models"
	"github.com/lumen/desktop/backend/services"
)

// App is the Wails-bound application object. It owns the services and
// exposes thin, bindable methods. Business logic lives in the services,
// NOT here, and NEVER in the frontend glue.
type App struct {
	ctx     context.Context
	fs      *services.Filesystem
	storage *services.Storage
	docs    *services.Documents
	ai      *services.AI
	work    *services.Workspace
}

func NewApp() *App {
	return &App{
		fs:      services.NewFilesystem("research_workspace"),
		storage: services.NewStorage(),
		docs:    services.NewDocuments(),
		ai:      services.NewAI(),
		work:    services.NewWorkspace(),
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	_ = a.storage.Init(ctx)
	// Future: a.Workspace.LoadSessions() → emit to frontend via runtime.EventsEmit
}

func (a *App) shutdown(_ context.Context) {}

/* --------------------------------------------------------------------
 * Bindable methods (frontend: window.go.main.App.*)
 * Keep these thin — they forward to services.
 * -------------------------------------------------------------------- */

// ShowContainingFolder reveals a document's folder in the OS file manager.
func (a *App) ShowContainingFolder(path string) (string, error) {
	return a.fs.ShowContainingFolder(path)
}

// ListWorkspace returns the workspace file tree for the reader sidebar.
func (a *App) ListWorkspace() ([]models.FileTreeNode, error) {
	return a.fs.ListRoot()
}

// GetSources returns persisted sources (SQLite-backed once storage is enabled).
func (a *App) GetSources(ctx context.Context) ([]models.Source, error) {
	return a.work.GetSources(ctx)
}

// SaveWorkspace persists the current session snapshot.
func (a *App) SaveWorkspace(ctx context.Context) error {
	return a.work.Save(ctx)
}

// StreamAIRequest runs an AI operation on the Go side and streams progress
// events to the frontend. The frontend's AIProvider interface can wrap this
// transport (see docs/ai-integration.md §"Go-side provider").
func (a *App) StreamAIRequest(ctx context.Context, req models.AIRequest, events chan<- models.AIEvent) {
	a.ai.Run(ctx, req, func(ev models.AIEvent) {
		runtime.EventsEmit(ctx, "ai://event", ev)
		events <- ev
	})
}
