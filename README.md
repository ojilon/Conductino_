# Conductino — desktop research browser + AI-assisted reader

A real project foundation: **Wails (Go) · React · TypeScript · Vite · Tailwind CSS 4 · SQLite (default)**.
Two application modes — **Browser** (research sessions with page tabs + AI Browse) and **Reader**
(document subtabs, PDF canvas + text views, AI chat with `@doc` targeting, editable research
summary with reviewed AI changes).

> **Stability note:** mid-migration; parts are unstable by design. Folder-open /
> library / extraction use typed failures (no mock fallback); chosen folder
> persists; summary path uses mirrors + tools/skills/workflows on the harness tip.
>
> **Planning source of truth on this line of work:**
> `docs/plans/00-current-priorities.md` + `docs/plans/README.md`.
> **Next foundation:** `docs/release-prep/` (tags, installer drive choice, local storage).
> Prefer those over older phase lists when they conflict. `tasks.md` remains useful for file-level bugs.

## Quick start

Two ways to run — they are NOT equivalent (pnpm is canonical; never npm —
`pnpm-lock.yaml` is the lockfile):

```bash
cd frontend
pnpm dev             # browser preview: full UI, ALL services mocked (no folder dialog)
```

```bash
wails dev          # desktop build from repo root: library + filesystem + AI go through real Go code
                   # (.txt/.md/.docx/.pdf open for real; PDFs also render on canvas)
```

AI keys (chat needs at least one — restart `wails dev` after editing):

```bash
# backend/.ai.env (git-ignored; process env wins; last line wins on duplicates)
AI_MODE=auto
AI_PRIMARY=groq
GEMINI_API_KEY=<key>        # primary quality + long context (free tier 429s when exhausted)
GROQ_API_KEY=<key>          # fast secondary (GROQ_MODEL, default: openai/gpt-oss-20b)
OPENROUTER_API_KEY=<key>    # tertiary pool (OPENROUTER_MODEL, default: qwen3 free router)
```

Checks: `go vet ./...` + `go build ./...` from root; `pnpm exec tsc --noEmit` and `pnpm build` from `frontend/`.

## Layout (current, not the old docs' version)

- `main.go` — thin router only: `frontend.NewApp()` → `frontend.Run(app)`.
- `frontend/app.go` + `frontend/main.go` (`package frontend`, importable) — ALL
  Wails concerns: bound `App` shell, asset embedding, `wails.Run`. Bound as
  `window.go.frontend.App.*`.
- `backend/main.go` (`package backend`, zero Wails imports) — pure-Go bridge:
  `Backend` aggregates one service instance and forwards to exactly one
  service per method.
- `backend/services/` — `filesystem.go` (walk/reveal/resolve), `workspace.go`
  (owns the library tree, composes `Filesystem`), `storage.go` +
  `sqlite_storage.go` (SQLite default, in-memory fallback; includes the
  `extract_cache` table), `ai_shim.go` + `extract_alias.go` (one-release
  aliases, see plan 06). `backend/models/` mirrors the TS domain types
  (`document.go` / `ai.go` / `storage.go` records).
- `backend/extract/` — pure-Go extraction, no network: `extract.go`
  (dispatch + typed errors), `text.go`, `docx.go`, `pdf.go`, `ids.go`
  (stable content-addressed block IDs), `normalize.go` + `page.go`
  (plan 07 windowed-reading stubs).
- `backend/mirror/` — summary `.md` working copies (op log, re-sync; see plan 10).
- `backend/tools/` — folder-scoped tools: `tools.go` (registry, dispatch,
  catalog, audit), `paths.go` (Resolve jail + did-you-mean), `search.go`
  (keyword search + caps). No model calls, no shell.
- `backend/skills/` + `backend/workflows/` — skill loader + gated routines (plan 11).
- `backend/usage/` — telemetry only: `usage.go` (meters, RPM rings),
  `policy.go` (cost classes, semaphore, explain cache).
- `backend/ai/` — network-only provider package: `service.go` (failover
  orchestration), `backend.go` + `openai_compat.go` (Gemini/Groq/OpenRouter),
  `chat.go` (multi-turn tool loop over `tools.ToolHost`), `prompts.go`.
  Keys live in Go only (env or git-ignored `backend/.ai.env`), never cross
  into JS.
- `frontend/src/services/backend.ts` — service boundary: `Wails*`
  implementations when `window.go` exists, mocks otherwise. Library,
  filesystem, storage (SQLite), and AI are live in the desktop build;
  sources/workspace metadata stay mocked.

## Try these flows

- **Library (desktop build):** Reader mode → **Library** rail → **Choose folder**
  → OS picker → real recursive tree renders. Click a `.txt`/`.md`/`.docx` file →
  new tab with real extracted blocks, toast `Opened <name>`. Click a `.pdf` →
  canvas pages with selectable text (extraction feeds chat/tools behind it).
  Read failures surface typed reasons, never mock content. **Documents** panel →
  **Show in library** jumps to the file's tree location.
- **Browser:** switch sessions (left sidebar) · navigate the mock article (links work) ·
  run **AI Browse** and watch the streaming phases fill ranked source cards ·
  **Send to Reader** a result · open the source preview.
- **Reader:** select any sentence in a source → toolbar appears → **Ask AI** / **Go deeper** /
  **Include in summary** / **Save note** · open **Chat** with `@Title` targeting ·
  summary editing / review path per current harness UI (mirrors + tools).
- **Panels:** drag the right panel edge to resize (double-click resets) · collapse either left
  panel · the workspace expands accordingly.

## Documentation map (answers to the obvious questions)

| Question | Doc |
|---|---|
| **What next? (ordered)** | **`docs/plans/00-current-priorities.md`** + **`docs/plans/README.md`** |
| **Tags, installer (D: drive), local storage, pre-release data root** | **`docs/release-prep/`** |
| Mirrors, empty docs, page canvas | `docs/plans/10-live-library-and-intermediates.md` |
| Skills, workflows, package split, thinking trace | `docs/plans/11-skills-workflows-and-split.md` |
| File-level bugs / inventory | `tasks.md` |
| Layers / architecture | `docs/architecture.md` |
| Document state | `docs/state-model.md` |
| AI API | `docs/ai-integration.md` |
| PDF / DOCX rendering | `docs/document-rendering.md` |
| Status table | `docs/future-work.md` |

## Status in one line

UI **WORKING** · app state **WORKING** · AI **REAL** (multi-provider via Go) ·
library + extract **WORKING in desktop** · summary **mirrors + skills/workflows on harness tip** ·
SQLite **default** · chosen folder **persisted**.
(Browser preview remains largely mocked for page content.)

## Lightweight by design

No Electron, no state library, no icon/font/animation frameworks beyond
Tailwind + two Google fonts — targets a low-spec Windows machine.
Offline packages stay stdlib-first where possible (`extract`, `tools`, `skills`, `mirror`, `usage`).
