# Phase 2 — remaining local patches

GitHub write tools cap single-file content at ~4KB, so the full `data.ts` (27KB)
and `ReaderMode.tsx` (12KB) updates could not be pushed in one shot.

## Already on branch `feature/academic-harness-reader-ai`

- `frontend/src/types/domain.ts` — WorkspaceSession, workspaceId fields, AppState.workspace
- `frontend/src/state/aiController.ts` — primarySummaryDocument / activeWorkspace
- `frontend/src/state/actions.ts` — workspace.* action union
- `frontend/src/state/selectors.ts` — primarySummaryDocument, workspaceIdFromRoot, …
- `frontend/src/state/reduceWorkspace.ts` — ensure / setActive / setPrimarySummary / createSummary
- `frontend/src/state/reducePart{1a,1b,2,3,4}.ts` — split reducer (size limit)
- `frontend/src/state/appState.tsx` — thin provider + workspace seed wrapper

## Apply locally (from repo root)

### 1. `frontend/src/mock/data.ts`

Add `workspaceId: "ws-demo"` to every seeded Source, Document, and DocumentChange.
In `createInitialState()`, add:

```ts
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

(The appState wrapper already injects this if missing, so the app runs without it;
tagging is still required for correct multi-workspace mock behavior.)

### 2. `frontend/src/features/reader/ReaderMode.tsx`

```ts
import { useApp, activeReaderDocument, workspaceIdFromRoot } from "../../state/appState";
```

In `pickFolder`, after resolving `abs`:

```ts
const wsId = workspaceIdFromRoot(abs);
dispatch({
  type: "workspace.ensure",
  session: {
    id: wsId,
    rootPath: abs,
    primarySummaryId: null,
    label: abs.split(/[/\\]/).filter(Boolean).pop() ?? abs,
  },
});
dispatch({ type: "workspace.setActive", id: wsId });
```

When opening a file, set `workspaceId: state.workspace.activeId ?? undefined` on
both the Source and Document.

Full patch also lives in the conversation artifacts as `phase2.patch`.
