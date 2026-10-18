package services

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	// NOTE (import fix): was `github.com/lumen/desktop/backend/models`.
	// The module is `Conductino` (go.mod), so the correct local path is
	// `Conductino/backend/models`. `models` supplies FileTreeNode, the
	// JSON shape returned to the frontend over the Wails boundary.
	"Conductino/backend/extract"
	"Conductino/backend/models"
)

// FilesystemService is the boundary for all OS filesystem access.
// The React UI NEVER builds paths or spawns processes; it calls the
// bound methods on this service.
type FilesystemService interface {
	// ListRoot returns the whole workspace as ONE nested root node (with
	// Children), matching the TS FileTreeNode shape. Nil + nil means
	// "no workspace yet" — the UI shows its empty state, not an error.
	ListRoot() (*models.FileTreeNode, error)
	// SetRoot points the service at a new workspace directory (absolute
	// path from the folder dialog). Stored as given; resolved to absolute
	// at read time, so a relative default still works.
	SetRoot(path string)
	ShowContainingFolder(path string) (string, error)
	// Root returns the absolute workspace root. Used to tag opened
	// documents with their opening folder (tasks.md 1.1 option b) so the
	// UI can detect stale tabs after a folder switch.
	Root() string
}

// Filesystem is the implementation. ListRoot recursively walks the
// workspace directory via walkDir (nested tree, root-relative Path tokens);
// ShowContainingFolder shells out per-platform.
type Filesystem struct {
	root string
}

func NewFilesystem(root string) *Filesystem { return &Filesystem{root: root} }

// SetRoot repoints the workspace, e.g. after the folder dialog in the
// Wails shell returns a new absolute path. An empty path is ignored so a
// cancelled dialog can call through safely.
func (f *Filesystem) SetRoot(path string) {
	if path == "" {
		return
	}
	f.root = path
}

// Root returns the workspace root as an absolute path. Falls back to the
// stored value when Abs fails so callers always get a usable tag.
func (f *Filesystem) Root() string {
	abs, err := filepath.Abs(f.root)
	if err != nil {
		return f.root
	}
	return abs
}

func (f *Filesystem) ListRoot() (*models.FileTreeNode, error) {
	abs, err := filepath.Abs(f.root)
	if err != nil {
		return nil, err
	}
	root, err := walkDir(abs, abs)
	if err != nil {
		if os.IsNotExist(err) {
			// Workspace not present yet → empty tree (frontend shows its
			// empty state; it is NOT an app-level failure).
			return nil, nil
		}
		return nil, err
	}
	return root, nil
}

// Resolve turns a FileTreeNode.Path token into an absolute disk path and
// verifies it stays inside the workspace root. Tokens are slash-normalized
// (FromSlash) and root-relative by contract, but absolute paths are accepted
// too — as long as they resolve inside the root. Anything escaping the root
// (crafted `..`, absolute outsiders, unresolvable relatives) is rejected:
// reads (OpenFile) must never serve files outside the workspace, and reveals
// must never point the OS at one.
func (f *Filesystem) Resolve(path string) (string, error) {
	abs, err := filepath.Abs(f.root)
	if err != nil {
		return "", err
	}
	clean := path
	if !filepath.IsAbs(path) {
		clean = filepath.Join(abs, filepath.FromSlash(path))
	}
	clean = filepath.Clean(clean)
	rel, err := filepath.Rel(abs, clean)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path escapes workspace: %q", path)
	}
	return clean, nil
}

// ShowContainingFolder reveals a path in the OS file manager and returns
// the revealed directory. Resolution and containment go through Resolve;
// per-platform mechanics are unchanged: Windows selects the file, macOS
// reveals it, Linux opens the containing directory.
func (f *Filesystem) ShowContainingFolder(path string) (string, error) {
	full, err := f.Resolve(path)
	if err != nil {
		return "", err
	}
	dir := full
	if !looksLikeDir(full) {
		dir = filepath.Dir(full)
	}
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("explorer", "/select,", full)
	case "darwin":
		cmd = exec.Command("open", "-R", full)
	default:
		cmd = exec.Command("xdg-open", dir)
	}
	if err := cmd.Start(); err != nil {
		return "", err
	}
	return dir, nil
}

func looksLikeDir(p string) bool {
	info, err := os.Stat(p)
	return err == nil && info.IsDir()
}

