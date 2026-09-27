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

/**
 * Empty initial state: one blank browser tab (browser chrome needs a valid
 * session), reader fully empty, no documents/sources/changes/AI history.
 * Sections show their empty states until the user picks a library folder
 * and opens a real file (desktop) or sends a source to the Reader.
 */
function createInitialState(): AppState {
  return {
    mode: "reader",
    chat: { activeId: null, byId: {} },
    workspace: {
      activeId: DEMO,
      byId: {
        [DEMO]: {
          id: DEMO,
          rootPath: null,
          primarySummaryId: "doc-summ",
          label: "Demo research workspace",
        },
      },
    },
    browser: {
      sessions: [{ id: "sess-1", title: "Session 1", tabIds: ["t-new"], activeTabId: "t-new" }],
      activeSessionId: "sess-1",
      tabs: {
        "t-new": {
          id: "t-new",
          title: "New tab",
          nav: { currentUrl: "lumen://newtab", history: ["lumen://newtab"], index: 0 },
        },
      },
      ui: { sidebarOpen: true, aiPanelOpen: true, aiPanelWidth: 360 },
    },
    reader: {
      session: { id: "rs-1", tabIds: [], activeTabId: null },
      tabs: {},
      ui: {
        railOpen: true,
        sidebarOpen: true,
        railView: "library",
        aiPanelOpen: true,
        aiPanelWidth: 330,
        aiPanelTab: "chat",
        companion: null,
        summaryPick: null,
      },
    },
    sources: {},
    documents: {},
    changes: {},
    aiActivities: [],
    browse: {
      status: "idle",
      query: "",
      phase: 0,
      sourceIds: [],
    },
    selection: null,
    toast: null,
    settingsOpen: false,
    previewSourceId: null,
  };
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
  summariesNearDocument,
  workspaceIdFromRoot,
} from "./selectors";
