/**
 * Lumen mock data — minimal seed (Phase 2 workspace + Phase 4 chat included).
 */
import type {
  AppState,
  BrowserSession,
  BrowserTab,
  Document,
  DocumentChange,
  AIActivity,
  FileTreeNode,
  Source,
} from "../types/domain";

const sources: Record<string, Source> = {
  "src-a": {
    id: "src-a",
    workspaceId: "ws-demo",
    kind: "pdf",
    title: "Chemiosmosis and ATP synthesis",
    origin: "research_papers/mitchell_1961.pdf",
    typeLabel: "PDF paper",
    abstract: "Mitchell chemiosmotic coupling account.",
    saved: true,
    inReader: true,
    documentId: "doc-a",
  },
  "src-summ": {
    id: "src-summ",
    workspaceId: "ws-demo",
    kind: "summary",
    title: "Research Summary: Chemiosmosis and ATP synthesis",
    origin: "workspace/research_summary",
    typeLabel: "Summary",
    abstract: "Working summary.",
    saved: true,
    inReader: true,
    documentId: "doc-summ",
  },
};

const docA: Document = {
  id: "doc-a",
  workspaceId: "ws-demo",
  kind: "source",
  sourceId: "src-a",
  currentPage: 1,
  metadata: { title: "Chemiosmosis and ATP synthesis", format: "pdf", pageCount: 12 },
  blocks: [{ id: "a1", type: "paragraph", segments: [{ text: "The proton gradient drives ATP synthesis." }] }],
  highlights: [],
};

const docSumm: Document = {
  id: "doc-summ",
  workspaceId: "ws-demo",
  kind: "summary",
  sourceId: "src-summ",
  sourceIds: ["src-a"],
  metadata: { title: "Research Summary", format: "summary" },
  blocks: [{ id: "s1", type: "heading", level: 1, segments: [{ text: "Overview" }] }],
  highlights: [],
};

const documents: Record<string, Document> = { "doc-a": docA, "doc-summ": docSumm };
const changes: Record<string, DocumentChange> = {};
const aiActivities: AIActivity[] = [];

const browserTabs: Record<string, BrowserTab> = {
  "bt-1": { id: "bt-1", title: "New tab", nav: { currentUrl: "lumen://newtab", history: ["lumen://newtab"], index: 0 } },
};
const browserSessions: BrowserSession[] = [
  { id: "sess-1", title: "Research", tabIds: ["bt-1"], activeTabId: "bt-1" },
];

export const fileTreeMock: FileTreeNode = {
  id: "root",
  label: "research_workspace",
  kind: "folder",
  children: [
    { id: "f1", label: "mitchell_1961.pdf", kind: "file", ext: "pdf", path: "research_papers/mitchell_1961.pdf", documentId: "doc-a" },
  ],
};

export function makeDocumentFromSource(sourceId: string, source: Source, docId: string): Document {
  return {
    id: docId,
    workspaceId: source.workspaceId,
    kind: "source",
    sourceId,
    currentPage: 1,
    metadata: { title: source.title, format: source.kind === "summary" ? "summary" : source.kind },
    blocks: [{ id: "b1", type: "paragraph", segments: [{ text: source.abstract }] }],
    highlights: [],
  };
}

export function createInitialState(): AppState {
  return {
    mode: "reader",
    chat: { activeId: null, byId: {} },
    workspace: {
      activeId: "ws-demo",
      byId: {
        "ws-demo": {
          id: "ws-demo",
          rootPath: null,
          primarySummaryId: "doc-summ",
          label: "Demo research workspace",
        },
      },
    },
    browser: {
      sessions: browserSessions,
      activeSessionId: "sess-1",
      tabs: browserTabs,
      ui: { sidebarOpen: true, aiPanelOpen: true, aiPanelWidth: 360 },
    },
    reader: {
      session: { id: "rs-1", tabIds: ["rt-a", "rt-summ"], activeTabId: "rt-a" },
      tabs: {
        "rt-a": { id: "rt-a", documentId: "doc-a", label: "Paper A" },
        "rt-summ": { id: "rt-summ", documentId: "doc-summ", label: "Research Summary" },
      },
      ui: {
        railOpen: true,
        sidebarOpen: true,
        railView: "documents",
        aiPanelOpen: true,
        aiPanelWidth: 330,
        aiPanelTab: "chat",
        companion: null,
        summaryPick: null,
      },
    },
    sources,
    documents,
    changes,
    aiActivities,
    browse: { status: "idle", query: "", phase: 0, sourceIds: [] },
    selection: null,
    toast: null,
    settingsOpen: false,
    previewSourceId: null,
  };
}
