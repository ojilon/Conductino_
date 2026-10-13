// Package list owns the library: the workspace root, the file tree the
// sidebar renders, and folder reveals. Old filesystem/workspace split
// merged here: one struct (Library), one tree walk, tokens all the way
// down. Old code untouched elsewhere until the wire-up cycle.
package list

import (
	"os"
	"path/filepath"
	"strings"

	"Conductino/backend/model"
)

// List returns the whole workspace as ONE nested root node.
// nil,nil means "no workspace yet" — the UI shows "Choose folder", not
// an error.
func List(libraryRoot string) (*model.Node, *model.Fail) {
	abs, err := filepath.Abs(libraryRoot)
	if err != nil {
		return nil, &model.Fail{Reason: model.ReasonParseError, Detail: err.Error()}
	}
	n, err := walk(abs, abs)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, &model.Fail{Reason: model.ReasonParseError, Detail: err.Error()}
	}
	return n, nil
}

// walk builds the node for dir. root stays fixed so every Path is
// root-relative. os.ReadDir returns entries sorted by filename, so tree
// order is deterministic. Unreadable subfolders are SKIPPED (continue) —
// one bad folder must not fail the whole tree.
func walk(root, dir string) (*model.Node, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	label := filepath.Base(dir)
	if label == "" || label == string(filepath.Separator) {
		label = dir // e.g. a drive root like `D:\` has no base name
	}
	rel, err := filepath.Rel(root, dir)
	if err != nil {
		return nil, err
	}
	relSlash := filepath.ToSlash(rel)
	id, nodePath := relSlash, relSlash
	if rel == "." {
		id, nodePath = ".", "" // the root itself needs no path token
	}
	n := &model.Node{ID: id, Kind: "folder", Label: label, Path: nodePath}
	for _, e := range entries {
		full := filepath.Join(dir, e.Name())
		childRel, err := filepath.Rel(root, full)
		if err != nil {
			continue // unresolvable path — skip it, don't fail the tree
		}
		childSlash := filepath.ToSlash(childRel)
		if e.IsDir() {
			child, err := walk(root, full)
			if err != nil {
				continue
			}
			n.Children = append(n.Children, *child)
		} else {
			// Links report IsDir()==false, so a symlink-to-folder lands
			// here as a file: links are never followed, cycles impossible.
			// Ext keeps the dot stripped ("pdf", not ".pdf").
			n.Children = append(n.Children, model.Node{
				ID: childSlash, Kind: "file", Label: e.Name(),
				Ext:  strings.TrimPrefix(filepath.Ext(e.Name()), "."),
				Path: childSlash,
			})
		}
	}
	return n, nil
}

// Flat collects every FILE node depth-first. The "Open document" menu and
// tests use this, not the nested tree.
func Flat(n *model.Node) []model.Node {
	var out []model.Node
	if n == nil {
		return out
	}
	for i := range n.Children {
		c := n.Children[i]
		if c.Kind == "file" {
			out = append(out, c)
		} else {
			out = append(out, Flat(&n.Children[i])...)
		}
	}
	return out
}
