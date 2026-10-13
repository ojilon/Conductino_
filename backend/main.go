// Package backend is pure Go — no Wails imports. One Backend struct wires
// the small packages together: list (library root + tree), read/save
// (documents), store (SQLite/memory), ai (models), mirror (summary working
// copies). Methods are thin and direct; the real logic lives in the
// packages, tested without any UI.
package backend

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"Conductino/backend/ai"
	"Conductino/backend/apppaths"
	"Conductino/backend/list"
	"Conductino/backend/mirror"
	"Conductino/backend/model"
	"Conductino/backend/read"
	"Conductino/backend/save"
	"Conductino/backend/store"
)

type Backend struct {
	lib     *list.Library
	storage store.StorageService
	ai      *ai.GeminiService
	mirrors *mirror.Store
	// workspaceID is the storage id for the current root ("ws-<folder>").
	workspaceID string
}

// opener adapts read.OpenDirect to the tools.FileOpener interface the AI
// loop takes (absolute path in, already Resolve-checked by the host).
type opener struct{}

func (opener) OpenFile(absPath, relKey string) (model.Doc, *model.Fail) {
	return read.OpenDirect(absPath, relKey, "")
}

func NewBackend() *Backend {
	lib := list.NewLibrary("research_workspace")
	storage := store.OpenDefaultStorage(context.Background())
	return &Backend{
		lib:         lib,
		storage:     storage,
		ai:          ai.NewWithTools(lib, opener{}),
		mirrors:     mirror.New(mirror.DefaultDir()),
		workspaceID: "ws-default",
	}
}

func (b *Backend) Init(ctx context.Context) error {
	if err := apppaths.EnsureDataRoot(); err != nil {
		fmt.Printf("appdata init: %v\n", err)
	}
	if err := b.storage.Init(ctx); err != nil {
		return err
	}
	// Restore last library root so the next session opens the same folder.
	if root, ok := b.storage.GetSetting("last_library_root"); ok && root != "" {
		b.lib.SetRoot(root)
		b.adoptWorkspaceID(root)
	}
	// Scratch-mirror hygiene: drop mirrors untouched for 7 days.
	// Best-effort — a failure here must never block startup.
	_ = b.mirrors.Sweep(7 * 24 * time.Hour)
	return nil
}

// adoptWorkspaceID finds or creates the storage id for root. Same folder
// keeps its id (threads survive restarts); a new folder gets ws-<name>.
func (b *Backend) adoptWorkspaceID(root string) {
	if existing, _ := b.storage.GetWorkspaceByRoot(root); existing != nil {
		b.workspaceID = existing.ID
		return
	}
	id := "ws-" + filepath.Base(root)
	if existing, _ := b.storage.GetWorkspace(id); existing != nil && existing.RootPath != root {
		id = "ws-" + filepath.Base(root) + "-" + time.Now().Format("150405")
	}
	b.workspaceID = id
	_ = b.storage.UpsertWorkspace(model.WorkspaceRecord{
		ID: id, RootPath: root, Label: filepath.Base(root), UpdatedAt: time.Now().UnixMilli(),
	})
}

// BuildContextPack assembles a budgeted context string server-side.
// The UI keeps rendering; budgeting lives behind this API.
func (b *Backend) BuildContextPack(blocksJSON, title, blockID, selText string, rangeStart, rangeEnd, windowBlocks int, includeOutline bool) (string, error) {
	return read.BuildContextPack(blocksJSON, read.PackOpts{
		Title: title, BlockID: blockID, SelText: selText,
		RangeStart: rangeStart, RangeEnd: rangeEnd,
		WindowBlocks: windowBlocks, IncludeOutline: includeOutline,
	})
}

// MatchSkills returns skill excerpts for an operation+intent as JSON.
func (b *Backend) MatchSkills(operation, intent string) (string, error) {
	return b.ai.MatchedSkills(operation, intent), nil
}

// ListLibraryTree returns ONE nested root node (nil = no folder mounted;
// the UI shows "Choose folder", not an error).
func (b *Backend) ListLibraryTree() (*model.Node, error) {
	n, fail := b.lib.Tree()
	if fail != nil {
		return nil, fmt.Errorf("%s", fail.Error())
	}
	return n, nil
}

