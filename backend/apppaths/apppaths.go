// Package apppaths is the single path resolver for all Conductino data.
//
// Resolution order (docs/release-prep/02-local-storage-layout.md and
// 04-temp-debug-mirror.md):
//
//  1. CONDUCTINO_DATA (absolute app-data root) when set.
//  2. Dev: .dev-data at the repo root when present (cwd or walked up).
//  3. Portable: directory next to the executable (./data) when present.
//  4. Installed: platform app-data under Conductino
//     (os.UserConfigDir, e.g. %AppData% on Windows).
//
// Every subsystem (SQLite, skills, mirrors, cache, logs) must resolve
// through here — never a second hard-coded C:\Users\… or backend/.work.
// Tests set CONDUCTINO_DATA to a temp dir.
package apppaths

import (
	"os"
	"path/filepath"
	"strings"
)

// repoMarkers identifies a repo root while walking up from cwd.
var repoMarkers = []string{"go.mod", "wails.json"}

// DataRoot returns the writable app-data root. It never creates
// directories; call EnsureDataRoot when the app needs them on disk.
func DataRoot() string {
	// 1. Explicit override wins (tests + portable power users).
	if p := strings.TrimSpace(os.Getenv("CONDUCTINO_DATA")); p != "" {
		if abs, err := filepath.Abs(p); err == nil {
			return abs
		}
		return p
	}
	// 2. Repo-local debug root: .dev-data beside go.mod/wails.json.
	if root := findRepoRoot(); root != "" {
		dev := filepath.Join(root, ".dev-data")
		if st, err := os.Stat(dev); err == nil && st.IsDir() {
			return dev
		}
	}
	// 3. Portable: ./data next to the executable when it exists.
	if exe, err := os.Executable(); err == nil {
		port := filepath.Join(filepath.Dir(exe), "data")
		if st, err := os.Stat(port); err == nil && st.IsDir() {
			return port
		}
	}
	// 4. Installed platform app-data.
	if dir, err := os.UserConfigDir(); err == nil && strings.TrimSpace(dir) != "" {
		return filepath.Join(dir, "Conductino")
	}
	// Last resort (UserConfigDir failed): beside the executable.
	if exe, err := os.Executable(); err == nil {
		return filepath.Join(filepath.Dir(exe), "Conductino-data")
	}
	return filepath.Join(os.TempDir(), "Conductino")
}

// findRepoRoot walks up from cwd looking for go.mod/wails.json.
func findRepoRoot() string {
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}
	dir := cwd
	for {
		for _, m := range repoMarkers {
			if _, err := os.Stat(filepath.Join(dir, m)); err == nil {
				return dir
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// IsDev reports whether DataRoot resolved to the repo-local .dev-data.
func IsDev() bool {
	root := findRepoRoot()
	if root == "" {
		return false
	}
	return DataRoot() == filepath.Join(root, ".dev-data")
}

// DBPath returns the SQLite file path. CONDUCTINO_DB overrides just the
// file (docs 02: optional later — implemented from day one so drive choice
// never strands user data).
func DBPath() string {
	if p := strings.TrimSpace(os.Getenv("CONDUCTINO_DB")); p != "" {
		if abs, err := filepath.Abs(p); err == nil {
			return abs
		}
		return p
	}
	return filepath.Join(DataRoot(), "conductino.db")
}

// WorkDir is the production home for what is today backend/.work/
// (mirrors, op logs).
func WorkDir() string { return filepath.Join(DataRoot(), "work") }

// SummariesDir holds summary mirror .md/.json pairs.
func SummariesDir() string { return filepath.Join(WorkDir(), "summaries") }

// SkillsDir holds global user skills (SKILL.md files).
func SkillsDir() string { return filepath.Join(DataRoot(), "skills") }

// DevSkillsDir is the repo-local staging area for experimental skills
// before promoting to backend/skills/ (docs 04).
func DevSkillsDir() string {
	if root := findRepoRoot(); root != "" {
		return filepath.Join(root, ".dev-data", "skills")
	}
	return filepath.Join(DataRoot(), "skills")
}

// CacheDir holds extracted-content cache spillover (SQLite is primary).
func CacheDir() string { return filepath.Join(DataRoot(), "cache", "extract") }

// ConfigPath is the app-data config.json location.
func ConfigPath() string { return filepath.Join(DataRoot(), "config.json") }

// EnsureDataRoot creates the standard tree (root, work/summaries,
// skills, cache). Best-effort callers ignore the error only in tests;
// production Init must surface it in logs.
func EnsureDataRoot() error {
	for _, d := range []string{DataRoot(), WorkDir(), SummariesDir(), SkillsDir(), CacheDir()} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return err
		}
	}
	return nil
}
