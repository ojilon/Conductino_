import type { AppState } from "../types/domain";
import type { Action } from "./actions";

export function reducePart1a(state: AppState, action: Action): AppState | null {
  switch (action.type) {
    case "mode.set":
      return { ...state, mode: action.mode };

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

    default: return null;
  }
}
