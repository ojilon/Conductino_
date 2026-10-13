# Cycle 1: open-file loop in `backend/read` (txt/md only)

Goal: type the click → contents loop yourself in ONE new package. No edits to
old code. Five new files, then `go test ./backend/read/` green.

## Files you create (in order)

1. `backend/read/read.go` — `Doc` + `Fail` structs, reason constants.
2. `backend/read/resolve.go` — token → absolute path jail (from `services/filesystem.go`).
3. `backend/read/text.go` — txt/md parse + caps + block IDs (from `extract/text.go`, `extract/ids.go`).
4. `backend/read/open.go` — `OpenFile(libraryRoot, dataRoot, token)` orchestrates all + dumps.
5. `backend/read/read_test.go` — jail, headings, dump-exists, unsupported, too-large.

## Go concepts you meet (5 lines each, from your own new code)

- `struct`: a named box holding fields (`Doc{Title, Kind, Blocks, Pages, Root}`).
- `*Fail` (pointer): `nil` = success; non-nil = classified failure. Check it first.
- `switch ext`: picks the parser by file extension; unknown → `unsupported`, never a crash.
- `defer f.Close()`: "close this file when I leave this function, no matter how".
- `_, _ = ...`: "run it, throw away both results" — dumps must never fail an open.

## Run

```
go vet ./backend/read/
go test ./backend/read/ -v
```

## Done when

- 5 tests pass (jail-reject, resolve-ok, headings, dump-equals-blocks, unsupported, too-large — 6 actually, jail has 2).
- You can point at each function and say which old file it was borrowed from.
- Next: Cycle 2 spec (library list) — mentor writes it after this is green.

## Deliberately NOT in this cycle

pdf/docx arms, SQLite cache, Wails wiring, touching old packages. Those are
Cycles 2–4. Small loop first, green test, then expand.
