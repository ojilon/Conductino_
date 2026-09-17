// Package backend is pure, independent Go — it knows nothing about Wails.
//
// Architecture (why it looks like this):
//   - frontend/ (package frontend, imported by root main.go) owns EVERYTHING
//     Wails: wails.Run options, asset embedding, lifecycle hooks, and
//     runtime.EventsEmit. It is the only place that imports
//     github.com/wailsapp/wails/... .
//   - backend/ owns everything else: domain types (models/) and capabilities
//     (services/: filesystem, storage, documents, ai, workspace). No file
//     under backend/ may import a Wails package, so the whole tree stays
//     testable with plain `go test` and reusable outside the desktop app.
//   - This file is the ONLY bridge between the two sides: it aggregates one
//     instance of each service into Backend and exposes thin forwarding
//     methods. The Wails shell (frontend/app.go) holds a *Backend and calls
//     these methods; each call passes ctx explicitly instead of storing it,
//     because only the Wails layer is allowed to own the runtime context.
//
// Import rule: go.mod declares `module Conductino`, so every internal import
// MUST be `Conductino/backend/...`. The old path
// `github.com/lumen/desktop/backend/models` referenced a module that does not
// exist and broke compilation — it is fixed everywhere under backend/.
package backend

import (
	"context"
	"path/filepath"
	"strings"

	// models: shared wire types (the "what" crosses to the UI as JSON).
	// Mirrors frontend/src/types/domain.ts — keep the two in sync manually.
	"Conductino/backend/models"
	// services: capability implementations (the "how" stays behind the bridge).
	"Conductino/backend/services"
)

// AIEventSink receives one streaming unit of an AI operation.
// It is an alias of the services-level sink so the Wails shell
// (frontend/app.go) can forward it straight to runtime.EventsEmit
// without importing the services package itself.
type AIEventSink = services.AIEventSink

// Backend aggregates all backend services into the single struct the
// Wails shell (frontend/app.go: App) holds. It keeps NO business logic:
// every method below forwards to exactly one service:
//
//   - models/   — pure data shapes (Source, FileTreeNode, AIRequest, AIEvent).
//   - services/ — filesystem, storage, documents, ai, workspace. Each service
//     exposes an interface + a mock implementation, so a real implementation
//     (SQLite, real AI provider) can be swapped in without touching callers.
type Backend struct {
	// One field per submodule — the "bridge" wiring. Services that still
	// have only one implementation keep their concrete type (e.g. Storage);
	// AI is held as the AIService interface so providers swap without
	// touching callers.
	// NOTE: the Filesystem instance is shared — Workspace composes this same
	// instance for the library tree, so the dialog, the tree, reveals and
	// file opens can never disagree about which folder is current.
	fs      *services.Filesystem      // OS capability: walk, resolve, reveal-in-folder
	storage *services.InMemoryStorage // persistence (memory today, SQLite tomorrow)
	docs    *services.Documents       // format extraction (PDF/DOCX/HTML/TXT → blocks)
	ai      services.AIService        // model access: Gemini via backend/services/ai/ (shim: services.NewAI)
	work    *services.Workspace       // library tree + sessions / saved sources / summary docs
}

// NewBackend builds the bridge with default service implementations.
// Called once by the Wails shell (frontend/app.go: NewApp).
func NewBackend() *Backend {
	fs := services.NewFilesystem("research_workspace")
	return &Backend{
		fs:      fs,
		storage: services.NewStorage(),
		docs:    services.NewDocuments(),
		ai:      services.NewAI(),
		work:    services.NewWorkspace(fs),
	}
}

// Init prepares persistence. The Wails shell calls it from Startup with the
// runtime context. Pure Go — no runtime.EventsEmit here; event emission is
// the frontend package's job.
func (b *Backend) Init(ctx context.Context) error {
	return b.storage.Init(ctx)
}

// Shutdown releases backend resources. No-op while storage is in-memory;
// add flush/close logic here when SQLite lands.
func (b *Backend) Shutdown(_ context.Context) {}

/* --------------------------------------------------------------------
 * Bridge methods — thin forwards, one service each.
 * ctx is always an explicit parameter (pure Go style): the caller
 * (frontend/app.go) supplies the ctx it captured at Startup.
 * -------------------------------------------------------------------- */

// ShowContainingFolder reveals a document's folder in the OS file manager.
// Raw OS capability — stays on Filesystem, not the Workspace curation layer.
// Returns the revealed directory for the UI note.
func (b *Backend) ShowContainingFolder(path string) (string, error) {
	return b.fs.ShowContainingFolder(path)
}

