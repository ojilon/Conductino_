package services

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	// NOTE (import fix): was `github.com/lumen/desktop/backend/models`.
	// The module is `Conductino` (go.mod), so the correct local path is
	// `Conductino/backend/models`. `models` supplies FileTreeNode, the
	// JSON shape returned to the frontend over the Wails boundary.
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

// ShowContainingFolder reveals a path in the OS file manager and returns
// the revealed directory. path is a FileTreeNode.Path token (see models):
// an absolute path is used as-is, a relative one is resolved against the
// workspace root and cleaned, so `sub/../x` cannot escape into a surprise
// location. Per-platform mechanics are unchanged: Windows selects the file,
// macOS reveals it, Linux opens the containing directory.
func (f *Filesystem) ShowContainingFolder(path string) (string, error) {
	full := path
	if !filepath.IsAbs(path) {
		abs, err := filepath.Abs(f.root)
		if err != nil {
			return "", err
		}
		full = filepath.Join(abs, path)
	}
	full = filepath.Clean(full)
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
