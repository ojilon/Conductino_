/**
 * Workspace session reducer cases (Phase 2).
 */
import type { AppState, WorkspaceSession, ID } from "../types/domain";
import type { Action } from "./actions";

export function reduceWorkspace(state: AppState, action: Action): AppState | null {
  switch (action.type) {
    case "workspace.ensure": {
      const s = action.session;
      const existing = state.workspace.byId[s.id];
      const merged: WorkspaceSession = existing
        ? {
            ...existing,
            rootPath: s.rootPath ?? existing.rootPath,
            primarySummaryId: s.primarySummaryId ?? existing.primarySummaryId,
            label: s.label ?? existing.label,
          }
        : s;
      return {
        ...state,
        workspace: {
          activeId: state.workspace.activeId ?? s.id,
          byId: { ...state.workspace.byId, [s.id]: merged },
        },
      };
    }

    case "workspace.setActive": {
      if (!state.workspace.byId[action.id]) return state;
      return { ...state, workspace: { ...state.workspace, activeId: action.id } };
    }

    case "workspace.setPrimarySummary": {
      const ws = state.workspace.byId[action.workspaceId];
      if (!ws) return state;
      return {
        ...state,
        workspace: {
          ...state.workspace,
          byId: {
            ...state.workspace.byId,
            [action.workspaceId]: { ...ws, primarySummaryId: action.summaryId },
          },
        },
      };
    }

    case "workspace.createSummary": {
      const { workspaceId, document, source, tabId, label, setPrimary } = action;
      if (state.documents[document.id]) return state;
      const ws = state.workspace.byId[workspaceId];
      const nextWs: WorkspaceSession = ws
        ? {
            ...ws,
            primarySummaryId:
              setPrimary || !ws.primarySummaryId ? document.id : ws.primarySummaryId,
          }
        : {
            id: workspaceId,
            rootPath: null,
            primarySummaryId: document.id,
            label: "Workspace",
          };
      const existingTab = state.reader.session.tabIds.find(
        (tid) => state.reader.tabs[tid]?.documentId === document.id,
      );
      let reader = state.reader;
      if (!existingTab) {
        reader = {
          ...state.reader,
          tabs: { ...state.reader.tabs, [tabId]: { id: tabId, documentId: document.id, label } },
          session: {
            ...state.reader.session,
            tabIds: [...state.reader.session.tabIds, tabId],
            activeTabId: tabId,
          },
        };
      } else {
        reader = {
          ...state.reader,
          session: { ...state.reader.session, activeTabId: existingTab },
        };
      }
      return {
        ...state,
        workspace: {
          activeId: state.workspace.activeId ?? workspaceId,
          byId: { ...state.workspace.byId, [workspaceId]: nextWs },
        },
        sources: { ...state.sources, [source.id]: source },
        documents: { ...state.documents, [document.id]: document },
        reader,
      };
    }

    default:
      return null;
  }
}
