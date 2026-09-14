# Conductino_ — desktop research browser + AI-assisted reader

```
THE README NOT YET CORRECTED
```

A real project foundation: **Wails (Go) · React · TypeScript · Vite · Tailwind CSS 4 · SQLite (planned)**.
Two application modes — **Browser** (research sessions with page tabs + AI Browse) and **Reader**
(document subtabs, AI reading companion, editable research summary with reviewed AI changes).

## Quick start

```bash
npm install
npm run dev        # full app with mock services (browser preview)
npm run build      # single-file dist/index.html
```

The Go side (`backend/`) is staged for the Wails shell — see
`docs/architecture.md` §"Running under Wails". No Go toolchain is required
to run the UI; every subsystem it will serve has a mock behind the same
interface today.

## Try these flows

- **Browser:** switch sessions (left sidebar) · navigate the mock article (links work) ·
  run **AI Browse** and watch the streaming phases fill ranked source cards ·
  **Send to Reader** a result · open the source preview.
- **Reader:** select any sentence in Paper A → toolbar appears → **Ask AI** / **Go deeper** /
  **Include in summary** (proposes a highlighted change in the Research Summary) / **Save note** ·
  open the **Research Summary** tab → **Review AI changes**: accept / reject / inspect / revise ·
  edit the summary text directly (it is editable) · open `supplementary.pdf` from the file tree.
- **Panels:** drag the right panel edge to resize (double-click resets) · collapse either left
  panel · the workspace expands accordingly.

## Documentation map (answers to the obvious questions)

| Question | Doc |
|---|---|
| How is the app structured? Layers? Wails ↔ React ↔ Go? | `docs/architecture.md` |
| Where does document state live? What is persisted? | `docs/state-model.md` |
| **Where do I plug in the real AI API?** | `docs/ai-integration.md` |
| **Where do I plug in the PDF renderer / DOCX parsing?** | `docs/document-rendering.md` |
| **Where does SQLite belong?** | `docs/architecture.md` + `backend/services/storage.go` |
| **How do I replace a mock service?** | `docs/architecture.md` §checklist |
| What is implemented vs mocked vs placeholder? | `docs/future-work.md` |

## Status in one line

UI **WORKING** · app state **WORKING** · AI boundary **WORKING** with **MOCK** provider ·
summary editor **BASIC WORKING** · browser engine / PDF / DOCX **PLACEHOLDER boundaries** ·
SQLite **boundary ready, in-memory today** · Go services **staged with mock impls**.
(No fake buttons: every visible control either works or points at a documented integration point — the Settings dialog reports each subsystem's live status.)

## Lightweight by design

No Electron, no state library, no icon/font/animation frameworks beyond
Tailwind + two Google fonts, no diff engine, no webview — the scaffold
runs comfortably on a low-spec Windows machine, and each future
integration (pdf.js, provider SDKs, SQLite) is additive, not a rewrite.
