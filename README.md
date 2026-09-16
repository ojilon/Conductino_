# Conductino — desktop research browser + AI-assisted reader

A real project foundation: **Wails (Go) · React · TypeScript · Vite · Tailwind CSS 4 · SQLite (planned)**.
Two application modes — **Browser** (research sessions with page tabs + AI Browse) and **Reader**
(document subtabs, AI reading companion, editable research summary with reviewed AI changes).

> **Stability note:** this project is mid-migration and parts are unstable by
> design. The folder-open / library / `.txt`-extraction pipe is newly wired
> and has known bugs (stale tabs on folder switch, silent mock fallback on
> read failure); multi-session tracking, Slate integration, and real PDF/DOCX
> extraction are designed but unbuilt. `docs/` matches the tree again; the
> authoritative inventory — what exists, what's wrong, why, and
> what to do — is **`tasks.md`** at the repo root. Read it before changing
> anything in `backend/` or the reader sidebar.

## Quick start

Two ways to run — they are NOT equivalent:

```bash
cd frontend
npm run dev        # browser preview: full UI, ALL services mocked (no folder dialog)
```

```bash
wails dev          # desktop build from repo root: library + filesystem go through real Go code
                   # (.txt/.md open for real; pdf/docx fall back to mock render)
```

Checks: `go vet ./...` + `go build ./...` from root; `node node_modules/typescript/bin/tsc --noEmit` from `frontend/`.

## Layout (current, not the old docs' version)

- `main.go` — thin router only: `frontend.NewApp()` → `frontend.Run(app)`.
- `frontend/app.go` + `frontend/main.go` (`package frontend`, importable) — ALL
  Wails concerns: bound `App` shell, asset embedding, `wails.Run`. Bound as
  `window.go.frontend.App.*`.
- `backend/main.go` (`package backend`, zero Wails imports) — pure-Go bridge:
  `Backend` aggregates one service instance and forwards to exactly one
  service per method.
- `backend/services/` — `filesystem.go` (walk/reveal/resolve), `workspace.go`
  (owns the library tree, composes `Filesystem`), `documents.go` (extraction;
  `.txt`/`.md` real, rest `ErrUnsupportedType`), `storage.go` (in-memory),
  `ai.go` (mock provider). `backend/models/` mirrors the TS domain types.
- `frontend/src/services/backend.ts` — service boundary: `Wails*`
  implementations when `window.go` exists, mocks otherwise. Library +
  filesystem are live in the desktop build; storage/sources/workspace/AI are
  mocked in both modes.

## Try these flows

- **Library (desktop build):** Reader mode → **Library** rail → **Choose folder**
  → OS picker → real recursive tree renders. Click a `.txt` file → new tab with
  real extracted paragraphs, toast `Opened <name>`. Click a `.pdf` → mock render
  with `(mock extraction)` toast (parsers not built yet). **Documents** panel →
  **Show in library** jumps to the file's tree location.
- **Browser:** switch sessions (left sidebar) · navigate the mock article (links work) ·
  run **AI Browse** and watch the streaming phases fill ranked source cards ·
  **Send to Reader** a result · open the source preview.
- **Reader:** select any sentence in Paper A → toolbar appears → **Ask AI** / **Go deeper** /
  **Include in summary** (proposes a highlighted change in the Research Summary) / **Save note** ·
  open the **Research Summary** tab → **Review AI changes**: accept / reject / inspect / revise ·
  edit the summary text directly (raw `contentEditable`, no Slate yet).
- **Panels:** drag the right panel edge to resize (double-click resets) · collapse either left
  panel · the workspace expands accordingly.

## Documentation map (answers to the obvious questions)

| Question | Doc |
|---|---|
| What is broken / missing / planned, with file:line refs? | **`tasks.md`** (read first) |
| How is the app structured? Layers? Wails ↔ React ↔ Go? | `docs/architecture.md` |
| Where does document state live? What is persisted? | `docs/state-model.md` |
| **Where do I plug in the real AI API?** | `docs/ai-integration.md` |
| **Where do I plug in the PDF renderer / DOCX parsing?** | `docs/document-rendering.md` + `tasks.md` §4 (library options) |
| **Where does SQLite belong?** | `docs/architecture.md` + `backend/services/storage.go` |
| What is implemented vs mocked vs placeholder? | `docs/future-work.md` status table (known bugs live in `tasks.md` §1) |

## Status in one line

UI **WORKING** · app state **WORKING** · AI boundary **WORKING** with **MOCK** provider ·
folder-open + library tree + `.txt` open **WORKING in desktop build, with known bugs (see `tasks.md` §1)** ·
pdf/docx open **MOCK fallback** · summary editor **raw contentEditable, no Slate** ·
SQLite **boundary ready, in-memory today** · chosen folder **not persisted (resets on restart)**.
(Browser preview remains fully mocked — no dialog, mock tree, mock extraction.)

## Lightweight by design

No Electron, no state library, no icon/font/animation frameworks beyond
Tailwind + two Google fonts, no diff engine — the scaffold targets a low-spec
Windows machine (4 GB RAM, Celeron), and each future integration (pdf.js,
provider SDKs, SQLite, extraction libs) is additive, not a rewrite. Extraction
library candidates were screened against this constraint — see `tasks.md` §4.
