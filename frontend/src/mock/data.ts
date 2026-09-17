/**
 * Lumen seed data — intentionally EMPTY.
 *
 * The app starts with no documents, sources, tabs, or AI history: every
 * section shows its empty state until the user picks a library folder and
 * opens a real file (desktop) or sends a source to the Reader. Nothing here
 * fabricates research content.
 *
 * What remains in this file (and why):
 *  - `fileTreeMock`: folder SHAPE only (no files mapped to documents), so
 *    browser/mock mode still demonstrates the Library tree UI.
 *  - `makeDocumentFromSource`: builds a reader document from a real Source
 *    record (used by "Send to Reader"); it renders that source's own
 *    abstract, it invents nothing.
 *  - `createInitialState`: the empty AppState (one blank browser tab so the
 *    browser chrome has a valid session; reader fully empty).
 */

import type {
  AppState,
  BrowserSession,
  BrowserTab,
  Document,
  FileTreeNode,
  Source,
} from "../types/domain";

/* ------------------------------------------------------------------ */
/* Browser shell (one blank session — chrome only, no content)         */
/* ------------------------------------------------------------------ */

const tab = (id: string, title: string, url: string): BrowserTab => ({
  id,
  title,
  nav: { currentUrl: url, history: [url], index: 0 },
});

const browserSessions: BrowserSession[] = [{ id: "sess-1", title: "Session 1", tabIds: ["t-new"], activeTabId: "t-new" }];

const browserTabs: Record<string, BrowserTab> = {
  "t-new": tab("t-new", "New tab", "lumen://newtab"),
};

/* ------------------------------------------------------------------ */
/* File tree (mock of the Go filesystem service — shape only)          */
/* ------------------------------------------------------------------ */

export const fileTreeMock: FileTreeNode = {
  id: "ft-root",
  label: "research_workspace",
  kind: "folder",
  children: [
    {
      id: "ft-papers",
      label: "research_papers",
      kind: "folder",
      children: [
        { id: "ft-figures", label: "figures", kind: "folder" },
        { id: "ft-suppl", label: "supplementary.pdf", kind: "file", ext: "pdf" },
      ],
    },
    {
      id: "ft-notes",
      label: "notes",
      kind: "folder",
      children: [{ id: "ft-field", label: "field_notes.docx", kind: "file", ext: "docx" }],
    },
  ],
};

/* ------------------------------------------------------------------ */
/* Document factory — used by "Send to Reader"                         */
/* ------------------------------------------------------------------ */

export function makeDocumentFromSource(sourceId: string, source: Source, docId: string): Document {
  const isWeb = source.kind === "web" || source.kind === "html";
  return {
    id: docId,
    kind: "source",
    sourceId,
    currentPage: 1,
    metadata: {
      title: source.title,
      author: source.origin,
      venue: source.typeLabel,
      format: source.kind === "pdf" ? "pdf" : source.kind === "docx" ? "docx" : isWeb ? "html" : "text",
      pageCount: source.kind === "pdf" ? 8 : 3,
      path: source.url ? undefined : source.origin,
    },
    highlights: [],
    blocks: [
      { id: `${docId}-s1`, type: "heading", level: 2, segments: [{ text: isWeb ? "About this article" : "Overview" }] },
      { id: `${docId}-p1`, type: "paragraph", segments: [{ text: source.abstract }] },
      { id: `${docId}-s2`, type: "heading", level: 2, segments: [{ text: "Why this is in your session" }] },
      {
        id: `${docId}-p2`,
        type: "paragraph",
        segments: [
          {
            text: isWeb
              ? "Opened from an AI Browse result."
              : "Opened from the workspace file tree.",
          },
        ],
      },
    ],
  };
}

/* ------------------------------------------------------------------ */
/* Initial app state — empty                                           */
/* ------------------------------------------------------------------ */

export function createInitialState(): AppState {
  return {
    mode: "reader",
    browser: {
      sessions: browserSessions,
      activeSessionId: "sess-1",
      tabs: browserTabs,
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
        companion: null,
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
