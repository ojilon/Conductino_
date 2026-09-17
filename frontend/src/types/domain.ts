/**
 * Lumen — domain model.
 *
 * Pure types only (no React, no services). Everything the UI renders or
 * persists should be expressible with the types in this file.
 *
 * Conventions:
 *  - IDs (string) reference entities; never rely on array positions.
 *  - "domain state" (sessions, documents, sources, changes, activities)
 *    is what a future SQLite layer would persist.
 *  - "ui state" (panel open/width, selection, toasts) is ephemeral.
 */

export type ID = string;

export type AppMode = "browser" | "reader";

/* ------------------------------------------------------------------ */
/* Browser                                                             */
/* ------------------------------------------------------------------ */

/** One tab's navigation stack (what a real webview would own). */
export interface NavigationState {
  currentUrl: string;
  history: string[];
  /** Pointer into `history`; currentUrl === history[index]. */
  index: number;
}

export interface BrowserTab {
  id: ID;
  title: string;
  nav: NavigationState;
}

export interface BrowserSession {
  id: ID;
  title: string;
  tabIds: ID[];
  activeTabId: ID | null;
}

/** Ephemeral UI state for Browser mode (never persisted). */
export interface BrowserUIState {
  sidebarOpen: boolean;
  aiPanelOpen: boolean;
  aiPanelWidth: number;
}

export interface BrowserState {
  sessions: BrowserSession[];
  activeSessionId: ID;
  tabs: Record<ID, BrowserTab>;
  ui: BrowserUIState;
}

/* ------------------------------------------------------------------ */
/* Sources                                                             */
/* ------------------------------------------------------------------ */

export type SourceKind = "web" | "pdf" | "docx" | "html" | "text" | "summary";
export type Relevance = "highest" | "high" | "good";

/**
 * A research source: a webpage, local file or generated summary.
 * Sources are the currency of the workflow: AI ranks them, the user
 * opens/saves them, and summaries cite them.
 */
export interface Source {
  id: ID;
  /** Owning workspace; undefined = legacy/unscoped (mock until tagged). */
  workspaceId?: ID;
  kind: SourceKind;
  title: string;
  /** Host (web) or file path (local). */
  origin: string;
  url?: string;
  typeLabel: string;
  abstract: string;
  rank?: number;
  relevance?: Relevance;
  saved: boolean;
  inReader: boolean;
  /** Set once the source has been opened as a reader document. */
  documentId?: ID;
}


/* ------------------------------------------------------------------ */
/* Workspace (folder-scoped session)                                  */
/* ------------------------------------------------------------------ */

/**
 * One open research folder. Maps sources/summaries so "include in summary"
 * never picks an arbitrary first summary (tasks.md §2 / Phase 2).
 * rootPath is absolute on desktop; null for the seeded mock workspace.
 */
export interface WorkspaceSession {
  id: ID;
  /** Absolute library root when known (desktop). */
  rootPath: string | null;
  /** Summary document that receives AI merges for this workspace. */
  primarySummaryId: ID | null;
  label?: string;
}

/* ------------------------------------------------------------------ */
/* AI                                                                  */
/* ------------------------------------------------------------------ */

export type AIOperation =
  | "AI_SEARCH"
  | "AI_SUMMARIZE"
  | "AI_EXPLAIN"
  | "AI_EXPAND"
  | "AI_MERGE"
  | "AI_REWRITE"
  | "AI_VERIFY"
  | "AI_CHAT";

export type AIStatus = "running" | "completed" | "error";

/** Streaming progress steps shown by the AI Browse panel. */
export const BROWSE_PHASES = [
  "Searching",
  "Finding sources",
  "Comparing sources",
  "Ranking results",
  "Preparing useful sources",
] as const;
export type BrowsePhaseIndex = 0 | 1 | 2 | 3 | 4;

/**
 * One unit of AI work. Activities answer: what is the AI doing right
 * now, on which source/document, and what did it propose?
 */
export interface AIActivity {
  id: ID;
  operation: AIOperation;
  status: AIStatus;
  workspaceId?: ID;
  /** Human-readable progress message ("Comparing sources…"). */
  message: string;
  documentId?: ID;
  sourceId?: ID;
  changeIds: ID[];
  selection?: string;
  startedAt: number;
  finishedAt?: number;
  error?: string;
}

/** One prior turn for multi-turn chat (Phase 4). */
export interface ChatTurn {
  role: "user" | "assistant" | "system";
  content: string;
}

