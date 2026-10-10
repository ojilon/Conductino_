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

## Optional cleanup

`frontend/src/mock/data.ts` can still be retagged at source with `workspaceId: "ws-demo"`
and an inline `workspace` block in `createInitialState` so the seed wrapper is no longer
needed. That is cosmetic: the wrapper already injects the same tags at runtime.

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
