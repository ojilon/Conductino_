package skills

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"Conductino/backend/apppaths"
	"Conductino/backend/models"
)

// The skills folder as a tiny FILE BROWSER (UI + AI share this).
//
// skills/ holds the AI's instruction files: bundled workflows (like
// summarize-source.md), the internal user guide (app-guide.md), and any
// custom folders the user creates from the frontend, e.g.
// skills/my_instructions/reading-rules.md. Everything here is plain
// Markdown — the same functions serve:
//
//   - the Skills panel in the UI (list / read / write / create), and
//   - the AI tool read_skill (read one instruction file by path).
//
// All paths are RELATIVE to the skills root and containment-checked, so
// neither the UI nor the AI can reach outside the folder.

// BrowseDir is the skills folder the UI browses and edits. In a dev checkout
// that is <repo>/backend/skills (edits are live for the AI on next run);
// installed builds use the app-data skills dir (apppaths.SkillsDir).
func BrowseDir() string {
	if root := findRepoRoot(); root != "" {
		dev := filepath.Join(root, "backend", "skills")
		if st, err := os.Stat(dev); err == nil && st.IsDir() {
			return dev
		}
	}
	return apppaths.SkillsDir()
}

// findRepoRoot walks up from the working directory looking for go.mod or
// wails.json (same trick mirror.DefaultDir uses). "" when not in a repo.
func findRepoRoot() string {
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}
	dir := cwd
	for {
		for _, marker := range []string{"go.mod", "wails.json"} {
			if _, err := os.Stat(filepath.Join(dir, marker)); err == nil {
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

// safeJoin resolves a relative token inside the skills dir, rejecting
// absolute paths and any ".." escape. Every read/write goes through it.
func safeJoin(baseDir, rel string) (string, error) {
	rel = strings.TrimSpace(filepath.ToSlash(rel))
	if rel == "" || rel == "." || rel == "skills" || rel == "skills/" {
		return baseDir, nil // the folder itself
	}
	if filepath.IsAbs(rel) || strings.HasPrefix(rel, "/") {
		return "", fmt.Errorf("skills path must be relative: %q", rel)
	}
	// Drop a leading "skills/" so UI tokens like "skills/foo.md" work too.
	rel = strings.TrimPrefix(rel, "skills/")
	full := filepath.Join(baseDir, filepath.FromSlash(rel))
	back, err := filepath.Rel(baseDir, full)
	if err != nil || back == ".." || strings.HasPrefix(back, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("skills path escapes the skills folder: %q", rel)
	}
	return full, nil
}

// skillFileExts are the text formats the skills browser shows and edits.
var skillFileExts = map[string]bool{".md": true, ".markdown": true, ".txt": true}

// ListTree returns the skills folder as the same FileTreeNode shape the
// library uses, so the frontend can render it with one tree component.
// The root node has Label "skills" and an empty Path token.
func ListTree() (*models.FileTreeNode, error) {
	base := BrowseDir()
	if err := os.MkdirAll(base, 0o755); err != nil {
		return nil, err
	}
	root, err := walkSkills(base, base)
	if err != nil {
		return nil, err
	}
	root.Label = "skills"
	root.Path = ""
	return root, nil
}

// walkSkills builds one folder node (recursively). Unreadable entries are
// skipped, hidden entries (dot-files) are ignored.
func walkSkills(base, absDir string) (*models.FileTreeNode, error) {
	entries, err := os.ReadDir(absDir)
	if err != nil {
		return nil, err
	}
	rel, err := filepath.Rel(base, absDir)
	if err != nil {
		return nil, err
	}
	token := filepath.ToSlash(rel)
	if token == "." {
		token = ""
	}
	node := &models.FileTreeNode{ID: "skills:" + token, Kind: "folder", Label: filepath.Base(absDir), Path: token}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".") {
			continue
		}
		full := filepath.Join(absDir, e.Name())
		// Child token = this folder's token + "/" + name (root token is "").
		childRel := e.Name()
		if token != "" {
			childRel = token + "/" + e.Name()
		}
		if e.IsDir() {
			child, err := walkSkills(base, full)
			if err != nil {
				continue
			}
			node.Children = append(node.Children, *child)
			continue
		}
		if !skillFileExts[strings.ToLower(filepath.Ext(e.Name()))] {
			continue // only instruction text files are browsable
		}
		node.Children = append(node.Children, models.FileTreeNode{
			ID:    "skills:" + childRel,
			Kind:  "file",
			Label: e.Name(),
			Ext:   strings.TrimPrefix(filepath.Ext(e.Name()), "."),
			Path:  childRel,
		})
	}
	return node, nil
}

// ReadFile returns one skill/instruction file's text.
func ReadFile(rel string) (string, error) {
	full, err := safeJoin(BrowseDir(), rel)
	if err != nil {
		return "", err
	}
	raw, err := os.ReadFile(full)
	if err != nil {
		return "", fmt.Errorf("cannot read skills file %q: %w", rel, err)
	}
	if len(raw) > 1<<20 {
		return "", fmt.Errorf("skills file %q is too large (1 MB cap)", rel)
	}
	return string(raw), nil
}

// WriteFile saves one skill/instruction file (creating parent folders), so
// the user can author custom instructions from the UI and the AI picks them
// up through loadBundledSkills / read_skill.
func WriteFile(rel, content string) error {
	full, err := safeJoin(BrowseDir(), rel)
	if err != nil {
		return err
	}
	if !skillFileExts[strings.ToLower(filepath.Ext(full))] {
		return fmt.Errorf("skills files must be .md/.markdown/.txt: %q", rel)
	}
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return err
	}
	return os.WriteFile(full, []byte(content), 0o644)
}

// CreateEntry makes a new empty file or folder inside skills/.
func CreateEntry(rel string, isDir bool) error {
	full, err := safeJoin(BrowseDir(), rel)
	if err != nil {
		return err
	}
	if full == BrowseDir() {
		return fmt.Errorf("give the new %s a name", map[bool]string{true: "folder", false: "file"}[isDir])
	}
	if isDir {
		return os.MkdirAll(full, 0o755)
	}
	if !skillFileExts[strings.ToLower(filepath.Ext(full))] {
		return fmt.Errorf("skills files must be .md/.markdown/.txt: %q", rel)
	}
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return err
	}
	// Never clobber an existing instruction file.
	if _, err := os.Stat(full); err == nil {
		return fmt.Errorf("%q already exists", rel)
	}
	return os.WriteFile(full, []byte("# "+filepath.Base(full)+"\n\nWrite instructions for the AI here.\n"), 0o644)
}
