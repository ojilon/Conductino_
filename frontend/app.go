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
	"Conductino/backend/services"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App is the Wails-bound shell. It holds the pure-Go Backend and the
// runtime context captured at Startup.
type App struct {
	ctx     context.Context
	backend *backend.Backend
}

// NewApp constructs the shell with a fresh Backend.
func NewApp() *App {
	return &App{backend: backend.NewBackend()}
}

// Startup is called by Wails once the runtime is ready. It stores ctx so
// later methods can call runtime.EventsEmit and forwards Init to Backend.
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
	if err := a.backend.Init(ctx); err != nil {
		fmt.Printf("backend init: %v\n", err)
	}
}

// Shutdown releases backend resources (SQLite handle) on app exit.
func (a *App) Shutdown(_ context.Context) {
	// Backend has no Close exposed; nothing to flush yet (SQLite closes on
	// process exit). Kept so wails.Run OnShutdown always has a target.
}

// ListTree returns the workspace file tree (legacy slice wrapper).
// New code should call ListLibraryTree (single nested root node).
func (a *App) ListTree() ([]models.FileTreeNode, error) {
	return a.backend.ListTree(a.ctx)
}

// ListLibraryTree is the canonical library call the TS frontend binds to
// (window.go.frontend.App.ListLibraryTree). A nil *FileTreeNode arrives in
// JS as null — the UI shows "Choose folder", not an error.
func (a *App) ListLibraryTree() (*models.FileTreeNode, error) {
	return a.backend.ListLibraryTree()
}

// SelectFolder opens the OS folder dialog, repoints the library at the
// picked folder, and returns its absolute path ("" on cancel). The TS
// caller re-reads library.list() + libraryRoot() afterwards.
func (a *App) SelectFolder() (string, error) {
	picked, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Choose research folder",
	})
	if err != nil {
		return "", err
	}
	if picked == "" {
		return "", nil // cancelled — caller keeps the current tree
	}
	if err := a.backend.SetLibraryRoot(a.ctx, picked); err != nil {
		return "", err
	}
	return picked, nil
}

// ShowContainingFolder reveals a workspace path in the OS file manager.
func (a *App) ShowContainingFolder(path string) (string, error) {
	return a.backend.ShowContainingFolder(path)
}

// OpenFile extracts a document from a path under the library root.
// Returns OpenedDocument (BlocksJSON wire payload + typed Reason on failure),
// matching Documents.OpenFile — never a fabricated document.
func (a *App) OpenFile(path string) (*models.OpenedDocument, error) {
	return a.backend.OpenFile(a.ctx, path)
}

// ResolvePath resolves a relative path against the library root.
func (a *App) ResolvePath(path string) (string, error) {
	return a.backend.ResolvePath(a.ctx, path)
}

// SetLibraryRoot binds the workspace to a folder on disk.
func (a *App) SetLibraryRoot(root string) error {
	return a.backend.SetLibraryRoot(a.ctx, root)
}

// LibraryRoot returns the current library root path.
func (a *App) LibraryRoot() string {
	return a.backend.LibraryRoot()
}

// WorkspaceID returns the current workspace id.
func (a *App) WorkspaceID() string {
	return a.backend.WorkspaceID()
}

// StorageEngine reports the active persistence engine ("memory"/"sqlite").
func (a *App) StorageEngine() string {
	return a.backend.StorageEngine()
}

// RunAI streams an AI operation. Events are emitted as "ai:event" (legacy).
// New code should call StreamAIRequest, which emits "ai://event" — the
// channel services/ai.ts actually subscribes to.
func (a *App) RunAI(req models.AIRequest) error {
	return a.backend.RunAI(a.ctx, req, func(ev models.AIEvent) {
		runtime.EventsEmit(a.ctx, "ai:event", ev)
	})
}

// StreamAIRequest is the AI bridge services/ai.ts binds to
// (window.go.frontend.App.StreamAIRequest). Progress + results stream back
// on the shared "ai://event" channel, demultiplexed by requestId.
func (a *App) StreamAIRequest(req models.AIRequest) error {
	return a.backend.RunAI(a.ctx, req, func(ev models.AIEvent) {
		runtime.EventsEmit(a.ctx, "ai://event", ev)
	})
}

// SetPrimarySummary marks a document as the workspace primary summary.
func (a *App) SetPrimarySummary(summaryID string) error {
	return a.backend.SetPrimarySummary(summaryID)
}

// SaveDocument persists a document record.
func (a *App) SaveDocument(doc services.DocumentRecord) error {
	return a.backend.SaveDocument(doc)
}

// LoadDocument loads a document by id.
func (a *App) LoadDocument(id string) (*services.DocumentRecord, error) {
	return a.backend.LoadDocument(id)
}

// ListDocuments lists documents for a workspace.
func (a *App) ListDocuments(workspaceID string) ([]services.DocumentRecord, error) {
	return a.backend.ListDocuments(workspaceID)
}

// SaveChange persists a document change.
func (a *App) SaveChange(ch services.ChangeRecord) error {
	return a.backend.SaveChange(ch)
}

// ListPendingChanges lists pending changes for a document.
func (a *App) ListPendingChanges(documentID string) ([]services.ChangeRecord, error) {
	return a.backend.ListPendingChanges(documentID)
}

// SaveThread upserts a chat thread.
func (a *App) SaveThread(t services.ChatThreadRecord) error {
	return a.backend.SaveThread(t)
}

// AppendChatMessage appends a message to a thread.
func (a *App) AppendChatMessage(msg services.ChatMessageRecord) error {
	return a.backend.AppendChatMessage(msg)
}

// LoadThread loads a chat thread by id.
func (a *App) LoadThread(id string) (*services.ChatThreadRecord, error) {
	return a.backend.LoadThread(id)
}

// ListThreads lists chat threads for a workspace.
func (a *App) ListThreads(workspaceID string) ([]services.ChatThreadRecord, error) {
	return a.backend.ListThreads(workspaceID)
}

// ListMessages lists messages in a thread.
func (a *App) ListMessages(threadID string) ([]services.ChatMessageRecord, error) {
	return a.backend.ListMessages(threadID)
}

// LoadPrimarySummary returns the primary summary document for a workspace.
func (a *App) LoadPrimarySummary(workspaceID string) (*services.DocumentRecord, error) {
	return a.backend.LoadPrimarySummary(workspaceID)
}
