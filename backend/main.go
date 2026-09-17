// Package backend is pure, independent Go — it knows nothing about Wails.
package backend

import (
	"context"
	"path/filepath"
	"strings"
	"time"

	"Conductino/backend/models"
	"Conductino/backend/services"
)

type AIEventSink = services.AIEventSink

type Backend struct {
	fs      *services.Filesystem
	storage services.StorageService
	docs    *services.Documents
	ai      services.AIService
	work    *services.Workspace
}

func NewBackend() *Backend {
	fs := services.NewFilesystem("research_workspace")
	docs := services.NewDocuments()
	store := services.OpenDefaultStorage(context.Background())
	return &Backend{
		fs:      fs,
		storage: store,
		docs:    docs,
		ai:      services.NewAI(fs, docs),
		work:    services.NewWorkspace(fs, store),
	}
}

func (b *Backend) Init(ctx context.Context) error {
	if err := b.storage.Init(ctx); err != nil {
		return err
	}
	// Restore last library root so the next session opens the same folder.
	// Workspace.SetLibraryRoot replays the stored root (no error return).
	if root, ok := b.storage.GetSetting("last_library_root"); ok && root != "" {
		b.work.SetLibraryRoot(root)
	}
	return nil
}

func (b *Backend) ListTree(_ context.Context) ([]models.FileTreeNode, error) {
	// Legacy slice wrapper kept for compat: ListLibraryTree is the canonical
	// call (single nested root node, matching the TS FileTreeNode shape).
	root, err := b.ListLibraryTree()
	if err != nil {
		return nil, err
	}
	if root == nil {
		return []models.FileTreeNode{}, nil
	}
	return []models.FileTreeNode{*root}, nil
}

// ListLibraryTree is the canonical library call bound as
// window.go.frontend.App.ListLibraryTree. It returns ONE nested root node
// (or nil when no folder is mounted — the UI shows "Choose folder", not an
// error). The old ListTree slice wrapper above stays for compat.
func (b *Backend) ListLibraryTree() (*models.FileTreeNode, error) {
	return b.work.LibraryTree()
}

func (b *Backend) OpenFile(_ context.Context, path string) (*models.OpenedDocument, error) {
	// Typed-failure contract (tasks.md 1.2): classified file problems are
	// returned as a VALUE with Reason set — never a Go error — so the UI can
	// tell "unsupported type" apart from real read failures without
	// string-matching. Only unexpected bridge failures return an error.
	// Root always tags the opening folder (tasks.md 1.1 option b).
	root := b.fs.Root()
	abs, err := b.fs.Resolve(path)
	if err != nil {
		reason, detail := services.ReasonOf(err)
		// Resolve errors (e.g. path escapes workspace) have no OpenError
		// wrapper, so ReasonOf falls back to parse_error — still a value.
		return &models.OpenedDocument{Root: root, Reason: string(reason), Detail: detail}, nil
	}
	opened, err := b.docs.OpenFile(abs)
	if err != nil {
		if reason, detail := services.ReasonOf(err); reason != "" {
			opened.Reason = string(reason)
			opened.Detail = detail
			opened.Root = root
			return &opened, nil
		}
		return nil, err
	}
	opened.Root = root
	return &opened, nil
}

// ShowContainingFolder reveals a workspace path in the OS file manager.
// Resolution + containment go through Filesystem.Resolve, so callers can
// never point the OS outside the open folder.
func (b *Backend) ShowContainingFolder(path string) (string, error) {
	return b.fs.ShowContainingFolder(path)
}

func (b *Backend) ResolvePath(_ context.Context, path string) (string, error) {
	return b.fs.Resolve(path)
}

func (b *Backend) SetLibraryRoot(_ context.Context, root string) error {
	// Workspace.SetLibraryRoot never fails (empty path is ignored); persist
	// the choice so Init can restore it next launch.
	b.work.SetLibraryRoot(root)
	_ = b.storage.SetSetting("last_library_root", root)
	return nil
}

