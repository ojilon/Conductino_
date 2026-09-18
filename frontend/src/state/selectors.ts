/**
 * App state selectors (pure).
 */
import type { AppState, Document, DocumentChange, AIActivity, ID, WorkspaceSession } from "../types/domain";

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

/** Seeded mock workspace id (createInitialState). */
export const DEMO_WORKSPACE_ID = "ws-demo";

export function activeWorkspace(state: AppState): WorkspaceSession | undefined {
  const id = state.workspace.activeId;
  return id ? state.workspace.byId[id] : undefined;
}

/**
 * Summary document that should receive AI merges for the active workspace.
 * Prefer explicit primarySummaryId; never fall back to "first summary in map"
 * across workspaces (that was the first-summary-wins bug).
 */
export function primarySummaryDocument(state: AppState): Document | undefined {
  const ws = activeWorkspace(state);
  if (ws?.primarySummaryId) {
    const doc = state.documents[ws.primarySummaryId];
    if (doc?.kind === "summary") return doc;
  }
  // Same-workspace fallback only: one summary tagged with this workspaceId
  if (ws) {
    const scoped = Object.values(state.documents).filter(
      (d) => d.kind === "summary" && d.workspaceId === ws.id,
    );
    if (scoped.length === 1) return scoped[0];
  }
  return undefined;
}

/** Stable id for a library root path (desktop folder pick). */
export function workspaceIdFromRoot(rootPath: string): ID {
  // Simple non-crypto hash → short id; collision risk acceptable for local single-user.
  let h = 0;
  for (let i = 0; i < rootPath.length; i++) h = (Math.imul(31, h) + rootPath.charCodeAt(i)) | 0;
  return `ws-${(h >>> 0).toString(16)}`;
}

/** Folder directory of a document path (POSIX or Windows). */
function parentDir(path: string | undefined | null): string | null {
  if (!path) return null;
  const norm = path.replace(/\\/g, "/");
  const i = norm.lastIndexOf("/");
  if (i <= 0) return null;
  return norm.slice(0, i);
}

/**
 * Summaries that can receive "include in summary" for a source document.
 * Prefer summaries in the same folder as the source; fall back to all
 * summaries in the same workspace. Used when primary is unset or ambiguous.
 */
export function summariesNearDocument(state: AppState, sourceDocId: ID): Document[] {
  const source = state.documents[sourceDocId];
  if (!source) return [];
  const wsId = source.workspaceId ?? activeWorkspace(state)?.id;
  const all = Object.values(state.documents).filter((d) => d.kind === "summary");
  const scoped = wsId ? all.filter((d) => d.workspaceId === wsId) : all;
  const pool = scoped.length ? scoped : all;
  if (pool.length <= 1) return pool;

  const srcDir = parentDir(source.metadata?.path);
  if (!srcDir) return pool;

  const sameFolder = pool.filter((d) => {
    const dDir = parentDir(d.metadata?.path);
    return dDir !== null && dDir === srcDir;
  });
  return sameFolder.length ? sameFolder : pool;
}
