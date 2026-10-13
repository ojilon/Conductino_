package list

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// ShowContainingFolder reveals a workspace path in the OS file manager and
// returns the revealed directory. Resolution and containment go through the
// jail; per-platform mechanics: Windows selects the file, macOS reveals it,
// Linux opens the containing directory.
func (l *Library) ShowContainingFolder(path string) (string, error) {
	full, err := l.Resolve(path)
	if err != nil {
		return "", err
	}
	dir := full
	if info, err := os.Stat(full); err != nil || !info.IsDir() {
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
