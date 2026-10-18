// Package backend is pure, independent Go — it knows nothing about Wails.
package backend

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"Conductino/backend/apppaths"
	"Conductino/backend/docpipe"
	"Conductino/backend/extract"
	"Conductino/backend/mirror"
	"Conductino/backend/models"
	"Conductino/backend/services"
	"Conductino/backend/skills"
	"Conductino/backend/tools"
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
	if err := apppaths.EnsureDataRoot(); err != nil {
		fmt.Printf("appdata init: %v\n", err)
	}
	if err := b.storage.Init(ctx); err != nil {
		return err
	}
	// Restore last library root so the next session opens the same folder.
	// Workspace.SetLibraryRoot replays the stored root (no error return).
	if root, ok := b.storage.GetSetting("last_library_root"); ok && root != "" {
		b.work.SetLibraryRoot(root)
	}
	// Scratch-mirror hygiene (plan 10 §2): drop mirrors untouched for 7 days.
	// Best-effort — a failure here must never block startup.
	_ = mirror.New(mirror.DefaultDir()).Sweep(7 * 24 * time.Hour)
	return nil
}

// BuildContextPack assembles a budgeted context string server-side
// (plan 11 §4: budgeting moves behind an API; the UI keeps rendering).
func (b *Backend) BuildContextPack(blocksJSON, title, blockID, selText string, rangeStart, rangeEnd, windowBlocks int, includeOutline bool) (string, error) {
	return extract.BuildContextPack(blocksJSON, extract.PackOpts{
		Title: title, BlockID: blockID, SelText: selText,
		RangeStart: rangeStart, RangeEnd: rangeEnd,
		WindowBlocks: windowBlocks, IncludeOutline: includeOutline,
	})
}

// MatchSkills returns skill excerpts for an operation+intent as JSON
// (plan 11 §1). Empty array when nothing matches or AI is unconfigured.
func (b *Backend) MatchSkills(operation, intent string) (string, error) {
	type matcher interface{ MatchedSkills(string, string) string }
	if m, ok := b.ai.(matcher); ok && m != nil {
		return m.MatchedSkills(operation, intent), nil
	}
	return "[]", nil
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
	// Extract cache (issues 24+26): unchanged files never re-extract.
	// Key is root-anchored so the same rel token under another folder can't
	// collide. Cache errors are non-fatal — fall through to extraction.
	cacheKey := root + "\x00" + path
	var statMtime, statSize int64 = -1, -1
	if info, serr := os.Stat(abs); serr == nil && !info.IsDir() {
		statMtime, statSize = info.ModTime().UnixMilli(), info.Size()
		if hit, cerr := b.storage.GetCachedExtract(cacheKey, statMtime, statSize); cerr == nil && hit != nil {
			cached := models.OpenedDocument{
				Title: hit.Title, BlocksJSON: hit.BlocksJSON,
				PageCount: hit.PageCount, Kind: hit.Kind, Root: root,
			}
			// Cache hits still refresh the mirror pair (cheap, and it keeps
			// content.json/meta.json on disk for the AI + save bridge).
			b.attachMirror(root, path, abs, &cached)
			return &cached, nil
		}
	}
	opened, err := b.docs.OpenFile(abs, path)
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
	// MIRROR STEP (backend/docpipe): every successful open also produces the
	// content.json + meta.json pair — the format-free mirror the frontend
	// canvas renders and the AI reads. Best-effort: if mirroring fails the
	// blocks JSON above still opens the document exactly as before.
	b.attachMirror(root, path, abs, &opened)
	if statMtime >= 0 {
		// Best-effort store; extraction already succeeded, so a cache
		// failure must never fail the open.
		_ = b.storage.PutCachedExtract(services.CachedExtract{
			Path: cacheKey, Mtime: statMtime, Size: statSize,
			Kind: opened.Kind, Title: opened.Title,
			BlocksJSON: opened.BlocksJSON, PageCount: opened.PageCount,
		})
	}
	return &opened, nil
}

// maxRawFileBytes caps raw file serving for the PDF canvas leaf (issue 18).
const maxRawFileBytes = 30 << 20 // 30 MiB

