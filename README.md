# Conductino — desktop research browser + AI-assisted reader

A real project foundation: **Wails (Go) · React · TypeScript · Vite · Tailwind CSS 4 · SQLite (default)**.
Two application modes — **Browser** (research sessions with page tabs + AI Browse) and **Reader**
(document subtabs, PDF canvas + text views, AI chat with `@doc` targeting, editable research
summary with reviewed AI changes).

> **Stability note:** this project is mid-migration and parts are unstable by
> design. The folder-open / library / extraction pipe is wired with typed
> failure reasons (no mock fallback); the chosen folder persists across
> restarts; summaries edit in Slate with user-gated AI proposals. Known open
> items: stale-tab handling on folder switch is tag-and-warn only, browser
> engine is still mocked, scanned PDFs are out of scope. `docs/` matches the
> tree again; the authoritative inventory — what exists, what's wrong, why,
> and what to do — is **`tasks.md`** at the repo root plus
> **`docs/plans/04-implementation-phases.md`**. Read them before changing
> anything in `backend/` or the reader sidebar.

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
- `backend/tools/` — folder-scoped tools: `tools.go` (registry, dispatch,
  catalog, audit), `paths.go` (Resolve jail + did-you-mean), `search.go`
  (keyword search + caps). No model calls, no shell.
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
  **Include in summary** (proposes a highlighted change in the Research Summary) / **Save note** ·
  open the **Chat** tab → ask with `@Title` to target a document, select text to anchor
  a region, say "shorten this proposal" to revise a pending change ·
  open the **Research Summary** tab → review inline proposals: accept / reject / revise ·
  edit the summary directly in Slate (autosaves to `.docx`; manual **Save DOCX** too).
- **Panels:** drag the right panel edge to resize (double-click resets) · collapse either left
  panel · the workspace expands accordingly.

## Documentation map (answers to the obvious questions)

| Question | Doc |
|---|---|
| What is broken / missing / planned, with file:line refs? | **`tasks.md`** (read first) |
| How is the app structured? Layers? Wails ↔ React ↔ Go? | `docs/architecture.md` |
| Where does document state live? What is persisted? | `docs/state-model.md` |
| **Where do I plug in the real AI API?** | `docs/ai-integration.md` (+ `docs/plans/05-multi-provider-apis.md` for Groq/OpenRouter/failover) |
| **Where do I plug in the PDF renderer / DOCX parsing?** | Landed: `PdfView.tsx` (pdf.js canvas) + `backend/extract/pdf.go`, `docx.go`; see `docs/document-rendering.md` + `tasks.md` §4 |
| **Where does SQLite belong?** | Landed default: `backend/services/sqlite_storage.go` (+ `extract_cache`); `docs/architecture.md` |
| What is implemented vs mocked vs placeholder? | `docs/future-work.md` status table (known bugs live in `tasks.md` §1) |
| What is the delivery order / what lands next? | `docs/plans/04-implementation-phases.md` (Phases 0–13 done; 14–15 planned) |
| Backend resplit / paged reading / skills / releases? | `docs/plans/06-backend-resplit.md`, `07-source-reading.md`, `08-operations-growth.md` (proposals) |

## Status in one line

UI **WORKING** · app state **WORKING** · AI **REAL** (Gemini/Groq/OpenRouter via Go, failover + budgets + Settings meters) ·
folder-open + library tree + `.txt`/`.md`/`.docx`/`.pdf` open **WORKING in desktop build** (PDFs render on canvas; failures are typed, never mock) ·
summary editor **Slate with inline AI-change review + gated DOCX autosave** ·
SQLite **default with extract cache** · chosen folder **persisted across restarts**.
(Browser preview remains fully mocked — no dialog, mock tree, mock extraction.)

## Lightweight by design

No Electron, no state library, no icon/font/animation frameworks beyond
Tailwind + two Google fonts, no diff engine — the scaffold targets a low-spec
Windows machine (4 GB RAM, Celeron), and each integration landed additively:
pdf.js canvas leaf, stdlib-only DOCX, pure-Go SQLite (`modernc.org/sqlite`),
stdlib HTTP model clients (no vendor SDKs). Extraction library candidates
were screened against this constraint — see `tasks.md` §4. cgo stays out
until Go-side profiling says otherwise (`docs/plans/06-backend-resplit.md` §6).
