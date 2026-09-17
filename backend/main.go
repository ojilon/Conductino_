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
	if root, ok := b.storage.GetSetting("last_library_root"); ok && root != "" {
		_ = b.work.SetRoot(root)
	}
	return nil
}

func (b *Backend) ListTree(ctx context.Context) ([]models.FileTreeNode, error) {
	return b.work.ListTree(ctx)
}

func (b *Backend) OpenFile(ctx context.Context, path string) (*models.Document, error) {
	return b.docs.OpenFile(ctx, path)
}

func (b *Backend) ResolvePath(ctx context.Context, path string) (string, error) {
	return b.fs.Resolve(path)
}

func (b *Backend) SetLibraryRoot(ctx context.Context, root string) error {
	if err := b.work.SetRoot(root); err != nil {
		return err
	}
	_ = b.storage.SetSetting("last_library_root", root)
	return nil
}

func (b *Backend) LibraryRoot() string {
	return b.work.Root()
}

func (b *Backend) WorkspaceID() string {
	return b.work.ID()
}

func (b *Backend) StorageEngine() string {
	return b.storage.Engine()
}

func (b *Backend) RunAI(ctx context.Context, req models.AIRequest, sink AIEventSink) error {
	// Phase 6: if chat omitted summary content, load primary summary from DB.
	if req.SummaryContent == "" && req.WorkspaceID != "" {
		wsID := req.WorkspaceID
		if sum, err := b.storage.GetPrimarySummary(wsID); err == nil && sum != nil {
			req.SummaryContent = sum.Body
			if req.PrimarySummaryID == "" {
				req.PrimarySummaryID = sum.ID
			}
		}
	}
	return b.ai.Run(ctx, req, sink)
}

func (b *Backend) SetPrimarySummary(summaryID string) error {
	return b.work.SetPrimarySummary(summaryID)
}

func (b *Backend) SaveDocument(doc services.DocumentRecord) error {
	doc.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	return b.storage.UpsertDocument(doc)
}

func (b *Backend) LoadDocument(id string) (*services.DocumentRecord, error) {
	return b.storage.GetDocument(id)
}

func (b *Backend) ListDocuments(workspaceID string) ([]services.DocumentRecord, error) {
	return b.storage.ListDocuments(workspaceID)
}

func (b *Backend) SaveChange(ch services.ChangeRecord) error {
	ch.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	return b.storage.UpsertChange(ch)
}

func (b *Backend) ListPendingChanges(documentID string) ([]services.ChangeRecord, error) {
	return b.storage.ListPendingChanges(documentID)
}

func (b *Backend) SaveThread(t services.ChatThreadRecord) error {
	t.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	return b.storage.UpsertThread(t)
}

func (b *Backend) AppendChatMessage(msg services.ChatMessageRecord) error {
	msg.CreatedAt = time.Now().UTC().Format(time.RFC3339)
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
		workspaceID = b.work.ID()
	}
	return b.storage.GetPrimarySummary(workspaceID)
}

// Convenience for absolute paths under the library root.
func (b *Backend) JoinUnderRoot(parts ...string) string {
	root := b.work.Root()
	args := append([]string{root}, parts...)
	return filepath.Join(args...)
}

func (b *Backend) IsUnderRoot(path string) bool {
	root := b.work.Root()
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