// walkDir builds a nested FileTreeNode for the directory at absPath.
// root is the absolute workspace root; every node's Path is stored relative
// to it (see FileTreeNode.Path), so no absolute path ever crosses to the
// frontend. IDs derive from that same relative path, keeping them stable
// across refreshes for as long as the folder doesn't move.
//
// Value semantics: models.FileTreeNode.Children is []FileTreeNode (values,
// not pointers), so children are appended dereferenced. os.ReadDir returns
// entries sorted by filename, so the tree order is deterministic.
// Unreadable subdirectories are skipped — one bad folder must not fail the
// whole tree. Symlinks are listed by link name (a symlink to a directory
// shows as a file), so a link cycle can never cause infinite recursion.
//
// NOTE on Ext: the dot is stripped ("pdf", not ".pdf"), matching the TS
// mock data. The old flat ListRoot kept the dot; it is now replaced, so
// the stripped form is the contract going forward.
func walkDir(root, absPath string) (*models.FileTreeNode, error) {
	entries, err := os.ReadDir(absPath)
	if err != nil {
		return nil, err
	}
	label := filepath.Base(absPath)
	if label == "" || label == string(filepath.Separator) {
		label = absPath // e.g. a drive root like `D:\` has no base name
	}
	rel, err := filepath.Rel(root, absPath)
	if err != nil {
		return nil, err
	}
	relSlash := filepath.ToSlash(rel)
	id, nodePath := relSlash, relSlash
	if rel == "." {
		id, nodePath = ".", "" // the root itself needs no path token (omitempty drops it)
	}
	node := &models.FileTreeNode{ID: id, Kind: "folder", Label: label, Path: nodePath}
	for _, e := range entries {
		full := filepath.Join(absPath, e.Name())
		childRel, err := filepath.Rel(root, full)
		if err != nil {
			continue // unresolvable path — skip it, don't fail the tree
		}
		childRelSlash := filepath.ToSlash(childRel)
		if e.IsDir() {
			child, err := walkDir(root, full)
			if err != nil {
				continue
			} // skip unreadable dirs, don't crash whole tree
			node.Children = append(node.Children, *child)
		} else {
			node.Children = append(node.Children, models.FileTreeNode{
				ID: childRelSlash, Kind: "file", Label: e.Name(),
				Ext:  strings.TrimPrefix(filepath.Ext(e.Name()), "."),
				Path: childRelSlash, // <-- relative token; extraction resolves it against root later
			})
		}
	}
	return node, nil
}

// ---------------------------------------------------------------------------
// File operations for the library UI (create / rename)
//
// Everything goes through Resolve first: the same containment check that
// reads use, so the UI can never create or rename files outside the open
// research folder. Paths returned are workspace-relative slash tokens (the
// same shape FileTreeNode.Path uses) so the frontend can refresh + locate.
// ---------------------------------------------------------------------------

// CreateFile makes a new empty file inside the workspace.
//
//	.docx → a VALID empty Word package (via extract.WriteDOCX), so the file
//	        opens immediately in the reader and in Word,
//	.txt/.md/.markdown → an empty text file,
//	anything else → an honest error (we refuse to fake unknown formats).
//
// relPath is the workspace-relative token, e.g. "notes/Summary.docx".
func (f *Filesystem) CreateFile(relPath string) (string, error) {
	rel := normalizeRelToken(relPath)
	if rel == "" {
		return "", fmt.Errorf("give the new file a name")
	}
	abs, err := f.Resolve(rel)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(abs); err == nil {
		return "", fmt.Errorf("%s already exists", rel)
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return "", err
	}
	switch strings.ToLower(filepath.Ext(abs)) {
	case ".docx":
		// One blank paragraph — Word rejects a truly empty document.xml body.
		blank := []extract.DocxBlock{{
			ID: "empty", Type: "paragraph",
			Segments: []extract.DocxSegment{{Text: ""}},
		}}
		if err := extract.WriteDOCX(abs, blank); err != nil {
			return "", err
		}
	case ".txt", ".md", ".markdown":
		if err := os.WriteFile(abs, []byte(""), 0o644); err != nil {
			return "", err
		}
	default:
		return "", fmt.Errorf("cannot create %q — supported new files: .docx, .txt, .md", filepath.Ext(abs))
	}
	return rel, nil
}

// CreateFolder makes a new folder inside the workspace (e.g. "skills-notes").
func (f *Filesystem) CreateFolder(relPath string) (string, error) {
	rel := normalizeRelToken(relPath)
	if rel == "" {
		return "", fmt.Errorf("give the new folder a name")
	}
	abs, err := f.Resolve(rel)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(abs); err == nil {
		return "", fmt.Errorf("%s already exists", rel)
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		return "", err
	}
	return rel, nil
}

// Rename renames a file or folder IN PLACE (same parent folder, new base
// name). newName is just the name — "Summary v2.docx", never a path — so a
// rename can never move things out of their folder.
func (f *Filesystem) Rename(relPath, newName string) (string, error) {
	rel := normalizeRelToken(relPath)
	newName = strings.TrimSpace(newName)
	if rel == "" || newName == "" {
		return "", fmt.Errorf("rename needs the old path and a new name")
	}
	if strings.ContainsAny(newName, `/\`) {
		return "", fmt.Errorf("new name must not contain path separators")
	}
	oldAbs, err := f.Resolve(rel)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(oldAbs); err != nil {
		return "", fmt.Errorf("%s does not exist", rel)
	}
	newAbs := filepath.Join(filepath.Dir(oldAbs), newName)
	// Containment check on the destination too (defense in depth).
	newRel, err := filepath.Rel(f.Root(), newAbs)
	if err != nil || strings.HasPrefix(newRel, "..") {
		return "", fmt.Errorf("rename would leave the workspace")
	}
	if _, err := os.Stat(newAbs); err == nil {
		return "", fmt.Errorf("%s already exists", newName)
	}
	if err := os.Rename(oldAbs, newAbs); err != nil {
		return "", err
	}
	return filepath.ToSlash(newRel), nil
}

// normalizeRelToken cleans a frontend path token: slash-normalized, no
// leading "./" or "/", no trailing spaces. Resolve still does the real
// containment check — this is just tidying.
func normalizeRelToken(relPath string) string {
	rel := strings.TrimSpace(filepath.ToSlash(relPath))
	rel = strings.TrimPrefix(rel, "./")
	rel = strings.TrimPrefix(rel, "/")
	return strings.TrimSuffix(rel, "/")
}
