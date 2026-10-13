# Cycle 2: library list in `backend/list` (folder vs file)

Goal: rebuild the sidebar tree yourself. One new package, two files, zero
edits to old code. Borrowed from `backend/services/filesystem.go` walkDir,
trimmed to beginner shape.

## Files you create

1. `backend/list/list.go` — `Node` struct + `List()` + `walk()` + `Flat()`.
2. `backend/list/list_test.go` — tree shape, tokens, missing root, flat files.

## The model (say it out loud)

- A **folder** node has `Children`. A **file** node has `Ext` + `Path`, no children.
- `Path` is a **token**: path relative to the library root, slashes always
  `/` (even on Windows). The frontend displays `Label`, sends back `Path`.
- The root node itself has `Path ""` — it needs no token, it IS the root.
- JSON tags match the TS `FileTreeNode` shape, so Cycle 4 wiring is trivial.
- Failures reuse `*read.Fail` (import `Conductino/backend/read`) — new
  packages may import each other; only OLD packages are frozen.

## Go concepts you meet

- `struct` with a slice of itself (`Children []Node`): a tree in 6 fields.
- Recursion: `walk` calls `walk` for each subfolder. `os.ReadDir` returns
  entries sorted by name, so the tree order is deterministic.
- `continue` inside `for`: skip one bad entry (unreadable dir) without
  failing the whole tree — one rotten folder must not kill the library.
- `nil, nil` return = "no workspace yet, not an error" (matches old `ListRoot`).

## Run

```
go vet ./backend/list/
go test ./backend/list/ -v
```

## Done when

- 4 tests pass (tree-shape, tokens+ext, missing-root-is-nil, flat-files).
- You can explain why a symlink-to-folder shows as a FILE (answer: `e.IsDir()`
  is false for links, so we never recurse into them — no infinite cycles).
- Next: Cycle 3 (save/write-back).
