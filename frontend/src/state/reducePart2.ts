/**
 * Source + reader reducer cases.
 */
import type { AppState } from "../types/domain";
import type { Action } from "./actions";

export function reducePart2(state: AppState, action: Action): AppState | null {
  switch (action.type) {
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

    default: return null;
  }
}
