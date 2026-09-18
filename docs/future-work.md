# Lumen — Future work & status

## Subsystem status (honest inventory)

| Subsystem | Status | Where |
|---|---|---|
| App shell, mode switch, top bar | **WORKING** | `src/App.tsx` |
| Browser UI (sessions, tabs, address bar, navigation, mock pages, search page, new tab) | **WORKING** (navigation real, page content mock) | `src/features/browser/` |
| Reader UI (subtabs, document info, TOC, file tree, AI panels, resize/collapse) | **WORKING** (tree moved to Library view; Documents panel slimmed) | `src/features/reader/` |
| Folder pick → library tree (recursive walk, path tokens, locate-in-tree) | **WORKING in desktop build / MOCK tree in browser** | `frontend/app.go:SelectFolder`, `backend/services/filesystem.go:walkDir`, `ReaderSidebar.tsx` LibraryPanel |
| `.txt`/`.md` file open (real bytes → blocks → tab) | **WORKING in desktop build** (2 MiB / 1000-block caps) | `App.OpenFile`, `backend/services/documents.go:OpenFile` |
| Open-failure reporting | **MISSING — all failures collapse to mock fallback** (see `tasks.md` §1.2) | `backend.ts:openFile`, `ReaderMode.tsx:openFile` |
| Stale tabs on folder switch | **BUG, not started** (see `tasks.md` §1.1) | `Filesystem.root`, `metadata.path` |
| Chosen-folder persistence across restarts | **MISSING** (memory only) | `backend/services/workspace.go:Save` |
| Multi-session source→summary tracking | **DESIGN NOTE only** (first-summary-wins today, see `tasks.md` §2) | `aiController.ts:122` |
| Application state (reducer, selectors, persistence-ready model) | **WORKING** | `src/state/appState.tsx` |
| Domain types (sessions/sources/documents/changes/activities) | **WORKING** | `src/types/domain.ts` |
| AI provider boundary + streaming UI (phases, activity tracking, history) | **WORKING** (Gemini/Groq/OpenRouter via Go; cost class + semaphore + explain cache; usage meters in Settings) | `src/services/ai.ts`, `backend/services/ai/` |
| AI responses (explanations, insertions, revisions) | **REAL** (failures surface as UI errors, no mock fallback) | `backend/services/ai.go` |
| AI web-search ranking (browser) | **NOT CONNECTED** (honest error, by design) | `GeminiAIService.Run` |
| Selection → AI action workflow | **WORKING** (block anchor + range offsets) | `DocumentView.tsx` |
| Highlights / saved notes | **WORKING** (block-anchored; range-precise spans where measured) | `doc.highlight.add` |
| Summary document editing | **WORKING** (Slate; pending modifies inline, deletes/inserts card-based; gated autosave to DOCX) | `SummaryDocumentView.tsx` |
| AI change proposals (insert/modify/delete, accept/reject/inspect/revise) | **WORKING** (tool + chat driven; user gatekeeps every change) | `change.*` actions |
| Chat `@doc` targeting + region selection | **WORKING** (`mentionIds` + selection anchor; autocomplete in composer) | `state/mentions.ts`, `aiController.runChat` |
| Source → summary provenance (`sourceIds`, cited-sources list) | **WORKING** | `summary.addSource` |
| Source preview / save / send-to-reader | **WORKING** | `AIBrowsePanel.tsx` |
| Browser engine (real web content) | **PLACEHOLDER** (integration boundary) | `MockWebPage.tsx` |
| PDF / DOCX rendering & extraction | **WORKING** (pdf: Go text-layer extract + canvas leaf w/ text layer; docx: stdlib extract + Save; scanned PDFs out of scope) | `PdfView.tsx`, `backend/services/pdf.go`, `documents.go` |
| Go backend services | **PARTIAL: filesystem walk/reveal/resolve + workspace library tree + `.txt` extraction real; storage + AI mock** | `backend/services/` |
| Wails wiring (bindings, events, embed) | **WORKING for library+filesystem** (`WailsFilesystem`/`WailsLibrary` in `backend.ts`); AI events + storage unwired | `frontend/app.go`, `frontend/main.go`, root `main.go` (thin router) |
| SQLite persistence | **BOUNDARY READY / not implemented** (in-memory today) | `backend/services/storage.go` |
| Rich formatting in summaries (bold/italic/lists) | **FUTURE** | editor upgrade |
| Character-range selection & diffs | **PARTIAL** (block-text offsets live; PDF text-layer coords need a pdf.js leaf) | `DocumentView.tsx`, `slateAdapter.ts` |
| Skills / workflows / parallel APIs | **PROPOSAL** | `docs/plans/08-operations-growth.md` §§1–3 |
| Chat observability (tokens, model names, tool trace, logs DB) | **PROPOSAL** (usage + audit rings exist in memory; persistence + UI pending) | `docs/plans/08-operations-growth.md` §§4–5 |
| CI + releases (tags, installer drive choice, portable zip) | **PROPOSAL** | `docs/plans/08-operations-growth.md` §§6–7 |
| Bookmarking pages, collections, workspace library content | **FUTURE** (UI placeholders exist) | sidebar views |

## Suggested order of real integration

1. **Wails wiring** — DONE for library+filesystem (`wails dev` serves the frontend with a real tree + `.txt` open). Remaining: AI events, storage.
2. **Open-failure reporting** — typed reasons, mock-fallback only on `unsupported` (`tasks.md` §1.2). Do this BEFORE adding parsers, or parser bugs will hide as placeholders.
3. **Stale tabs on folder switch** — invalidate or root-tag (`tasks.md` §1.1).
4. **Chosen-folder persistence** — persist root via `Workspace.Save`/`Init`.
5. **PDF/DOCX extraction** — extraction-first into block/segment JSON per `tasks.md` §§3–4 (this reuses the entire AI/selection stack; canvas rendering stays optional).
6. **AI provider** — implement one real provider behind `AIProvider` (docs/ai-integration.md). Browse + explain immediately become real. (Includes wiring the staged Go AI path + fixing the `Selection` shape mismatch noted there.)
7. **SQLite** — `SQLiteStorage` behind `StorageService`; start with sources + sessions, then documents/changes/activities (schema sketched in `backend/services/storage.go`).
8. **Multi-session tracking** — `sessionId` design per `tasks.md` §2, BEFORE any multi-session UI.
9. **Slate adapter + DOCX save-out** — thin boundary adapters per `tasks.md` §3 steps 4–5, only once merges are production-quality.
10. **Browser engine** — Go webview behind `MockWebPage`; per-tab instances; forward navigation to `NavigationState`.
11. **Range-precise selection/highlights** — alongside the renderer upgrades.

## Replacing mock data

`src/mock/data.ts` is the seed for the in-memory state. Once storage exists,
`createInitialState()` becomes "defaults + load from backend" — the loader
calls `backend.*` (already promise-based) and dispatches the same actions.
Keep the file as fixtures for tests until the real corpus replaces it.
