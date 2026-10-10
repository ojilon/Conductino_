# 01-11 File-by-file: backend/services

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: one paragraph per Go file. Start with the map below, then verify each
line yourself against the tree (this file tracks a moving target — see sweep note).
Exercise: fill one line per file: 'owns X, never does Y'.
Done when: map complete.

## Sweep note (2026-10-10): restructure arrived via merge, not by you

The `backend/extract/`, `backend/tools/`, `backend/mirror/`, `backend/skills/`,
`backend/usage/`, `backend/workflows/`, `backend/apppaths/`, `backend/ai/`,
`backend/version/` packages were NOT your doing and NOT this session's doing.
They are prior AI-agent work from main-side history (e.g. `705b568 restructure
backend`, `aafe802 massive restructure by ai agent`) that entered this branch
through merge `e771890` (main `a48a1dc` + journey `5242c5d`). Consequence: the
old `backend/services/documents.go|docx.go|pdf.go|blockids.go` + `services/ai/*`
layout this journey was written against no longer exists — extraction now lives
in `backend/extract/`, with `backend/services/extract_alias.go` re-exporting it
for compat. Sweep the NEW layout below, file by file, before trusting any older
journey reference to `services/documents.go`.

## Current package map (verify each line yourself)

- `backend/main.go` — pure-Go `Backend` bridge: aggregates one service each,
  forwards one method per call. Owns NOTHING except wiring + extract-cache
  orchestration + last_library_root restore.
- `backend/bridge/bridge.go` — (journey step 3) versioned heavy↔network contract.
- `backend/internal/heavy/` — (step 3) pure txt/md parse, no os/net/exec.
- `backend/internal/netops/` — doc only: what stays network.
- `backend/extract/` — CANONICAL extraction: `extract.go` (dispatch + OpenError/
  ReasonOf + emptyDocument), `text.go` (txt/md arm), `docx.go`, `pdf.go`,
  `page.go`, `context.go` (context packs), `ids.go` (stable IDs), `normalize.go`,
  `doc.go`. Owns bytes→blocks; never touches net/http.
- `backend/services/` — filesystem (walk/resolve/reveal jail), workspace
  (curated tree), storage + sqlite_storage, `ai_shim.go`,
  `extract_alias.go` (compat re-exports — do not add logic here),
  `documents_heavy.go` (split-path adapter + parity test, step 3).
- `backend/ai/` — provider service (Gemini/Groq/OpenRouter, failover, chat loop);
  `tools_alias.go` re-exports `backend/tools/`. Check which side owns what.
- `backend/tools/` — tool registry (`tools.go`, `paths.go`, `search.go`).
  Sweep: jail guards vs duplicates of extract logic.
- `backend/mirror/` — mirror concept (`mirror.go`, `PlainToMarkdown`).
- `backend/models/` — split twins: `models.go`, `document.go`, `ai.go`,
  `storage.go`. Sweep: which struct mirrors which TS type in `domain.ts`.
- `backend/skills/`, `backend/usage/` (meters + policy), `backend/workflows/`,
  `backend/apppaths/` (app-data / D: drive), `backend/version/`.

## Open sweep questions (yours to answer, one per session)

1. `heavy.ParseText` vs `extract.openTextFile`: duplicate FNV-ID scheme + caps —
   intended bridge step, or accidental duplication? (Empty-file case already
   diverges: `b-empty` vs content hash — see parity-test fix.)
2. `services/extract_alias.go` + `ai/tools_alias.go`: thin compat or hiding
   real callers? List who imports the alias vs the canonical package.
3. `models/` split into 4 files vs one TS `domain.ts`: twins still exact?
   Add a `// TS:` twin comment per struct as you verify each.