func (b *Backend) LibraryRoot() string {
	return b.work.LibraryRoot()
}

func (b *Backend) WorkspaceID() string {
	return b.work.WorkspaceID()
}

func (b *Backend) StorageEngine() string {
	return b.storage.Engine()
}

func (b *Backend) RunAI(ctx context.Context, req models.AIRequest, sink AIEventSink) error {
	// Phase 6: if chat omitted summary content, load primary summary from DB.
	// The summary snapshot is the BlocksJSON wire payload — the tool host
	// truncates/proposes from it, so no format conversion happens here.
	if req.SummaryContent == "" && req.WorkspaceID != "" {
		wsID := req.WorkspaceID
		if sum, err := b.storage.GetPrimarySummary(wsID); err == nil && sum != nil {
			req.SummaryContent = sum.BlocksJSON
			if req.PrimarySummaryID == "" {
				req.PrimarySummaryID = sum.ID
			}
		}
	}
	// AIService.Run streams via sink and reports failures as error EVENTS,
	// not Go errors — so always return nil after dispatch.
	b.ai.Run(ctx, req, sink)
	return nil
}

func (b *Backend) SetPrimarySummary(summaryID string) error {
	return b.work.SetPrimarySummary(summaryID)
}

func (b *Backend) SaveDocument(doc services.DocumentRecord) error {
	// Timestamps are Unix millis (INTEGER in SQLite); zero is filled by UpsertDocument.
	doc.UpdatedAt = time.Now().UnixMilli()
	return b.storage.UpsertDocument(doc)
}

func (b *Backend) LoadDocument(id string) (*services.DocumentRecord, error) {
	return b.storage.GetDocument(id)
}

func (b *Backend) ListDocuments(workspaceID string) ([]services.DocumentRecord, error) {
	return b.storage.ListDocuments(workspaceID)
}

func (b *Backend) SaveChange(ch services.ChangeRecord) error {
	// ChangeRecord tracks CreatedAt (not UpdatedAt) — UpsertChange also defaults it.
	ch.CreatedAt = time.Now().UnixMilli()
	return b.storage.UpsertChange(ch)
}

func (b *Backend) ListPendingChanges(documentID string) ([]services.ChangeRecord, error) {
	return b.storage.ListPendingChanges(documentID)
}

func (b *Backend) SaveThread(t services.ChatThreadRecord) error {
	// Threads carry both CreatedAt/UpdatedAt millis; only bump UpdatedAt here
	// so the original creation time survives re-saves.
	t.UpdatedAt = time.Now().UnixMilli()
	return b.storage.UpsertThread(t)
}

func (b *Backend) AppendChatMessage(msg services.ChatMessageRecord) error {
	// Messages are append-only; stamp CreatedAt millis (UpsertThread bumps the parent).
	msg.CreatedAt = time.Now().UnixMilli()
	return b.storage.AppendMessage(msg)
}

func (b *Backend) LoadThread(id string) (*services.ChatThreadRecord, error) {
	return b.storage.GetThread(id)
}

func (b *Backend) ListThreads(workspaceID string) ([]services.ChatThreadRecord, error) {
	return b.storage.ListThreads(workspaceID)
}

func (b *Backend) ListMessages(threadID string) ([]services.ChatMessageRecord, error) {
	return b.storage.ListMessages(threadID)
}

func (b *Backend) LoadPrimarySummary(workspaceID string) (*services.DocumentRecord, error) {
	if workspaceID == "" {
		workspaceID = b.work.WorkspaceID()
	}
	return b.storage.GetPrimarySummary(workspaceID)
}

// Convenience for absolute paths under the library root.
func (b *Backend) JoinUnderRoot(parts ...string) string {
	root := b.work.LibraryRoot()
	args := append([]string{root}, parts...)
	return filepath.Join(args...)
}

func (b *Backend) IsUnderRoot(path string) bool {
	root := b.work.LibraryRoot()
	abs, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(root, abs)
	if err != nil {
		return false
	}
	return !strings.HasPrefix(rel, "..")
}
