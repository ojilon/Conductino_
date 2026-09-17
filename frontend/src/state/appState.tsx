/**
 * Application state — single reducer, no UI logic.
 *
 * This is the ONLY place domain state mutates. Components read via
 * useApp() and express intent with dispatched actions. AI providers
 * and backend services feed state exclusively through actions too,
 * which keeps the mock → real replacement local to src/services.
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
import type {
  AIActivity,
  AppMode,
  AppState,
  BrowserUIState,
  Document,
  DocumentBlock,
  DocumentChange,
  Highlight,
  ID,
  ReaderUIState,
  TextSelection,
  WorkspaceSession,
} from "../types/domain";
import { createInitialState } from "../mock/data";
import { uid } from "../utils/helpers";

/* ------------------------------------------------------------------ */
/* Actions                                                             */
/* ------------------------------------------------------------------ */

export type Action =
  | { type: "mode.set"; mode: AppMode }
  // browser
  | { type: "browser.session.select"; id: ID }
  | { type: "browser.tab.add"; tabId: ID; sessionId: ID }
  | { type: "browser.tab.select"; tabId: ID }
  | { type: "browser.tab.close"; tabId: ID }
  | { type: "browser.navigate"; tabId: ID; url: string }
  | { type: "browser.nav"; tabId: ID; dir: -1 | 1 }
  | { type: "browser.ui"; patch: Partial<BrowserUIState> }
  // ai browse
  | { type: "browse.start"; query: string }
  | { type: "browse.phase"; phase: number }
  | { type: "browse.done"; sourceIds: ID[] }
  | { type: "browse.error"; message: string }
  | { type: "browse.reset" }
  // ai activities
  | { type: "activity.start"; activity: AIActivity }
  | { type: "activity.update"; id: ID; patch: Partial<AIActivity> }
  // sources
  | { type: "source.toggleSave"; id: ID }
  | { type: "source.markReader"; id: ID; documentId: ID }
  | { type: "source.preview"; id: ID | null }
  // reader
  | { type: "reader.doc.open"; tabId: ID; documentId: ID; label: string }
  | { type: "reader.tab.select"; tabId: ID }
  | { type: "reader.tab.close"; tabId: ID }
  | { type: "reader.ui"; patch: Partial<ReaderUIState> }
  // documents
  | { type: "doc.add"; document: AppState["documents"][ID] }
  | { type: "source.add"; source: AppState["sources"][ID] }
  | { type: "doc.block.text"; documentId: ID; blockId: ID; text: string }
  | { type: "doc.title"; documentId: ID; text: string }
  | { type: "doc.highlight.add"; documentId: ID; highlight: Highlight }
  // ai changes
  | { type: "change.propose"; change: DocumentChange; block?: DocumentBlock; afterBlockId?: ID }
  | { type: "change.decide"; id: ID; status: "accepted" | "rejected" }
  | { type: "change.revise"; id: ID; newContent: string; highlightFragment?: string }
  | { type: "summary.addSource"; documentId: ID; sourceId: ID }
  // workspace (Phase 2)
  | { type: "workspace.ensure"; session: WorkspaceSession }
  | { type: "workspace.setActive"; id: ID }
  | { type: "workspace.setPrimarySummary"; workspaceId: ID; summaryId: ID }
  | {
      type: "workspace.createSummary";
      workspaceId: ID;
      document: Document;
      source: AppState["sources"][ID];
      tabId: ID;
      label: string;
      setPrimary?: boolean;
    }
  // ephemeral
  | { type: "selection.set"; selection: TextSelection | null }
  | { type: "toast"; message: string | null }
  | { type: "settings.set"; open: boolean };

/* ------------------------------------------------------------------ */
/* Reducer                                                             */
/* ------------------------------------------------------------------ */

function patchDoc(state: AppState, documentId: ID, fn: (doc: AppState["documents"][ID]) => AppState["documents"][ID]): AppState {
  const doc = state.documents[documentId];
  if (!doc) return state;
  return { ...state, documents: { ...state.documents, [documentId]: fn(doc) } };
}

function reducer(state: AppState, action: Action): AppState {
  switch (action.type) {
    case "mode.set":
      return { ...state, mode: action.mode };

    /* ---------------- browser ---------------- */

    case "browser.session.select":
      return { ...state, browser: { ...state.browser, activeSessionId: action.id };

    default:
      return state;
  }
}

export function AppProvider({ children }: { children: ReactNode }) {
  const [state, dispatch] = useReducer(reducer, undefined, createInitialState);
  const value = useMemo(() => ({ state, dispatch }), [state]);
  return <AppContext.Provider value={value}>{children}</AppContext.Provider>;
}
