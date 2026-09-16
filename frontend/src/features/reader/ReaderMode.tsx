/**
 * Reader mode layout:
 *   [rail + secondary panel] [document tabs + document view] [AI Reading panel]
 *
 * All open documents stay mounted (hidden) so per-document scroll
 * position, edits, highlights and the selection state survive switching
 * subtabs — a requirement, not a nicety, for research workflows.
 */

import { useCallback, useEffect, useState } from "react";
import { useApp, activeReaderDocument } from "../../state/appState";
import { backend } from "../../services/backend";
import { ResizablePanel } from "../../components/ui";
import { uid } from "../../utils/helpers";
import { makeDocumentFromSource } from "../../mock/data";
import type { DocumentBlock, FileTreeNode, Source } from "../../types/domain";
import ReaderSidebar from "./ReaderSidebar";
import ReaderTabs, { labelForPath } from "./ReaderTabs";
import SourceDocumentView from "./DocumentView";
import SummaryDocumentView from "./SummaryDocumentView";
import AIReadingPanel from "./AIReadingPanel";

export default function ReaderMode() {
  const { state, dispatch } = useApp();
  const ui = state.reader.ui;
  const doc = activeReaderDocument(state);
  const session = state.reader.session;

  // Single owner of the library tree: undefined = loading, null = no
  // folder yet (empty state), FileTreeNode = live tree (mock data in browser
  // mode, real Workspace-owned tree in the desktop app).
  const [tree, setTree] = useState<FileTreeNode | null | undefined>(undefined);
  const [treeError, setTreeError] = useState<string | null>(null);
  // Path token the Library panel should expand + highlight (set by "Show in
  // library" in the Documents panel, cleared once consumed there).
  const [locatePath, setLocatePath] = useState<string | null>(null);

  const refreshTree = useCallback(async () => {
    try {
      setTreeError(null);
      setTree(await backend.library.list());
    } catch (e) {
      setTreeError(e instanceof Error ? e.message : "Could not load library");
    }
  }, []);

  useEffect(() => {
    let live = true;
    backend.library
      .list()
      .then((t) => live && setTree(t))
      .catch((e: unknown) => live && setTreeError(e instanceof Error ? e.message : "Could not load library"));
    return () => {
      live = false;
    };
  }, []);

  // Folder dialog → backend repoints the library itself (App.SelectFolder) →
  // re-read. Null = cancelled: keep the current tree, no toast. In browser
  // mock mode there is no dialog at all — say so instead of staying silent.
  const pickFolder = useCallback(async () => {
    const picked = await backend.filesystem.selectFolder().catch(() => null);
    if (picked == null) {
      if (backend.mode === "mock") {
        dispatch({ type: "toast", message: "Folder picking needs the desktop app — run `wails dev`." });
      }
      return;
    }
    await refreshTree();
    dispatch({ type: "toast", message: `Library: ${picked}` });
  }, [refreshTree, dispatch]);

  // Documents panel → Library panel handoff: switch rail views and ask the
  // tree to reveal this file's location.
  const showInLibrary = useCallback(
    (path: string) => {
      setLocatePath(path);
      dispatch({ type: "reader.ui", patch: { railView: "library", sidebarOpen: true } });
    },
    [dispatch],
  );

  /** Open a workspace file as a reader document. */
  const openFile = async (node: FileTreeNode & { path: string }) => {
    if (node.documentId) {
      const existingTab = session.tabIds.find((t) => state.reader.tabs[t]?.documentId === node.documentId);
      if (existingTab) {
        dispatch({ type: "reader.tab.select", tabId: existingTab });
        return;
      }
      const title = state.documents[node.documentId]?.metadata.title ?? node.label;
      const short: Record<string, string> = { "doc-c": "Paper C" };
      dispatch({
        type: "reader.doc.open",
        tabId: uid("rt"),
        documentId: node.documentId,
        label: short[node.documentId] ?? (title.length > 18 ? `${title.split(" ")[0]} ${title.split(" ")[1] ?? ""}` : title),
      });
      return;
    }
    // File without a loaded document → try the real pipe first
    // (App.OpenFile: extension-dispatched extraction, .txt/.md today).
    // Anything else (mock mode, unsupported type, read error) falls back to
    // the mock factory so the click still opens something demonstrable.
    const ext = (node.ext ?? "pdf").toLowerCase();
    try {
      const opened = await backend.filesystem.openFile(node.path);
      if (opened) {
        const parsed: unknown = JSON.parse(opened.blocksJSON);
        const blocks = (parsed as { blocks: DocumentBlock[] }).blocks;
        if (!Array.isArray(blocks)) throw new Error("bad blocks shape");
        const sourceId = uid("src");
        const docId = uid("doc");
        const firstText = blocks
          .map((b) => b.segments.map((s) => s.text).join(""))
          .find((t) => t.trim()) ?? "";
        const source: Source = {
          id: sourceId,
          kind: "text",
          title: opened.title || labelForPath(node.label),
          origin: node.path, // root-relative token, not a built path
          typeLabel: `${ext.toUpperCase()} document`,
          abstract: firstText.slice(0, 280),
          saved: false,
          inReader: true,
          documentId: docId,
        };
        dispatch({ type: "source.add", source });
        dispatch({
          type: "doc.add",
          document: {
            id: docId,
            kind: "source",
            sourceId,
            metadata: {
              title: source.title,
              format: "text",
              pageCount: opened.pageCount ?? 1,
              path: node.path,
            },
            blocks,
            highlights: [],
            currentPage: 1,
          },
        });
        const label = source.title;
        dispatch({ type: "reader.doc.open", tabId: uid("rt"), documentId: docId, label: label.length > 18 ? `${label.slice(0, 16)}…` : label });
        dispatch({ type: "toast", message: `Opened ${node.label}` });
        return;
      }
    } catch {
      // fall through to the mock factory below
    }
    // Mock fallback: fabricate the DocumentModel locally.
    const sourceId = uid("src");
    const docId = uid("doc");
    const title = labelForPath(node.label);
    const source: Source = {
      id: sourceId,
      kind: ext === "docx" ? "docx" : ext === "html" ? "html" : ext === "txt" ? "text" : "pdf",
      title,
      origin: node.path, // already root-relative (FileTreeNode.path token)
      typeLabel: `${ext.toUpperCase()} document`,
      abstract: `Workspace file “${node.path}”, extracted with the mock document pipeline. Real extraction (pdf.js / docx) is a documented integration point — see docs/document-rendering.md.`,
      saved: false,
      inReader: true,
      documentId: docId,
    };
    dispatch({ type: "source.add", source });
    dispatch({ type: "doc.add", document: makeDocumentFromSource(sourceId, source, docId) });
    dispatch({ type: "reader.doc.open", tabId: uid("rt"), documentId: docId, label: title.length > 18 ? `${title.slice(0, 16)}…` : title });
    dispatch({ type: "toast", message: `Opened ${node.label} (mock extraction)` });
  };

  return (
    <div className="flex h-full min-h-0">
      <ReaderSidebar
        doc={doc}
        onOpenFile={openFile}
        tree={tree}
        treeError={treeError}
        onPickFolder={pickFolder}
        onRefreshTree={refreshTree}
        locatePath={locatePath}
        onLocateDone={() => setLocatePath(null)}
        onShowInLibrary={showInLibrary}
      />

      <div className="flex min-w-0 flex-1 flex-col">
        <ReaderTabs onOpenFile={openFile} tree={tree} />
        <div
          className="min-h-0 flex-1 overflow-y-auto bg-cream-50"
          onScroll={() => {
            if (state.selection) dispatch({ type: "selection.set", selection: null });
          }}
        >
          {session.tabIds.length === 0 ? (
            <div className="flex h-full items-center justify-center">
              <p className="text-[13px] text-mute">No documents open — use “Open document” above.</p>
            </div>
          ) : (
            session.tabIds.map((tabId) => {
              const tab = state.reader.tabs[tabId];
              if (!tab) return null;
              const d = state.documents[tab.documentId];
              if (!d) return null;
              const active = tabId === session.activeTabId;
              return (
                <div key={tabId} className={active ? "block min-h-full" : "hidden"}>
                  {d.kind === "summary" ? <SummaryDocumentView doc={d} /> : <SourceDocumentView doc={d} />}
                </div>
              );
            })
          )}
        </div>
      </div>

      {ui.aiPanelOpen && doc && (
        <ResizablePanel
          width={ui.aiPanelWidth}
          min={300}
          max={520}
          onWidth={(w) => dispatch({ type: "reader.ui", patch: { aiPanelWidth: w } })}
        >
          <AIReadingPanel doc={doc} />
        </ResizablePanel>
      )}
    </div>
  );
}
