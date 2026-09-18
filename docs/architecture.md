# Lumen — Architecture

Desktop research browser + AI-assisted reading workspace.
Stack: **Wails (Go) · React 19 · TypeScript · Vite · Tailwind CSS 4 · SQLite (planned)**.

## The seven layers (and who owns what)

| Layer | Location | Responsibility |
|---|---|---|
| 1. UI | `src/features/**`, `src/components/**` | Rendering + user intent. No business logic, no service calls except the two sanctioned boundaries below. |
| 2. Application state | `src/state/appState.tsx` | Single reducer + context. ALL state mutation flows through dispatched actions. |
| 3. Domain/workspace state | `src/types/domain.ts` | Pure types: sessions, tabs, sources, documents, changes, activities. |
| 4. Backend services | `frontend/src/services/backend.ts` (TS interfaces + mock + Wails impls) / `backend/{extract,tools,services}/*.go` | Filesystem walk/reveal/resolve, Workspace-owned library tree, extraction dispatch, storage (SQLite), source tools (mock). React never touches the OS — and never builds paths (it passes `FileTreeNode.path` tokens back opaquely). |
| 5. AI operations | `src/services/ai.ts` (provider boundary) / `backend/ai/` | All model access. The app speaks `AIProvider`, never a vendor SDK. |
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
   → (for Go) backend.* (mock in browser; Wails impls for library +
        filesystem in the desktop build)   [services/backend.ts]
        → real: window.go.frontend.App.* + EventsEmit("ai://event")
  → re-render from updated state
```

Domain state lives in `AppState` and mutates only via dispatched actions — with one deliberate exception: the library tree (`tree`/`treeError`/`locatePath` in `ReaderMode.tsx`) is server-mirror cache held in `useState`, refreshed from `backend.library.list()`. It is re-fetchable at any time and never the source of truth for documents. Everything else `useState` holds stays view-local (open menus, drafts, drag widths) — and even panel widths live in `AppState.*.ui` so they survive mode switches.

## Project structure

```
├── main.go                              Thin router: frontend.NewApp() → frontend.Run(app)
├── wails.json                           Wails project config (run `wails dev` from here)
├── frontend/
│   ├── main.go, app.go                  package frontend (importable): ALL Wails concerns —
│   │                                     bound App shell, asset embed (all:dist), wails.Run
│   ├── index.html, vite.config.ts       Vite + singlefile build
│   ├── wailsjs/                         Generated bindings output dir
│   └── src/
│       ├── App.tsx                      Shell: top bar, mode switch, modals, toast
│       ├── types/domain.ts              Domain model (all shared types)
│       ├── state/
│       │   ├── appState.tsx             Reducer + context + selectors
│       │   └── aiController.ts         Run AI operations, stream into state
│       ├── services/
│       │   ├── ai.ts                    WailsAIProvider (Go/Gemini via StreamAIRequest)
│       │   └── backend.ts               Service interfaces + browser-mode nulls + Wails impls
│       ├── components/                  icons.tsx, ui.tsx (Button/Menu/Modal/ResizablePanel/…)
│       └── features/
│           ├── browser/                 BrowserMode, WorkspaceSidebar, BrowserChrome,
│           │                            MockWebPage (BrowserView), AIBrowsePanel
│           └── reader/                  ReaderMode (owns library-tree state), ReaderSidebar
│                                        (Library panel + Documents panel), ReaderTabs,
│                                        DocumentView (source), SummaryDocumentView,
│                                        AIReadingPanel
├── backend/                             Pure Go (zero Wails imports)
│   ├── main.go                          Bridge only: Backend aggregates one service each
│   ├── models/                          document.go (library/open) / ai.go (wire) /
│   │                                     storage.go (records) — mirrors TS domain.ts
│   ├── extract/                         Pure extraction, no network: extract (dispatch +
│   │                                     typed errors) / text / docx / pdf / ids (stable
│   │                                     block IDs) / normalize + page (plan 07 stubs)
│   ├── tools/                           Folder-scoped harness tools, no model calls:
│   │                                     tools (registry/dispatch/audit) / paths (Resolve
│   │                                     jail + did-you-mean) / search (keyword + caps)
│   ├── usage/                           Telemetry only: usage (meters/RPM) / policy
│   │                                     (cost classes, semaphore, explain cache)
│   ├── ai/                              Network ONLY: service (Run/failover) / backends
│   │                                     (gemini, openai_compat) / prompts / chat (tool loop)
│   └── services/                        filesystem (walk/reveal/resolve) / workspace (owns
│                                        library tree, composes Filesystem) /
│                                        storage (interface + SQLite + memory) /
│                                        ai_shim + extract_alias (one-release aliases)
├── docs/                                This documentation
└── tasks.md                             Authoritative bugs/design/extraction plan (read first)
```

## React ↔ Go (Wails) communication

- **Binding:** `App` methods are bound in `frontend/main.go` (`Run`, `Bind: app`). The frontend calls `window.go.frontend.App.ListLibraryTree()` etc. (Wails generates `window.go` — the glue lives in the generated file, business logic does not). Only `frontend/*.go` may import `wailsapp` packages; `backend/` stays pure Go.
- **Streaming:** long-running work (AI) emits `runtime.EventsEmit(ctx, "ai://event", ev)`; the frontend listens and feeds `AIProvider` handlers. `App.StreamAIRequest` shows the shape. NOTE: no TS code calls it yet — `aiController.ts` uses the TS `AIProvider` only; the Go AI path is staged, not wired.
- **Mock vs Wails:** `createBackend()` in `src/services/backend.ts` probes `window.go.frontend.App`. Browser mode → mocks for everything. Desktop build → real `WailsFilesystem` + `WailsLibrary` (dialog, walk, `.txt` open, reveal); storage/sources/workspace/AI stay mocked in both modes.
- **Library/file split:** raw OS capability lives in `Filesystem` (`walkDir`, `Resolve` with workspace containment, per-OS reveal); the curated tree the UI shows is owned by `Workspace`, which composes the same `Filesystem` instance. `FileTreeNode.path` is a root-relative opaque token — the UI displays `label` and sends `path` back, never builds paths.

### Running under Wails

`wails.json` exists at the repo root; the desktop entrypoint is root `main.go` (thin router into `package frontend`).

1. `cd frontend && npm run build` → `frontend/dist` (embedded via `//go:embed all:dist` in `frontend/main.go`).
2. `wails dev` from the repo root — serves the real UI in the desktop shell.
3. Checks: `go vet ./...` + `go build ./...` (root); `tsc --noEmit` (via `node node_modules/typescript/bin/tsc` in `frontend/`).

`npm run dev` (inside `frontend/`) runs the full application with null-services — no folder dialog, no library tree, no extraction. The two modes are deliberately NOT equivalent; failures surface as honest toasts, never fabricated documents.

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
