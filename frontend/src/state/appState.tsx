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
import { createInitialState } from "../mock/data";
import type { Action } from "./actions";
import { reducePart1a } from "./reducePart1a";
import { reducePart1b } from "./reducePart1b";
import { reducePart2 } from "./reducePart2";
import { reducePart3 } from "./reducePart3";
import { reducePart4 } from "./reducePart4";
import { reduceWorkspace } from "./reduceWorkspace";

export type { Action } from "./actions";

function reducer(state: AppState, action: Action): AppState {
  return (
    reduceWorkspace(state, action) ??
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