export interface AIRequest {
  operation: AIOperation;
  query?: string;
  documentId?: ID;
  sourceId?: ID;
  selection?: { blockId: ID; text: string };
  changeId?: ID;
  /** Phase 2/3: owning workspace when known. */
  workspaceId?: ID;
  /** User-authored instruction (selection toolbar / chat composer). */
  customPrompt?: string;
  /** When true (default for explain family), attach document context pack. */
  includeDocumentContext?: boolean;
  /**
   * Pre-assembled context (selection window + outline). Built on the
   * frontend while blocks live in AppState; backend appends to the prompt.
   */
  contextPack?: string;
  /** Phase 4: prior turns (current user message is query / customPrompt). */
  messageHistory?: ChatTurn[];
  /** "oneshot" | "chat" — hint for prompt assembly. */
  mode?: "oneshot" | "chat";
}

/** What a provider can produce back to the app. */
export interface AIResult {
  explanation?: string;
  relatedSources?: { title: string; meta: string }[];
  insertion?: { text: string; citation: string };
  revision?: string;
}

export interface AIHandlers {
  /** index: 0..4 for browse-style progress (see BROWSE_PHASES). */
  onPhase(index: number, label: string): void;
  /** Browse-only: the ranked set of source ids the provider found. */
  onBrowseSources(sourceIds: ID[]): void;
  onDone(result: AIResult): void;
  onError(message: string): void;
}

/**
 * Provider boundary. The React app talks only to this interface.
 * Swap MockAIProvider for an OpenAI/Anthropic/local model provider by
 * implementing this type and calling setAIProvider() — see
 * docs/ai-integration.md.
 */
export interface AIProvider {
  readonly name: string;
  /** False while the built-in mock is in use. */
  readonly configured: boolean;
  /** Runs the request, streaming phases; returns a cancel function. */
  run(request: AIRequest, handlers: AIHandlers): () => void;
}

/* ------------------------------------------------------------------ */
/* Chat (Phase 4 — in-memory; SQLite in Phase 6)                       */
/* ------------------------------------------------------------------ */

export type ChatRole = "user" | "assistant" | "system";

export interface ChatMessage {
  id: ID;
  role: ChatRole;
  content: string;
  createdAt: number;
  /** Optional link to the document that was in focus when sent. */
  documentId?: ID;
}

/**
 * One conversation thread, scoped to a workspace (and optionally a document).
 * Threads live in AppState for Phase 4; Phase 6 persists to SQLite.
 */
export interface ChatThread {
  id: ID;
  workspaceId: ID;
  /** When set, the thread is tied to a specific open document. */
  documentId?: ID;
  title?: string;
  messages: ChatMessage[];
  createdAt: number;
  updatedAt: number;
}

export type AiPanelTab = "chat" | "reading";

/* ------------------------------------------------------------------ */
/* Documents                                                           */
/* ------------------------------------------------------------------ */

export type BlockType = "heading" | "paragraph" | "list" | "page";

/** Inline run of text with optional styling / highlight link. */
export interface Segment {
  text: string;
  em?: boolean;
  strong?: boolean;
  highlightId?: ID;
}

export interface DocumentBlock {
  id: ID;
  type: BlockType;
  /** Heading level (1..3). */
  level?: 1 | 2 | 3;
  segments: Segment[];
  /** For type === "list". */
  listItems?: Segment[][];
  /** Set when the block was created/modified by an AI change. */
  changeId?: ID;
}

export interface DocumentMetadata {
  title: string;
  author?: string;
  venue?: string;
  year?: string;
  format: SourceKind;
  pageCount?: number;
  path?: string;
  /**
   * Absolute workspace root the file was opened from (tasks.md 1.1 option b).
   * Compared against the current library root before any resolve/reveal/
   * locate — a mismatch means "file belongs to another folder", never a
   * silent re-resolve against the new root. Undefined = untagged (mock data,
   * summaries, pre-fix documents) — no stale check applies.
   */
  rootPath?: string;
}

/**
 * A highlight/annotation anchored to a block. (A future renderer
 * upgrade should carry precise character offsets; the UI already
 * distinguishes block-level from range-level annotations.)
 */
export interface Highlight {
  id: ID;
  blockId: ID;
  text: string;
  note?: string;
  createdAt: number;
}

export interface Document {
  id: ID;
  /** Owning workspace; undefined = legacy/unscoped. */
  kind: "source" | "summary";
  workspaceId?: ID;
  sourceId: ID;
  metadata: DocumentMetadata;
  blocks: DocumentBlock[];
  highlights: Highlight[];
  /** Summary documents only: sources that contributed to this summary. */
  sourceIds?: ID[];
  /** Mock pagination for source documents. */
  currentPage?: number;
}

