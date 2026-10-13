# Backend redo — plan (owner-led, logic by logic)

`docs/journey/` is frozen history — do not edit it. This folder is the live
build log for rewriting `backend/` one complete loop at a time, with YOU
typing the code and the mentor supplying paste-ready snippets.

## Why redo instead of refactor

The AI-built backend works but has 3 homes for 1 idea (e.g. text parsing in
`backend/extract/text.go`, `backend/internal/heavy/parse_text.go`,
`backend/services/documents_heavy.go`; alias file
`backend/services/extract_alias.go`). A beginner cannot tell which copy is
real. So: **new small packages beside the old ones, old code untouched**
until a new loop provably replaces it.

## Rules

1. New code lives in NEW subfolders of `backend/` (`read`, then `list`, `save`...).
2. Old packages are never edited (except one append-only Wails wire-up at the end).
3. Tiny structs, plain functions, no interfaces until a second implementation exists.
4. Every borrowed algorithm cites its source file in a comment, then gets trimmed.
5. Every cycle ends with `go vet` + `go test` green AND visible proof via
   `scripts\dev.bat` (dumps in `.dev-data\temporary\extract_dump\`).
6. `docs/journey/` is never touched. This folder (`docs/backend-redo/`) is rewritten freely.

## Package map (target)

| New package | Owns | Borrowed from |
|---|---|---|
| `backend/read` (Cycle 1) | click → jail-check → parse → dump JSON | `services/filesystem.go` Resolve, `extract/text.go`, `extract/ids.go` |
| `backend/list` (Cycle 2) | folder → file tree | `services/filesystem.go` walkDir |
| `backend/save` (Cycle 3) | blocks → write back to disk | `extract/docx.go` WriteDOCX |
| wire-up (Cycle 4) | ONE new `App` method per loop, old methods stay | `frontend/app.go`, `backend/main.go` (append-only) |
| retire (Cycle 5) | delete old packages only when all loops are green | everything above |

## Cycles

- [x] Cycle 1: open-file loop — `backend/read`, txt/md only (`cycle-01-open-file.md`)
- [x] Cycle 2: library list — `backend/list`, folder vs file tree tokens (`cycle-02-list.md`)
- [x] Cycle 3: save — `backend/save`, write blocks back (`cycle-03-save.md`)
- [x] Cycle 4 (owner-entrusted full remake, 2026-10): `model/` single type home;
  parsers unified in `read/` (+pdf/docx/normalize/context/dump); `Library` in
  `list/`; docx-write in `save/`; `store/` moved from services; aliases, bridge,
  internal/, models/, extract/, services/ deleted; Backend slimmed (dead
  wrappers dropped); dump-backed OpenFile + ReadDumpFile; `backend.ts` +
  ReaderMode switched. See `docs/journey/01-maps/11-file-by-file-backend.md`.

## The one loop to memorize (Cycle 1)

```
click token -> Resolve (jail) -> Stat (caps) -> Parse (blocks JSON)
  -> Dump to <data>/temporary/extract_dump/*.json -> return Doc
```

`dataRoot ""` means "ask `apppaths`" — same binary dumps to `.dev-data`
under `dev.bat` and to `Conductino_Data` when installed.
