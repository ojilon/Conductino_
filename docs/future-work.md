# Lumen — Future work & status

## Subsystem status (honest inventory)

| Subsystem | Status | Where |
|---|---|---|
| App shell, mode switch, top bar | **WORKING** | `src/App.tsx` |
| Browser UI (sessions, tabs, address bar, navigation, mock pages, search page, new tab) | **WORKING** (navigation real, page content mock) | `src/features/browser/` |
| Reader UI (subtabs, document info, TOC, file tree, AI panels, resize/collapse) | **WORKING** | `src/features/reader/` |
| Application state (reducer, selectors, persistence-ready model) | **WORKING** | `src/state/appState.tsx` |
| Domain types (sessions/sources/documents/changes/activities) | **WORKING** | `src/types/domain.ts` |
| AI provider boundary + streaming UI (phases, activity tracking, history) | **WORKING boundary / MOCKED provider** | `src/services/ai.ts` |
| AI responses (explanations, insertions, revisions, ranking) | **MOCKED** | `src/mock/aiContent.ts` |
| Selection → AI action workflow | **WORKING** (block-level precision) | `DocumentView.tsx` |
| Highlights / saved notes | **WORKING** (block-anchored) | `doc.highlight.add` |
| Summary document editing | **BASIC WORKING** (plain text, contentEditable) | `SummaryDocumentView.tsx` |
| AI change proposals (insert/modify/delete, accept/reject/inspect/revise) | **WORKING UI / MOCKED proposals** | `change.*` actions |
| Source → summary provenance (`sourceIds`, cited-sources list) | **WORKING** | `summary.addSource` |
| Source preview / save / send-to-reader | **WORKING** | `AIBrowsePanel.tsx` |
| Browser engine (real web content) | **PLACEHOLDER** (integration boundary) | `MockWebPage.tsx` |
| PDF / DOCX rendering & extraction | **PLACEHOLDER** (mock structured renderer) | `DocumentView.tsx`, `backend/services/documents.go` |
| Go backend services (filesystem, workspace, storage, AI) | **PLACEHOLDER / mock implementations** | `backend/services/` |
| Wails wiring (bindings, events, embed) | **PLACEHOLDER** (staged, see architecture.md §Running under Wails) | `backend/main.go` |
| SQLite persistence | **BOUNDARY READY / not implemented** (in-memory today) | `backend/services/storage.go` |
| Rich formatting in summaries (bold/italic/lists) | **FUTURE** | editor upgrade |
| Character-range selection & diffs | **FUTURE** (boundary documented) | architecture.md §Diff |
| Bookmarking pages, collections, workspace library content | **FUTURE** (UI placeholders exist) | sidebar views |

## Suggested order of real integration

1. **Wails wiring** — make `wails dev` serve this frontend; verify the mock app inside the shell (no code changes expected).
2. **AI provider** — implement one real provider behind `AIProvider` (docs/ai-integration.md). Browse + explain immediately become real.
3. **SQLite** — `SQLiteStorage` behind `StorageService`; start with sources + sessions, then documents/changes/activities (schema sketched in `backend/services/storage.go`).
4. **PDF rendering** — decide canvas (pdf.js) vs extraction-first (docs/document-rendering.md). Extraction-first reuses the entire AI/selection stack sooner.
5. **DOCX** — same decision per format.
6. **Browser engine** — Go webview behind `MockWebPage`; per-tab instances; forward navigation to `NavigationState`.
7. **Rich summary editor + real diff** — only once AI merges are production-quality.
8. **Range-precise selection/highlights** — alongside the renderer upgrades.

## Replacing mock data

`src/mock/data.ts` is the seed for the in-memory state. Once storage exists,
`createInitialState()` becomes "defaults + load from backend" — the loader
calls `backend.*` (already promise-based) and dispatches the same actions.
Keep the file as fixtures for tests until the real corpus replaces it.
