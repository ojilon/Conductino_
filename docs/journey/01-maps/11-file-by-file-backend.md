# 01-11 File-by-file: backend

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: one paragraph per Go file. Map below is post-remake (2026-10); `docs/backend-redo/PLAN.md` is the build log.
Exercise: fill one line per file: 'owns X, never does Y'.
Done when: map complete.

## Current package map (remade — verify each line yourself)

- `backend/main.go` — pure-Go `Backend{lib, storage, ai, mirrors, workspaceID}`: cache orchestration, workspace ids, dump-backed OpenFile, thread/chat persistence. Owns NO parsing, NO SQL, NO model calls.
- `backend/model/` — single wire/record home: `docs.go` (Doc/Node/Source), `blocks.go` (Block/Segment + marshal/parse), `fail.go` (Fail + reasons), `ai.go`, `storage.go`. No aliases, imports nothing.
- `backend/read/` — all content: `open.go` (UI + tools entry), `resolve.go` (jail), `text.go`/`pdf.go`/`docx.go` (arms), `normalize.go`, `context.go` (packs), `dump.go` (extract_dump). Never net/http.
- `backend/list/` — library: `library.go` (Library root + Resolve adapter), `list.go` (walk + Flat), `reveal.go` (OS reveal).
- `backend/save/` — write-back: `save.go` (txt/md), `docx.go` (WriteDOCX + summaries path).
- `backend/store/` — `storage.go` (memory + interface), `sqlite_storage.go`. Records live in model.
- `backend/ai/` — provider service (Gemini/Groq/OpenRouter, failover, chat loop). Takes narrow `tools.*` interfaces; calls `tools.` qualified (no aliases).
- `backend/tools/` — tool registry (`tools.go`, `paths.go`, `search.go`). Jail + honest failures.
- `backend/mirror/` — summary `.md` working copies + op log.
- `backend/skills/`, `backend/usage/` (meters + policy), `backend/workflows/`, `backend/apppaths/` (app-data / D: drive), `backend/version/`.
- DELETED in the remake: `models/`, `extract/`, `services/`, `bridge/`, `internal/`, all `*_alias.go`, `documents_heavy.go`, legacy App/Backend wrappers (ListTree, RunAI, doc/change persistence).

## History (why older notes mention other paths)

Pre-remake layout (`services/documents.go`, `backend/extract/`, alias files) entered via merges; the remake replaced it file-by-file, porting tests. If a journey note names a deleted path, check this map first.
