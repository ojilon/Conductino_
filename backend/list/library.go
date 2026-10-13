package list

import (
	"errors"
	"path/filepath"

	"Conductino/backend/model"
	"Conductino/backend/read"
)

// Library is the workspace root plus everything derived from it. One
// struct replaces the old Filesystem+Workspace pair: the React UI never
// builds paths, it sends tokens back and Library resolves them.
type Library struct {
	root string
}

// NewLibrary points at a workspace directory (absolute path from the
// folder dialog, or the "research_workspace" default).
func NewLibrary(root string) *Library { return &Library{root: root} }

// SetRoot repoints the workspace, e.g. after the folder dialog returns a
// new absolute path. Empty is ignored so a cancelled dialog is safe.
func (l *Library) SetRoot(path string) {
	if path == "" {
		return
	}
	l.root = path
}

// Root returns the workspace root as an absolute path.
func (l *Library) Root() string {
	abs, err := filepath.Abs(l.root)
	if err != nil {
		return l.root
	}
	return abs
}

// Tree lists the workspace (see List). Nil + nil = "no folder yet".
func (l *Library) Tree() (*model.Node, *model.Fail) { return List(l.root) }

// Resolve turns a token into an absolute disk path, jailed to the root.
// Reads must never serve files outside the workspace.
func (l *Library) Resolve(path string) (string, error) {
	abs, fail := read.Resolve(l.root, path)
	if fail != nil {
		return "", errors.New(fail.Error())
	}
	return abs, nil
}

// ListRoot is the tools-interface spelling of Tree (error, not Fail).
func (l *Library) ListRoot() (*model.Node, error) {
	n, fail := l.Tree()
	if fail != nil {
		return nil, errors.New(fail.Error())
	}
	return n, nil
}