// ListLibraryTree returns the curated library tree for the Library rail view
// as ONE nested root node (with Children). Owned by Workspace (which composes
// Filesystem); a missing folder yields nil + nil so the UI shows its empty
// state instead of an error.
func (b *Backend) ListLibraryTree() (*models.FileTreeNode, error) {
	return b.work.LibraryTree()
}

// SetLibraryRoot repoints the library at a new directory (absolute path from
// the folder dialog). The Wails shell calls this right after the dialog
// returns, so the next ListLibraryTree reads the newly chosen folder.
func (b *Backend) SetLibraryRoot(path string) {
	b.work.SetLibraryRoot(path)
}

// OpenFile opens one workspace file by its Path token (root-relative, as
// carried on FileTreeNode). The token is resolved + containment-checked
// against the workspace root (Filesystem.Resolve — escapes rejected), then
// dispatched by extension in Documents: plain text reads for real today,
// other types report ReasonUnsupported.
//
// Typed failures (tasks.md 1.2): classified failures come back as a VALUE
// with Reason set and BlocksJSON empty — never a Go error — so the UI shows
// an honest per-reason message and opens nothing. A Go error return is
// reserved for truly unexpected bridge failures.
// The returned document is tagged with the absolute opening root
// (tasks.md 1.1 option b) so the UI can detect stale tabs after a switch.
func (b *Backend) OpenFile(relPath string) (models.OpenedDocument, error) {
	root := b.fs.Root()
	title := strings.TrimSuffix(filepath.Base(relPath), filepath.Ext(relPath))
	if title == "" || title == "." {
		title = relPath
	}
	abs, err := b.fs.Resolve(relPath)
	if err != nil {
		reason, detail := services.ReasonOf(err)
		// Resolve only fails on escapes / bad roots: a containment
		// rejection is a refusal, not a missing extractor — surface it as
		// permission_denied, never as "unsupported" (which the UI treats
		// leniently) and never as a fabricated document.
		if reason == models.ReasonUnsupported || strings.Contains(detail, "escapes workspace") {
			reason = models.ReasonPermissionDenied
		}
		if reason != models.ReasonNotFound && reason != models.ReasonPermissionDenied {
			reason = models.ReasonParseError
		}
		return models.OpenedDocument{Title: title, Reason: string(reason), Detail: detail, Root: root}, nil
	}
	opened, err := b.docs.OpenFile(abs)
	if err != nil {
		reason, detail := services.ReasonOf(err)
		return models.OpenedDocument{Title: title, Reason: string(reason), Detail: detail, Root: root}, nil
	}
	opened.Root = root
	return opened, nil
}

// LibraryRoot returns the absolute workspace root so the UI can tag opened
// documents and detect stale tabs after a folder switch (tasks.md 1.1 b).
func (b *Backend) LibraryRoot() string {
	return b.work.LibraryRoot()
}

// SetWorkspaceRoot repoints the workspace at a new directory (absolute path
// from the folder dialog). The Wails shell calls this right after the dialog
// returns, so the next ListWorkspace reads the newly chosen folder.
func (b *Backend) SetWorkspaceRoot(path string) {
	b.fs.SetRoot(path)
}

// GetSources returns persisted sources (empty while storage is the
// in-memory mock; SQLite-backed once storage is enabled).
func (b *Backend) GetSources(ctx context.Context) ([]models.Source, error) {
	return b.work.GetSources(ctx)
}

// SaveWorkspace persists the current session snapshot via Workspace,
// which in turn writes through StorageService.
func (b *Backend) SaveWorkspace(ctx context.Context) error {
	return b.work.Save(ctx)
}

// ExtractDocument turns a raw Source into renderable blocks via the
// Documents service, so the reader renders files without ever touching
// the filesystem itself. Returns the blocks JSON (page count is dropped
// at this boundary; the UI only needs the blocks today).
func (b *Backend) ExtractDocument(ctx context.Context, source models.Source) (string, error) {
	blocksJSON, _, err := b.docs.Extract(ctx, source)
	return blocksJSON, err
}

// StorageEngine reports which persistence engine is active
// ("memory" today, "sqlite" in the future) for the settings UI.
func (b *Backend) StorageEngine() string {
	return b.storage.Engine()
}

// RunAI runs an AI operation and pushes each progress unit into sink.
// Pure Go: it does NOT emit Wails events itself — the Wails shell wraps
// sink with runtime.EventsEmit (see frontend/app.go: StreamAIRequest).
func (b *Backend) RunAI(ctx context.Context, req models.AIRequest, sink AIEventSink) {
	b.ai.Run(ctx, req, sink)
}
