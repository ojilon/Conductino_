# Lumen — Architecture

Desktop research browser + AI-assisted reading workspace.
Stack: **Wails (Go) · React 19 · TypeScript · Vite · Tailwind CSS 4 · SQLite (planned)**.

## The seven layers (and who owns what)

| Layer | Location | Responsibility |
|---|---|---|
| 1. UI | `src/features/**`, `src/components/**` | Rendering + user intent. No business logic, no service calls except the two sanctioned boundaries below. |
| 2. Application state | `src/state/appState.tsx` | Single reducer + context. ALL state mutation flows through dispatched actions. |
| 3. Domain/workspace state | `src/types/domain.ts` | Pure types: sessions, tabs, sources, documents, changes, activities. |
| 4. Backend services | `src/services/backend.ts` (TS interface + mock) / `backend/services/*.go` | Filesystem, storage, source extraction, workspace. React never touches the OS. |
| 5. AI operations | `src/services/ai.ts` (provider boundary) / `backend/services/ai.go` | All model access. The app speaks `AIProvider`, never a vendor SDK. |
| 6. Document rendering | `src/features/reader/DocumentView.tsx`, `SummaryDocumentView.tsx`, `src/features/browser/MockWebPage.tsx` | DocumentModel → pixels. Swappable per format. |
| 7. Persistence | `backend/services/storage.go` (in-memory today, SQLite tomorrow) | Sessions, sources, documents, changes, activities, bookmarks. |

**Rule of thumb:** a file in layer 1 may import types (3) and use `useApp()` (2). It may import a service (4/5) ONLY inside the files that already define that boundary's mock — those files are the plug points, documented in the docs linked below.

## Data flow (one request end to end)

```
User action (React event)
  → dispatch(action)                      [state/appState.tsx]
  → reducer updates AppState              [pure, synchronous]
  → (for AI) useAIRunners.runX() calls AIProvider.run()
       → provider streams handlers.onPhase/onDone
       → each handler dispatches actions  [state/aiController.ts]
  → (for Go) backend.* (mock today)       [services/backend.ts]
       → real: window.go.main.App.* + EventsEmit("ai://event")
  → re-render from updated state
```

No component ever holds domain state in `useState`. `useState` is reserved for view-local concerns (open menus, drafts, drag widths) — and even panel widths live in `AppState.*.ui` so they survive mode switches.

## Project structure

```
├── index.html, vite.config.ts          Vite + singlefile build
├── src/
│   ├── App.tsx                          Shell: top bar, mode switch, modals, toast
│   ├── types/domain.ts                  Domain model (all shared types)
│   ├── state/
│   │   ├── appState.tsx                 Reducer + context + selectors
│   │   └── aiController.ts             Run AI operations, stream into state
│   ├── services/
│   │   ├── ai.ts                        AIProvider registry + MockAIProvider
│   │   └── backend.ts                   Go service interfaces + mock backend
│   ├── mock/
│   │   ├── data.ts                      Sessions, documents, sources, changes, file tree
│   │   └── aiContent.ts                Mock AI response templates
│   ├── components/                      icons.tsx, ui.tsx (Button/Menu/Modal/ResizablePanel/…)
│   └── features/
│       ├── browser/                     BrowserMode, WorkspaceSidebar, BrowserChrome,
│       │                                MockWebPage (BrowserView), AIBrowsePanel
│       └── reader/                      ReaderMode, ReaderSidebar, ReaderTabs,
│                                        DocumentView (source), SummaryDocumentView,
│                                        AIReadingPanel
├── backend/                             Go side (Wails boundary)
│   ├── main.go, app.go                  Shell + thin bindable methods
│   ├── models/models.go                 Go mirror of the TS domain types
│   └── services/                        ai / filesystem / storage / documents / workspace
└── docs/                                This documentation
```

## React ↔ Go (Wails) communication

- **Binding:** Go methods on `App` and the services are listed in `main.go` `Bind:`. The frontend calls `window.go.main.App.ShowContainingFolder(path)` etc. (Wails generates `window.go` — the glue lives in the generated file, business logic does not).
- **Streaming:** long-running work (AI) emits `runtime.EventsEmit(ctx, "ai://event", ev)`; the frontend listens and feeds `AIProvider` handlers. `app.StreamAIRequest` shows the shape.
- **Mock mode:** `createBackend()` in `src/services/backend.ts` checks for `window.go` and returns the mock today. Swapping to Wails changes one function.

### Running under Wails

This repository is a Vite project; the Go side is staged under `backend/`. To make it a real desktop app:

1. `wails init -n lumen` (or copy `backend/` to the module root and write a `wails.json` pointing `frontend:directory` at this repo).
2. Build the frontend (`npm run build`) into `frontend/dist`.
3. `wails dev` — the embedded `assets` in `main.go` then serve the real UI.

Until then, `npm run dev` / the built `dist/index.html` run the full application with mock services.

## Browser integration boundary

- `BrowserTab.nav` (URL, history, index) is the **NavigationState** a real webview would own. Back/forward/reload/address entry already mutate it.
- `MockWebPage` is the **BrowserView** placeholder: it renders two realistic mock articles, a mock search-results page, a new-tab page, and a fallback — and its links drive real tab navigation.
- **To integrate a real browser:** replace the `<MockWebPage>` usage in `BrowserMode.tsx` with a component that mounts the Wails/Go webview per tab, feeding it `tab.nav.currentUrl` and reporting navigation back via `dispatch({type:"browser.navigate"})`. Nothing else changes.

## Diff / proposal system (boundary for a real engine)

AI edits flow as `DocumentChange { id, type: insert|modify|delete, blockId, oldContent, newContent, highlightFragment, sourceId, activityId, status: pending|accepted|rejected }`.

- The mock uses whole-block granularity + an optional `highlightFragment` substring for the "modified" underline.
- The UI already distinguishes **inserted** (amber card, green text), **modified** (amber underline), **deleted** (strike-through, type supported), and shows accept / reject / inspect / revise.
- **To integrate a real diff engine:** replace how `newContent`/`highlightFragment` are produced (in the AI layer) and optionally move the granularity from block to character range. The reducer cases `change.propose` / `change.decide` / `change.revise` are the only code that touches document blocks for changes.

## Replacing a mock service (checklist)

1. Find the boundary file (`src/services/ai.ts` or `src/services/backend.ts`).
2. Implement the existing interface with real code — do not change the interface unless the capability truly changed.
3. Register it (`setAIProvider(...)` / `createBackend()` returns the Wails implementation).
4. Update the status row in the Settings dialog (it reads the live provider/backend) and the status table in `docs/future-work.md`.
5. Delete the now-unused mock when comfortable.
