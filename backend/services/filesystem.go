package services

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/lumen/desktop/backend/models"
)

// FilesystemService is the boundary for all OS filesystem access.
// The React UI NEVER builds paths or spawns processes; it calls the
// bound methods on this service.
type FilesystemService interface {
	ListRoot() ([]models.FileTreeNode, error)
	ShowContainingFolder(path string) (string, error)
}

// Filesystem is the implementation. ListRoot currently walks the real
// workspace directory (a small, dependency-free implementation);
// ShowContainingFolder shells out per-platform.
type Filesystem struct {
	root string
}

func NewFilesystem(root string) *Filesystem { return &Filesystem{root: root} }

func (f *Filesystem) ListRoot() ([]models.FileTreeNode, error) {
	entries, err := os.ReadDir(f.root)
	if err != nil {
		// Workspace not present yet → empty tree (frontend shows its
		// empty state; it is NOT an app-level failure).
		return nil, nil
	}
	var out []models.FileTreeNode
	for i, e := range entries {
		node := models.FileTreeNode{
			ID:    "fs-" + itoa(i),
			Label: e.Name(),
			Kind:  "file",
		}
		if e.IsDir() {
			node.Kind = "folder"
		} else {
			node.Ext = filepath.Ext(e.Name())
		}
		out = append(out, node)
	}
	return out, nil
}

func (f *Filesystem) ShowContainingFolder(path string) (string, error) {
	full := filepath.Join(f.root, path)
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

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var b [8]byte
	pos := len(b)
	for i > 0 {
		pos--
		b[pos] = byte('0' + i%10)
		i /= 10
	}
	return string(b[pos:])
}