// ReadRawFile returns base64-encoded bytes of a workspace file for renderers
// that need the original bytes (PDF canvas leaf). Resolve-jailed like
// OpenFile; oversized files are refused, never streamed partially.
func (b *Backend) ReadRawFile(path string) (string, error) {
	abs, err := b.fs.Resolve(path)
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

// UsageMeters exposes AI session telemetry (per-provider calls + tool
// counts) for the Settings dialog. Empty when nothing ran yet.
func (b *Backend) UsageMeters() string {
	if b.ai == nil {
		return ""
	}
	return b.ai.UsageMeters()
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

// WriteSummaryDOCX writes canonical blocks as a .docx under the library root.
// relPath is workspace-relative (e.g. "summaries/My-Paper.docx"); empty uses
// DefaultSummaryDOCXName from the first heading/title in blocksJSON.
// Returns the absolute path written.
func (b *Backend) WriteSummaryDOCX(relPath, blocksJSON string) (string, error) {
	root := b.work.LibraryRoot()
	if root == "" {
		return "", fmt.Errorf("no library root set")
	}
	if relPath == "" {
		title := "summary"
		if blocks, err := services.ParseBlocksJSON(blocksJSON); err == nil && len(blocks) > 0 {
			t := ""
			for _, s := range blocks[0].Segments {
				t += s.Text
			}
			if t != "" {
				title = t
			}
		}
		relPath = services.DefaultSummaryDOCXName(title)
	}
	return b.docs.WriteSummaryDOCX(root, relPath, blocksJSON)
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

// ===========================================================================
// The MIRROR PIPELINE bridge methods (backend/docpipe)
//
// OpenFile (above) already attaches the mirror pair to every opened
// document. The methods below cover the rest of the lifecycle:
//
//   SaveDocumentMirror — the SAVE BRIDGE: edited canvas JSON → verified
//                        write-back into the source .docx/.txt/.md
//   CreateFile / CreateFolder / RenameFile — library file operations
//   SetSummaryTarget / GetSummaryTarget   — the workspace summary document
//   ListSkillsTree / ReadSkillFile / WriteSkillFile / CreateSkillEntry
//                                          — the AI skills folder as files
//   RunShell                               — the basic sandboxed AI shell
// ===========================================================================

// attachMirror builds the mirror pair (content.json + meta.json) for one
// opened document, saves it under the app-data cache, and attaches the
// compact JSON strings to the wire value. Best-effort by design: any
// failure leaves BlocksJSON untouched and the document opens as before.
func (b *Backend) attachMirror(root, relPath, absPath string, opened *models.OpenedDocument) {
	if opened == nil || opened.BlocksJSON == "" || opened.Reason != "" {
		return
	}
	kind := opened.Kind
	if kind == "" {
		// Fall back to the extension so odd arms still mirror.
		switch strings.ToLower(filepath.Ext(relPath)) {
		case ".pdf":
			kind = "pdf"
		case ".docx":
			kind = "docx"
		default:
			kind = "text"
		}
	}
	content, meta, err := docpipe.Build(kind, opened.Title, relPath, absPath, opened.PageCount, opened.BlocksJSON)
	if err != nil {
		return
	}
	dir, _ := docpipe.SaveFor(root, relPath, content, meta)
	opened.MirrorDir = dir
	if cj, jerr := json.Marshal(content); jerr == nil {
		opened.MirrorJSON = string(cj)
	}
	if mj, jerr := json.Marshal(meta); jerr == nil {
		opened.MetaJSON = string(mj)
	}
}

// SaveDocumentMirror is the SAVE BRIDGE: the frontend sends the edited
// mirror content ({version,kind,title,blocks}) and docpipe.WriteBack
// verifies which blocks changed (per-block hash + source stamp from
// meta.json) before writing the source file back. Returns the WriteReport
// as JSON: {written, changed[], added[], removed[], unchangedCount, …}.
func (b *Backend) SaveDocumentMirror(path, contentJSON string) (string, error) {
	abs, err := b.fs.Resolve(path)
	if err != nil {
		return "", err
	}
	report, err := docpipe.WriteBack(b.fs.Root(), path, abs, contentJSON)
	if err != nil {
		return "", err
	}
	raw, err := json.Marshal(report)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

// CreateFile makes a new empty file in the workspace (.docx gets a valid
// empty Word package; .txt/.md an empty text file). Returns the relative
// path token of the created file.
func (b *Backend) CreateFile(relPath string) (string, error) {
	return b.fs.CreateFile(relPath)
}

// CreateFolder makes a new folder in the workspace.
func (b *Backend) CreateFolder(relPath string) (string, error) {
	return b.fs.CreateFolder(relPath)
}

// RenameFile renames a file/folder in place (new base name only).
func (b *Backend) RenameFile(relPath, newName string) (string, error) {
	return b.fs.Rename(relPath, newName)
}

// SetSummaryTarget marks a workspace .docx as THE document all summaries
// are siphoned into (library right-click → "Make this the target summary
// document"). The marker lives in app-data (docpipe/target.go) so the Go
// tools and the AI can find it without any extra wiring.
func (b *Backend) SetSummaryTarget(relPath string) error {
	abs, err := b.fs.Resolve(relPath)
	if err != nil {
		return err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return fmt.Errorf("file not found in workspace: %s", relPath)
	}
	if info.IsDir() {
		return fmt.Errorf("the summary target must be a file, not a folder")
	}
	if !strings.EqualFold(filepath.Ext(abs), ".docx") {
		return fmt.Errorf("the summary target must be a .docx file (so the AI can write into it)")
	}
	base := filepath.Base(abs)
	return docpipe.SaveTarget(docpipe.SummaryTarget{
		Root:    b.fs.Root(),
		RelPath: filepath.ToSlash(relPath),
		Title:   strings.TrimSuffix(base, filepath.Ext(base)),
	})
}

// GetSummaryTarget returns the marked summary target as JSON
// ({"root":…,"relPath":…,"title":…}) or "" when none is marked for the
// currently open workspace folder.
func (b *Backend) GetSummaryTarget() string {
	target, ok := docpipe.LoadTarget()
	if !ok || target.Root != b.fs.Root() {
		return ""
	}
	raw, err := json.Marshal(target)
	if err != nil {
		return ""
	}
	return string(raw)
}

// ListSkillsTree returns the AI skills folder (backend/skills in dev,
// app-data skills installed) as the same tree shape the library uses, so
// the frontend renders it with familiar components.
func (b *Backend) ListSkillsTree() (*models.FileTreeNode, error) {
	return skills.ListTree()
}

// ReadSkillFile returns one instruction file's text (path relative to the
// skills folder, e.g. "my_instructions/reading-rules.md").
func (b *Backend) ReadSkillFile(rel string) (string, error) {
	return skills.ReadFile(rel)
}

// WriteSkillFile saves one instruction file (used by the Skills panel).
func (b *Backend) WriteSkillFile(rel, content string) error {
	return skills.WriteFile(rel, content)
}

// CreateSkillEntry creates a new file (isDir=false) or folder (isDir=true)
// inside the skills folder — e.g. ("my_instructions", true) then
// ("my_instructions/rules.md", false).
func (b *Backend) CreateSkillEntry(rel string, isDir bool) error {
	return skills.CreateEntry(rel, isDir)
}

// RunShell executes one basic command in the sandboxed AI shell
// (backend/tools/shell.go): cwd = the open research folder, 10s timeout,
// capped output, deny-list. Exposed to the UI for transparency/debugging
// and used by the AI through the "shell" tool.
func (b *Backend) RunShell(command string) (string, error) {
	return tools.RunShell(b.fs.Root(), command)
}

// UpdateMirrorContent pushes the LIVE editor state into content.json (no
// source-file write). Called debounced while the user types so the mirror —
// and therefore the AI's read_mirror view — always matches what the user
// sees. meta.json (the source guide) is left alone until a real save.
func (b *Backend) UpdateMirrorContent(path, contentJSON string) error {
	return docpipe.UpdateContent(b.fs.Root(), path, contentJSON)
}
