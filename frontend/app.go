// Package frontend owns everything Wails: the bound App shell in this file
// plus the wails.Run bootstrap in main.go (Run). It is importable (plain
// `package frontend`, not `package main`) so the tiny root main.go can route
// into it — Go cannot import a `package main`, which is why the Wails shell
// lives here as a library instead of a second main package.
//
// Layering:
//   - backend/ (package backend) is pure Go: no wails imports, ctx passed
//     explicitly, AI streaming via callback. See backend/main.go.
//   - This file is the Wails adapter: it holds a *backend.Backend, captures
//     the runtime ctx at Startup, and exposes thin bindable methods whose
//     signatures stay JSON-safe (no ctx params, no channels — values that
//     cannot cross the JS boundary). Wails exposes them as
//     window.go.frontend.App.* (was window.go.main.App.* when App lived in
//     package main; see frontend/src/services/backend.ts).
package frontend

import (
	"context"
	"fmt"

	// backend: pure-Go bridge aggregated from models/ + services/.
	// `Conductino` matches `module Conductino` in go.mod — never the old
	// `github.com/lumen/desktop/...` path, which pointed at a nonexistent
	// module. models supplies the JSON wire types used below.
	"Conductino/backend"
	"Conductino/backend/models"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App is the single struct bound to Wails (see Run in main.go, Bind: app).
// It holds the Wails runtime ctx plus the pure-Go backend bridge; every
// method forwards to exactly one backend.Backend method.
type App struct {
	// ctx is captured in Startup and reused by methods that emit events.
	// Stored here (not taken as a parameter) because bindable methods must
	// stay JSON-serialisable — context values cannot cross to JS.
	ctx context.Context
	// backend is the pure-Go bridge (models + services aggregator).
	// All business logic lives behind it; this shell only translates the
	// boundary (stored ctx, event emission).
	backend *backend.Backend
}

// NewApp creates the Wails shell with default backend services.
// Called once by root main.go, then handed to Run.
func NewApp() *App {
	return &App{backend: backend.NewBackend()}
}

// Startup is called by Wails when the webview is ready. It saves ctx for
// later EventsEmit calls and initialises persistence via the bridge.
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
	_ = a.backend.Init(ctx)
	// Future: load sessions via backend → emit to UI with runtime.EventsEmit
}

// Shutdown is called by Wails on exit. Add flush logic behind
// backend.Shutdown when SQLite persistence lands.
func (a *App) Shutdown(ctx context.Context) {
	a.backend.Shutdown(ctx)
}

// Greet returns a greeting — kept from the default Wails scaffold so any
// existing template bindings keep working. Not part of the domain model.
func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, It's show time!", name)
}

/* --------------------------------------------------------------------
 * Bindable methods (UI: window.go.frontend.App.*).
 * Thin forwards: stored a.ctx in, backend result out. No logic here.
 * -------------------------------------------------------------------- */

// ShowContainingFolder reveals a document's folder in the OS file manager.
func (a *App) ShowContainingFolder(path string) (string, error) {
	return a.backend.ShowContainingFolder(path)
}

// ListLibraryTree returns the curated library tree for the Library rail view
// as ONE nested root node (with Children). Nil + nil means "no workspace yet".
func (a *App) ListLibraryTree() (*models.FileTreeNode, error) {
	return a.backend.ListLibraryTree()
}

// SetLibraryRoot repoints the library at a new directory. Exported (and
// bound) so a future UI flow can set the root without the dialog.
func (a *App) SetLibraryRoot(path string) {
	a.backend.SetLibraryRoot(path)
}

// LibraryRoot returns the absolute workspace root so the UI can tag opened
// documents with their opening folder (tasks.md 1.1 option b).
func (a *App) LibraryRoot() string {
	return a.backend.LibraryRoot()
}

// OpenFile opens one workspace file by its Path token and returns real
// extracted content for supported types (.txt/.md today). Unsupported types
// and escapes come back as errors — the UI falls back to its mock extract.
func (a *App) OpenFile(path string) (models.OpenedDocument, error) {
	return a.backend.OpenFile(path)
}

// GetSources returns persisted sources (empty on the in-memory mock).
func (a *App) GetSources() ([]models.Source, error) {
	return a.backend.GetSources(a.ctx)
}

// SaveWorkspace persists the current session snapshot.
func (a *App) SaveWorkspace() error {
	return a.backend.SaveWorkspace(a.ctx)
}

// ExtractDocument turns a raw Source into renderable blocks JSON.
func (a *App) ExtractDocument(source models.Source) (string, error) {
	return a.backend.ExtractDocument(a.ctx, source)
}

// StorageEngine reports the active persistence engine ("memory"/"sqlite").
func (a *App) StorageEngine() string {
	return a.backend.StorageEngine()
}

// StreamAIRequest runs an AI operation and streams progress units to the UI
// via runtime.EventsEmit(ctx, "ai://event", ev). Channels cannot cross the
// JS boundary, so streaming goes exclusively through EventsEmit — the pure
// backend.RunAI callback is wrapped here, at the Wails edge, which is the
// only place allowed to touch the runtime package.
func (a *App) StreamAIRequest(req models.AIRequest) error {
	a.backend.RunAI(a.ctx, req, func(ev models.AIEvent) {
		runtime.EventsEmit(a.ctx, "ai://event", ev)
	})
	return nil
}

// SelectFolder opens the OS folder dialog and repoints the workspace at the
// chosen directory, so the next ListWorkspace reads the newly picked folder.
// A cancelled dialog returns "" + nil and leaves the current root untouched.
// (Drive-by fixes while wiring this up: OpenDialogOptions needed its runtime
// qualifier, and the dialog call is OpenDirectoryDialog in Wails v2 — the
// pasted WithOptions name does not exist there.)
func (a *App) SelectFolder() (string, error) {
	path, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select folder to read from",
	})
	if err != nil || path == "" {
		return path, err
	}
	a.backend.SetLibraryRoot(path)
	return path, nil
}
