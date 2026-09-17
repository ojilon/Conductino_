/**
 * App state action union (Phase 2 includes workspace.*).
 */
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
  WorkspaceSession,
  ChatMessage,
  ChatThread,
} from "../types/domain";

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
  // browse (AI search pipeline)
  | { type: "browse.start"; query: string }
  | { type: "browse.phase"; phase: number }
  | { type: "browse.done"; sourceIds: ID[] }
  | { type: "browse.error"; message: string }
  | { type: "browse.reset" }
  // AI activity log
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
  // documents / changes
  | { type: "doc.add"; document: AppState["documents"][ID] }
  | { type: "source.add"; source: AppState["sources"][ID] }
  | { type: "doc.block.text"; documentId: ID; blockId: ID; text: string }
  | { type: "doc.title"; documentId: ID; text: string }
  | { type: "doc.highlight.add"; documentId: ID; highlight: Highlight }
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
      document: AppState["documents"][ID];
      source: AppState["sources"][ID];
      tabId: ID;
      label: string;
      setPrimary?: boolean;
    }
  // chat (Phase 4)
  | { type: "chat.ensure"; thread: ChatThread }
  | { type: "chat.setActive"; id: ID }
  | { type: "chat.append"; threadId: ID; message: ChatMessage }
  | { type: "chat.clear"; threadId: ID }
  // selection / toast / settings
  | { type: "selection.set"; selection: TextSelection | null }
  | { type: "toast"; message: string | null }
  | { type: "settings.set"; open: boolean };