/* ------------------------------------------------------------------ */
/* AI document changes (proposal system)                               */
/* ------------------------------------------------------------------ */

export type ChangeType = "insert" | "modify" | "delete";
export type ChangeStatus = "pending" | "accepted" | "rejected";

/**
 * A proposed AI change. For "insert" the block already exists (created
 * at proposal time, rendered as pending). For "modify"/"delete" the
 * target block keeps its original content; the pending view renders
 * newContent / strike-through, accept commits, reject reverts.
 * A real diff engine can later refine `location` — see
 * docs/architecture.md §Diff system.
 */
export interface DocumentChange {
  id: ID;
  documentId: ID;
  /** Owning workspace (same as target document when set). */
  workspaceId?: ID;
  type: ChangeType;
  blockId: ID;
  oldContent: string;
  newContent: string;
  /** Exact substring rendered with the "modified" treatment. */
  highlightFragment?: string;
  sourceId: ID;
  activityId: ID;
  status: ChangeStatus;
  createdAt: number;
}

/* ------------------------------------------------------------------ */
/* Reader                                                              */
/* ------------------------------------------------------------------ */

export interface ReaderTab {
  id: ID;
  documentId: ID;
  label: string;
}

export type RailView = "documents" | "bookmarks" | "notes" | "library" | "settings";

/** Ephemeral UI state for Reader mode (never persisted). */
export interface ReaderUIState {
  railOpen: boolean;
  sidebarOpen: boolean;
  railView: RailView;
  aiPanelOpen: boolean;
  aiPanelWidth: number;
  /** Phase 4: which face of the AI panel is visible. */
  aiPanelTab: AiPanelTab;
  /** AI Reading companion payload for the current document. */
  companion: {
    documentId: ID;
    passage: string;
    explanation: string;
    sourceLabel: string;
    /** Related sources returned by the provider with the explanation. */
    relatedSources?: { title: string; meta: string }[];
  } | null;
}

export interface ReaderState {
  session: { id: ID; tabIds: ID[]; activeTabId: ID | null };
  tabs: Record<ID, ReaderTab>;
  ui: ReaderUIState;
}

/* ------------------------------------------------------------------ */
/* App level                                                           */
/* ------------------------------------------------------------------ */

export interface TextSelection {
  documentId: ID;
  blockId: ID;
  text: string;
  /** Viewport coordinates of the selection (for the floating toolbar). */
  x: number;
  y: number;
}

export type BrowseStatus = "idle" | "running" | "done" | "error";

export interface BrowseState {
  status: BrowseStatus;
  query: string;
  /** Index into BROWSE_PHASES while running. */
  phase: number;
  sourceIds: ID[];
  error?: string;
}

export interface ToastMsg {
  id: ID;
  message: string;
}

/** Root application state. See docs/state-model.md. */
export interface AppState {
  mode: AppMode;
  browser: BrowserState;
  reader: ReaderState;
  /**
   * Folder-scoped sessions. activeId is the library currently open;
   * primarySummaryId on that session is the merge target.
   */
  workspace: {
    activeId: ID | null;
    byId: Record<ID, WorkspaceSession>;
  };
  /**
   * Phase 4 chat threads (in-memory). activeId is the thread currently
   * shown in the Chat face; byId is keyed by thread id.
   */
  chat: {
    activeId: ID | null;
    byId: Record<ID, ChatThread>;
  };
  sources: Record<ID, Source>;
  documents: Record<ID, Document>;
  changes: Record<ID, DocumentChange>;
  /** AI activity log, newest first. */
  aiActivities: AIActivity[];
  browse: BrowseState;
  selection: TextSelection | null;
  toast: ToastMsg | null;
  settingsOpen: boolean;
  /** Source preview modal. */
  previewSourceId: ID | null;
}

/* ------------------------------------------------------------------ */
/* Backend service payloads (shape of Go data crossing the boundary)   */
/* ------------------------------------------------------------------ */

export interface FileTreeNode {
  id: ID;
  label: string;
  kind: "folder" | "file";
  ext?: string;
  /**
   * Disk location, relative to the workspace root the Go backend holds
   * privately (set from the folder dialog). Opaque token: display `label`,
   * send `path` back for open/extract/reveal, never build paths in TS.
   * Mirrors `Path` on the Go FileTreeNode (backend/models/models.go).
   */
  path?: string;
  children?: FileTreeNode[];
  /** Set when the file maps to an openable reader document. */
  documentId?: ID;
}
