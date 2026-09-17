/**
 * Application state — single reducer, no UI logic.
 *
 * Domain state mutates only here. Components read via useApp() and
 * express intent with dispatched actions. Phase 2 workspace cases live
 * in reduceWorkspace; other cases are split across reducePart*.
 *
 * See docs/state-model.md for what is persisted vs. ephemeral.
 */

import {
  createContext,
  useContext,
  useMemo,
  useReducer,
  type Dispatch,
  type ReactNode,
} from "react";
import type { AppState } from "../types/domain";
import { createInitialState as mockInitialState } from "../mock/data";
import type { Action } from "./actions";
import { reducePart1a } from "./reducePart1a";
import { reducePart1b } from "./reducePart1b";
import { reducePart2 } from "./reducePart2";
import { reducePart3 } from "./reducePart3";
import { reducePart4 } from "./reducePart4";
import { reduceWorkspace } from "./reduceWorkspace";
import { reduceChat } from "./reduceChat";

export type { Action } from "./actions";

const DEMO = "ws-demo";

/** Ensure mock seed has workspace + tags (Phase 2) until data.ts is fully retagged. */
function createInitialState(): AppState {
  const s = mockInitialState() as AppState;
  if (!s.workspace?.byId) {
    s.workspace = {
      activeId: DEMO,
      byId: {
        [DEMO]: {
          id: DEMO,
          rootPath: null,
          primarySummaryId: "doc-summ",
          label: "Demo research workspace",
        },
      },
    };
  }
  if (!s.chat?.byId) {
    s.chat = { activeId: null, byId: {} };
  }
  const wsId = s.workspace.activeId ?? DEMO;
  for (const src of Object.values(s.sources)) {
    if (!src.workspaceId) src.workspaceId = wsId;
  }
  for (const doc of Object.values(s.documents)) {
    if (!doc.workspaceId) doc.workspaceId = wsId;
  }
  for (const ch of Object.values(s.changes)) {
    if (!ch.workspaceId) ch.workspaceId = wsId;
  }
  return s;
}

function reducer(state: AppState, action: Action): AppState {
  return (
    reduceWorkspace(state, action) ??
    reduceChat(state, action) ??
    reducePart1a(state, action) ??
    reducePart1b(state, action) ??
    reducePart2(state, action) ??
    reducePart3(state, action) ??
    reducePart4(state, action) ??
    state
  );
}

interface AppContextValue {
  state: AppState;
  dispatch: Dispatch<Action>;
}

const AppContext = createContext<AppContextValue | null>(null);

export function AppProvider({ children }: { children: ReactNode }) {
  const [state, dispatch] = useReducer(reducer, undefined, createInitialState);
  const value = useMemo(() => ({ state, dispatch }), [state]);
  return <AppContext.Provider value={value}>{children}</AppContext.Provider>;
}

export function useApp(): AppContextValue {
  const ctx = useContext(AppContext);
  if (!ctx) throw new Error("useApp must be used inside <AppProvider>");
  return ctx;
}

// Re-export selectors so existing imports from appState keep working.
export {
  activeBrowserSession,
  activeBrowserTab,
  activeReaderTab,
  activeReaderDocument,
  pendingChangesFor,
  runningActivity,
  DEMO_WORKSPACE_ID,
  activeWorkspace,
  primarySummaryDocument,
  workspaceIdFromRoot,
} from "./selectors";
