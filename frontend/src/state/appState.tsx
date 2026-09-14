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
  DocumentBlock,
  DocumentChange,
  Highlight,
  ID,
  ReaderUIState,
  TextSelection,
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
      return { ...state, browser: { ...state.browser, activeSessionId: action.id } };

    case "browser.tab.add": {
      const tabId = action.tabId;
      const newTab = { id: tabId, title: "New tab", nav: { currentUrl: "lumen://newtab", history: ["lumen://newtab"], index: 0 } };
      return {
        ...state,
        browser: {
          ...state.browser,
          tabs: { ...state.browser.tabs, [tabId]: newTab },
          sessions: state.browser.sessions.map((s) =>
            s.id === action.sessionId ? { ...s, tabIds: [...s.tabIds, tabId], activeTabId: tabId } : s,
          ),
        },
      };
    }

    case "browser.tab.select":
      return {
        ...state,
        browser: {
          ...state.browser,
          sessions: state.browser.sessions.map((s) =>
            s.tabIds.includes(action.tabId) ? { ...s, activeTabId: action.tabId } : s,
          ),
        },
      };

    case "browser.tab.close": {
      return {
        ...state,
        browser: {
          ...state.browser,
          sessions: state.browser.sessions.map((s) => {
            if (!s.tabIds.includes(action.tabId)) return s;
            const idx = s.tabIds.indexOf(action.tabId);
            const tabIds = s.tabIds.filter((t) => t !== action.tabId);
            let activeTabId = s.activeTabId;
            if (activeTabId === action.tabId) {
              activeTabId = tabIds[Math.min(idx, tabIds.length - 1)] ?? null;
            }
            return { ...s, tabIds, activeTabId };
          }),
        },
      };
    }

    case "browser.navigate": {
      const tab = state.browser.tabs[action.tabId];
      if (!tab) return state;
      let nav;
      if (action.url === tab.nav.currentUrl) {
        nav = { ...tab.nav, history: tab.nav.history.map((u, i) => (i === tab.nav.index ? action.url : u)) };
      } else {
        const history = [...tab.nav.history.slice(0, tab.nav.index + 1), action.url];
        nav = { currentUrl: action.url, history, index: history.length - 1 };
      }
      return {
        ...state,
        browser: { ...state.browser, tabs: { ...state.browser.tabs, [action.tabId]: { ...tab, nav } } },
      };
    }

    case "browser.nav": {
      const tab = state.browser.tabs[action.tabId];
      if (!tab) return state;
      const index = tab.nav.index + action.dir;
      if (index < 0 || index >= tab.nav.history.length) return state;
      return {
        ...state,
        browser: {
          ...state.browser,
          tabs: {
            ...state.browser.tabs,
            [action.tabId]: { ...tab, nav: { ...tab.nav, index, currentUrl: tab.nav.history[index] } },
          },
        },
      };
    }

    case "browser.ui":
      return { ...state, browser: { ...state.browser, ui: { ...state.browser.ui, ...action.patch } } };

    /* ---------------- ai browse ---------------- */

    case "browse.start":
      return { ...state, browse: { status: "running", query: action.query, phase: 0, sourceIds: [] } };

    case "browse.phase":
      if (state.browse.status !== "running") return state;
      return { ...state, browse: { ...state.browse, phase: action.phase } };

    case "browse.done":
      return { ...state, browse: { ...state.browse, status: "done", phase: 5, sourceIds: action.sourceIds } };

    case "browse.error":
      return { ...state, browse: { ...state.browse, status: "error", error: action.message } };

    case "browse.reset":
      return { ...state, browse: { status: "idle", query: state.browse.query, phase: 0, sourceIds: [] } };

    /* ---------------- ai activities ---------------- */

    case "activity.start":
      return { ...state, aiActivities: [action.activity, ...state.aiActivities] };

    case "activity.update":
      return {
        ...state,
        aiActivities: state.aiActivities.map((a) => (a.id === action.id ? { ...a, ...action.patch } : a)),
      };

    /* ---------------- sources ---------------- */

    case "source.toggleSave": {
      const src = state.sources[action.id];
      if (!src) return state;
      return { ...state, sources: { ...state.sources, [action.id]: { ...src, saved: !src.saved } } };
    }

    case "source.markReader": {
      const src = state.sources[action.id];
      if (!src) return state;
      return {
        ...state,
        sources: { ...state.sources, [action.id]: { ...src, inReader: true, documentId: action.documentId } },
      };
    }

    case "source.preview":
      return { ...state, previewSourceId: action.id };

    /* ---------------- reader ---------------- */

    case "reader.doc.open": {
      const existing = state.reader.session.tabIds.find(
        (tid) => state.reader.tabs[tid]?.documentId === action.documentId,
      );
      if (existing) {
        return {
          ...state,
          reader: { ...state.reader, session: { ...state.reader.session, activeTabId: existing } },
        };
      }
      const tabId = action.tabId;
      return {
        ...state,
        reader: {
          ...state.reader,
          tabs: { ...state.reader.tabs, [tabId]: { id: tabId, documentId: action.documentId, label: action.label } },
          session: {
            ...state.reader.session,
            tabIds: [...state.reader.session.tabIds, tabId],
            activeTabId: tabId,
          },
        },
      };
    }

    case "reader.tab.select":
      if (!state.reader.session.tabIds.includes(action.tabId)) return state;
      return {
        ...state,
        reader: { ...state.reader, session: { ...state.reader.session, activeTabId: action.tabId } },
      };

    case "reader.tab.close": {
      const { tabIds } = state.reader.session;
      if (!tabIds.includes(action.tabId)) return state;
      const idx = tabIds.indexOf(action.tabId);
      const next = tabIds.filter((t) => t !== action.tabId);
      const activeTabId =
        state.reader.session.activeTabId === action.tabId
          ? (next[Math.min(idx, next.length - 1)] ?? null)
          : state.reader.session.activeTabId;
      return {
        ...state,
        reader: {
          ...state.reader,
          tabs: Object.fromEntries(Object.entries(state.reader.tabs).filter(([k]) => k !== action.tabId)),
          session: { ...state.reader.session, tabIds: next, activeTabId },
        },
      };
    }

    case "reader.ui":
      return { ...state, reader: { ...state.reader, ui: { ...state.reader.ui, ...action.patch } } };

    /* ---------------- documents ---------------- */

    case "doc.add":
      if (state.documents[action.document.id]) return state;
      return { ...state, documents: { ...state.documents, [action.document.id]: action.document } };

    case "source.add":
      if (state.sources[action.source.id]) return state;
      return { ...state, sources: { ...state.sources, [action.source.id]: action.source } };

    case "doc.block.text":
      return patchDoc(state, action.documentId, (doc) => ({
        ...doc,
        blocks: doc.blocks.map((b) =>
          b.id === action.blockId ? { ...b, segments: [{ text: action.text }] } : b,
        ),
      }));

    case "doc.title":
      return patchDoc(state, action.documentId, (doc) => ({
        ...doc,
        metadata: { ...doc.metadata, title: action.text },
      }));

    case "doc.highlight.add":
      return patchDoc(state, action.documentId, (doc) => ({
        ...doc,
        highlights: [...doc.highlights, action.highlight],
      }));

    /* ---------------- ai changes ---------------- */

    case "change.propose": {
      const withChange: AppState = {
        ...state,
        changes: { ...state.changes, [action.change.id]: action.change },
      };
      if (!action.block) return withChange;
      return patchDoc(withChange, action.change.documentId, (doc) => {
        const block = action.block as DocumentBlock;
        const at = action.afterBlockId ? doc.blocks.findIndex((b) => b.id === action.afterBlockId) : -1;
        const blocks =
          at >= 0
            ? [...doc.blocks.slice(0, at + 1), block, ...doc.blocks.slice(at + 1)]
            : [...doc.blocks, block];
        return { ...doc, blocks };
      });
    }

    case "change.decide": {
      const change = state.changes[action.id];
      if (!change) return state;
      const next = patchDoc(state, change.documentId, (doc) => {
        if (action.status === "accepted" && change.type === "modify") {
          return {
            ...doc,
            blocks: doc.blocks.map((b) =>
              b.id === change.blockId ? { ...b, segments: [{ text: change.newContent }] } : b,
            ),
          };
        }
        if (
          (action.status === "accepted" && change.type === "delete") ||
          (action.status === "rejected" && change.type === "insert")
        ) {
          return { ...doc, blocks: doc.blocks.filter((b) => b.id !== change.blockId) };
        }
        return doc;
      });
      return {
        ...next,
        changes: { ...next.changes, [action.id]: { ...change, status: action.status } },
      };
    }

    case "change.revise": {
      const change = state.changes[action.id];
      if (!change) return state;
      return {
        ...state,
        changes: {
          ...state.changes,
          [action.id]: {
            ...change,
            newContent: action.newContent,
            highlightFragment: action.highlightFragment ?? change.highlightFragment,
          },
        },
      };
    }

    case "summary.addSource":
      return patchDoc(state, action.documentId, (doc) => {
        if (!doc.sourceIds || doc.sourceIds.includes(action.sourceId)) return doc;
        return { ...doc, sourceIds: [...doc.sourceIds, action.sourceId] };
      });

    /* ---------------- ephemeral ---------------- */

    case "selection.set":
      return { ...state, selection: action.selection };

    case "toast":
      return { ...state, toast: action.message ? { id: uid("toast"), message: action.message } : null };

    case "settings.set":
      return { ...state, settingsOpen: action.open };
  }
}

/* ------------------------------------------------------------------ */
/* Context                                                             */
/* ------------------------------------------------------------------ */

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

/* ------------------------------------------------------------------ */
/* Selectors                                                           */
/* ------------------------------------------------------------------ */

export function activeBrowserSession(state: AppState) {
  return state.browser.sessions.find((s) => s.id === state.browser.activeSessionId) ?? state.browser.sessions[0];
}

export function activeBrowserTab(state: AppState) {
  const session = activeBrowserSession(state);
  return session.activeTabId ? state.browser.tabs[session.activeTabId] : undefined;
}

export function activeReaderTab(state: AppState) {
  const id = state.reader.session.activeTabId;
  return id ? state.reader.tabs[id] : undefined;
}

export function activeReaderDocument(state: AppState) {
  const tab = activeReaderTab(state);
  return tab ? state.documents[tab.documentId] : undefined;
}

export function pendingChangesFor(state: AppState, documentId: ID): DocumentChange[] {
  return Object.values(state.changes)
    .filter((c) => c.documentId === documentId && c.status === "pending")
    .sort((a, b) => a.createdAt - b.createdAt);
}

export function runningActivity(state: AppState): AIActivity | undefined {
  return state.aiActivities.find((a) => a.status === "running");
}
