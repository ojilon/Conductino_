// Package frontend owns everything Wails: the bound App shell in this file
// plus the wails.Run bootstrap in main.go (Run). Plain `package frontend`
// (not main) so the tiny root main.go can import it.
//
// Layering: backend/ is pure Go (no wails imports, ctx passed explicitly).
// This file is the adapter: it holds a *backend.Backend, captures the
// runtime ctx at Startup, and exposes thin bindable methods with JSON-safe
// signatures. Wails exposes them as window.go.frontend.App.* (see
// frontend/src/services/backend.ts). Only methods the UI actually calls
// are bound — dead surface was removed in the backend remake.
package frontend

import (
	"context"
	"fmt"

	"Conductino/backend"
	"Conductino/backend/model"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App is the Wails-bound shell: pure-Go Backend + runtime ctx from Startup.
type App struct {
	ctx     context.Context
	backend *backend.Backend
}

// NewApp constructs the shell with a fresh Backend.
func NewApp() *App {
	return &App{backend: backend.NewBackend()}
}

// Startup stores ctx for EventsEmit and forwards Init to Backend.
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
	if err := a.backend.Init(ctx); err != nil {
		fmt.Printf("backend init: %v\n", err)
	}
}

// Shutdown is kept so wails.Run OnShutdown always has a target.
func (a *App) Shutdown(_ context.Context) {}

// ListLibraryTree is the canonical library call. A nil node arrives in JS
// as null — the UI shows "Choose folder", not an error.
func (a *App) ListLibraryTree() (*model.Node, error) {
	return a.backend.ListLibraryTree()
}

// SelectFolder opens the OS folder dialog, repoints the library, and
// returns the absolute path ("" on cancel).
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

// ReadRawFile serves base64-encoded workspace bytes for renderers needing
// originals (PDF canvas leaf). Resolve-jailed; oversized files refused.
func (a *App) ReadRawFile(path string) (string, error) {
	return a.backend.ReadRawFile(path)
}

// OpenFile extracts a document. Content lands in the dump dir; the UI
// reads it back via ReadDumpFile (Doc.Dump). Classified failures arrive
// as a value with Reason set — never a fabricated document.
func (a *App) OpenFile(path string) (*model.Doc, error) {
	return a.backend.OpenFile(a.ctx, path)
}

// ReadDumpFile returns one extract_dump file's content by NAME (Doc.Dump).
func (a *App) ReadDumpFile(name string) (string, error) {
	return a.backend.ReadDumpFile(name)
}

// LibraryRoot returns the current library root path.
func (a *App) LibraryRoot() string {
	return a.backend.LibraryRoot()
}

// AIMeters reports AI session telemetry for the Settings dialog.
func (a *App) AIMeters() string {
	return a.backend.UsageMeters()
}

// StreamAIRequest runs an AI operation; progress streams back on the
// shared "ai://event" channel, demultiplexed by requestId.
func (a *App) StreamAIRequest(req model.AIRequest) error {
	return a.backend.RunAI(a.ctx, req, func(ev model.AIEvent) {
		runtime.EventsEmit(a.ctx, "ai://event", ev)
	})
}

// SetPrimarySummary marks a document as the workspace primary summary.
func (a *App) SetPrimarySummary(summaryID string) error {
	return a.backend.SetPrimarySummary(summaryID)
}

// WriteSummaryDOCX writes summary blocks as a .docx under the library root.
// Empty relPath auto-names from the first block's title.
func (a *App) WriteSummaryDOCX(relPath, blocksJSON string) (string, error) {
	return a.backend.WriteSummaryDOCX(relPath, blocksJSON)
}

// SaveThread upserts a chat thread.
func (a *App) SaveThread(t model.ChatThreadRecord) error {
	return a.backend.SaveThread(t)
}

// AppendChatMessage appends a message to a thread.
func (a *App) AppendChatMessage(msg model.ChatMessageRecord) error {
	return a.backend.AppendChatMessage(msg)
}

// ListThreads lists chat threads for a workspace.
func (a *App) ListThreads(workspaceID string) ([]model.ChatThreadRecord, error) {
	return a.backend.ListThreads(workspaceID)
}

// ListMessages lists messages in a thread.
func (a *App) ListMessages(threadID string) ([]model.ChatMessageRecord, error) {
	return a.backend.ListMessages(threadID)
}

// BuildContextPack assembles a budgeted context string server-side.
// rangeStart/rangeEnd are UTF-16 offsets, -1 = no range.
func (a *App) BuildContextPack(blocksJSON, title, blockID, selText string, rangeStart, rangeEnd, windowBlocks int, includeOutline bool) (string, error) {
	return a.backend.BuildContextPack(blocksJSON, title, blockID, selText, rangeStart, rangeEnd, windowBlocks, includeOutline)
}

// MatchSkills returns skill excerpts for an operation+intent as JSON.
func (a *App) MatchSkills(operation, intent string) (string, error) {
	return a.backend.MatchSkills(operation, intent)
}
