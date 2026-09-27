# Phase 2 — status

**Exit criteria met** on `feature/academic-harness-reader-ai`.

## Landed

| Piece | Status |
|-------|--------|
| Domain `WorkspaceSession` + `workspaceId` | done |
| `actions` / `selectors` / `reduceWorkspace` / reduce parts | done |
| `aiController` → `primarySummaryDocument` | done |
| `ReaderMode` folder pick + openFile `workspaceId` | done |
| Mock seed: workspace + runtime tags on sources/docs/changes | done (via `createInitialState` wrapper in `appState.tsx`) |

## Optional cleanup — DONE

`src/mock/` is deleted. `createInitialState` lives directly in
`state/appState.tsx` (empty chrome + inline `ws-demo` workspace block, no
wrapper, no retagging) and "Send to Reader" builds documents from real
source fields in `AIBrowsePanel.tsx`. The snippet below is kept as the
record of what the wrapper used to inject:

```ts
// createInitialState workspace block (optional, already injected by appState):
workspace: {
  activeId: "ws-demo",
  byId: {
    "ws-demo": {
      id: "ws-demo",
      rootPath: null,
      primarySummaryId: "doc-summ",
      label: "Demo research workspace",
    },
  },
},
```