// OpenFile extracts one workspace file. Classified problems come back as a
// VALUE with Reason set (never a Go error) so the UI tells "unsupported
// type" apart from real failures without string-matching. Content ALWAYS
// lands in the dump dir too (Dump name on the value) — the UI reads it
// back via ReadDumpFile instead of holding every file in memory twice.
func (b *Backend) OpenFile(_ context.Context, path string) (*model.Doc, error) {
	root := b.lib.Root()
	abs, fail := read.Resolve(root, path)
	if fail != nil {
		return &model.Doc{Root: root, Reason: fail.Reason, Detail: fail.Detail}, nil
	}
	cacheKey := root + "\x00" + path
	mtime, size, keyed := read.CacheProbe(abs)
	if keyed {
		if hit, cerr := b.storage.GetCachedExtract(cacheKey, mtime, size); cerr == nil && hit != nil {
			out := &model.Doc{
				Title: hit.Title, BlocksJSON: hit.BlocksJSON,
				PageCount: hit.PageCount, Kind: hit.Kind, Root: root,
			}
			// Dump-backed transfer: content goes to disk, the UI reads it
			// back by name. Inline JSON is cleared so big files never sit
			// in two memories at once.
			if name, derr := read.DumpFor("", root, path, hit.BlocksJSON); derr == nil {
				out.Dump = name
				out.BlocksJSON = ""
			}
			return out, nil
		}
	}
	opened, fail := read.OpenFile(root, "", path)
	if fail != nil {
		return &model.Doc{Root: root, Reason: fail.Reason, Detail: fail.Detail}, nil
	}
	opened.Root = root
	if keyed {
		// Best-effort store; extraction already succeeded, so a cache
		// failure must never fail the open.
		_ = b.storage.PutCachedExtract(model.CachedExtract{
			Path: cacheKey, Mtime: mtime, Size: size,
			Kind: opened.Kind, Title: opened.Title,
			BlocksJSON: opened.BlocksJSON, PageCount: opened.PageCount,
		})
	}
	// Dump-backed transfer (see cache-hit path): inline JSON cleared.
	if opened.Dump != "" {
		opened.BlocksJSON = ""
	}
	return &opened, nil
}

// maxRawFileBytes caps raw file serving for the PDF canvas leaf.
const maxRawFileBytes = 30 << 20 // 30 MiB

// ReadRawFile returns base64-encoded bytes of a workspace file for
// renderers needing originals (PDF canvas). Resolve-jailed; oversized
// files are refused, never streamed partially.
func (b *Backend) ReadRawFile(path string) (string, error) {
	abs, err := b.lib.Resolve(path)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return "", err
	}
	if info.IsDir() || info.Size() > maxRawFileBytes {
		return "", fmt.Errorf("file not servable: %s", path)
	}
	raw, err := os.ReadFile(abs)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(raw), nil
}

// ReadDumpFile returns one extract_dump file's content by NAME (see
// Doc.Dump). Names are fenced to the dump dir — no separators, no `..`.
func (b *Backend) ReadDumpFile(name string) (string, error) {
	return read.ReadDump("", name)
}

// ShowContainingFolder reveals a workspace path in the OS file manager.
// Resolution + containment go through the jail, so callers can never point
// the OS outside the open folder.
func (b *Backend) ShowContainingFolder(path string) (string, error) {
	return b.lib.ShowContainingFolder(path)
}

func (b *Backend) SetLibraryRoot(_ context.Context, root string) error {
	b.lib.SetRoot(root)
	if root != "" {
		b.adoptWorkspaceID(root)
		_ = b.storage.SetSetting("last_library_root", root)
	}
	return nil
}

func (b *Backend) LibraryRoot() string {
	return b.lib.Root()
}

// UsageMeters exposes AI session telemetry for the Settings dialog.
// Empty when nothing ran yet.
func (b *Backend) UsageMeters() string {
	if b.ai == nil {
		return ""
	}
	return b.ai.UsageMeters()
}

func (b *Backend) RunAI(ctx context.Context, req model.AIRequest, sink ai.EventSink) error {
	// If chat omitted summary content, load the primary summary from the DB.
	if req.SummaryContent == "" && req.WorkspaceID != "" {
		if sum, err := b.storage.GetPrimarySummary(req.WorkspaceID); err == nil && sum != nil {
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
	ws, err := b.storage.GetWorkspace(b.workspaceID)
	if err != nil {
		return err
	}
	if ws == nil {
		ws = &model.WorkspaceRecord{ID: b.workspaceID, RootPath: b.lib.Root()}
	}
	ws.PrimarySummaryID = summaryID
	ws.UpdatedAt = time.Now().UnixMilli()
	return b.storage.UpsertWorkspace(*ws)
}

// WriteSummaryDOCX writes canonical blocks as a .docx under the library
// root. Empty relPath auto-names from the first block's title.
func (b *Backend) WriteSummaryDOCX(relPath, blocksJSON string) (string, error) {
	root := b.lib.Root()
	if root == "" {
		return "", fmt.Errorf("no library root set")
	}
	if relPath == "" {
		title := "summary"
		if blocks, err := model.ParseBlocks(blocksJSON); err == nil && len(blocks) > 0 {
			if t := strings.TrimSpace(blocks[0].Text()); t != "" {
				title = t
			}
		}
		relPath = save.DefaultSummaryDOCXName(title)
	}
	return save.WriteSummaryDOCXPath(root, relPath, blocksJSON)
}

func (b *Backend) SaveThread(t model.ChatThreadRecord) error {
	// Only bump UpdatedAt so the original creation time survives re-saves.
	t.UpdatedAt = time.Now().UnixMilli()
	return b.storage.UpsertThread(t)
}

func (b *Backend) AppendChatMessage(msg model.ChatMessageRecord) error {
	// Messages are append-only; stamp CreatedAt millis.
	msg.CreatedAt = time.Now().UnixMilli()
	return b.storage.AppendMessage(msg)
}

func (b *Backend) ListThreads(workspaceID string) ([]model.ChatThreadRecord, error) {
	return b.storage.ListThreads(workspaceID)
}

func (b *Backend) ListMessages(threadID string) ([]model.ChatMessageRecord, error) {
	return b.storage.ListMessages(threadID)
}
