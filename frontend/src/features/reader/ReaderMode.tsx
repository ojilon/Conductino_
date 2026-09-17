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
import { backend, type OpenedFile } from "../../services/backend";
import { EmptyState, ResizablePanel } from "../../components/ui";
import { Icon } from "../../components/icons";
import { isStaleRoot, uid } from "../../utils/helpers";
import type { DocumentBlock, FileTreeNode, Source } from "../../types/domain";
import ReaderSidebar from "./ReaderSidebar";
import ReaderTabs, { labelForPath } from "./ReaderTabs";
import SourceDocumentView from "./DocumentView";
import SummaryDocumentView from "./SummaryDocumentView";
import AIReadingPanel from "./AIReadingPanel";

/** Human-readable label per typed open failure (tasks.md 1.2). */
function openFailureLabel(reason: NonNullable<OpenedFile["reason"]>): string {
  switch (reason) {
    case "unsupported":
      return "extraction is not implemented for this file type yet";
    case "not_found":
      return "file not found";
    case "permission_denied":
      return "access denied";
    case "too_large":
      return "file too large to open";
    case "parse_error":
      return "could not be read";
  }
}

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
  // Absolute workspace root tagging the current folder (tasks.md 1.1 option b).
  // Null = unknown (mock mode / bridge unavailable) — stale checks are skipped.
  const [currentRoot, setCurrentRoot] = useState<string | null>(null);

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
    backend.filesystem
      .libraryRoot()
      .then((r) => live && r && setCurrentRoot(r))
      .catch(() => {});
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
    // The dialog already repointed the Go root; the returned absolute path is
    // the new current root. Refresh from the bridge as well in case the shell
    // normalized it — open tabs keep their old rootPath tags (1.1 option b).
    const root = await backend.filesystem.libraryRoot().catch(() => null);
    setCurrentRoot(root ?? picked);
    dispatch({ type: "toast", message: `Library: ${picked}` });
  }, [refreshTree, dispatch]);

  // Documents panel → Library panel handoff: switch rail views and ask the
  // tree to reveal this file's location. Stale guard (tasks.md 1.1 option b):
  // a tab tagged with another folder's root never resolves against the
  // current tree — explicit message instead of a silent mis-locate.
  const showInLibrary = useCallback(
    (path: string, rootPath?: string) => {
      if (isStaleRoot(rootPath, currentRoot)) {
        dispatch({ type: "toast", message: "This file belongs to another folder — switch back to reveal it in the library." });
        return;
      }
      setLocatePath(path);
      dispatch({ type: "reader.ui", patch: { railView: "library", sidebarOpen: true } });
    },
    [dispatch, currentRoot],
  );

  /** Open a workspace file as a reader document. */
  const openFile = async (node: FileTreeNode & { path: string }) => {
    if (node.documentId) {
      const existingTab = session.tabIds.find((t) => state.reader.tabs[t]?.documentId === node.documentId);
      if (existingTab) {
        dispatch({ type: "reader.tab.select", tabId: existingTab });
        return;
      }
      // The linked document may be gone (stale tree entry): never open a
      // ghost tab — fall through to the real extraction pipe below, which
      // either opens the file's content or reports an honest error.
      const linked = state.documents[node.documentId];
      if (linked) {
        const title = linked.metadata.title;
        dispatch({
          type: "reader.doc.open",
          tabId: uid("rt"),
          documentId: node.documentId,
          label: title.length > 18 ? `${title.split(" ")[0]} ${title.split(" ")[1] ?? ""}` : title,
        });
        return;
      }
    }
    // File without a loaded document → real extraction pipe only
    // (App.OpenFile: extension-dispatched extraction, .txt/.md today).
    // No mock fallback (tasks.md 1.2): every failure — unsupported type,
    // missing file, access denied, too large, unreadable — shows an honest
    // error naming the file and reason, and opens nothing.
    const ext = (node.ext ?? "pdf").toLowerCase();
    let opened: OpenedFile | null;
    try {
      opened = await backend.filesystem.openFile(node.path);
    } catch (e) {
      dispatch({ type: "toast", message: `Could not open ${node.label}: ${e instanceof Error ? e.message : "unexpected error"}` });
      return;
    }
    if (!opened) {
      dispatch({ type: "toast", message: "Opening files needs the desktop app — run `wails dev`." });
      return;
    }
    if (opened.reason) {
      dispatch({ type: "toast", message: `Could not open ${node.label}: ${openFailureLabel(opened.reason)}${opened.detail ? ` — ${opened.detail}` : ""}` });
      return;
    }
    let blocks: DocumentBlock[];
    try {
      const parsed: unknown = JSON.parse(opened.blocksJSON);
      blocks = (parsed as { blocks: DocumentBlock[] }).blocks;
      if (!Array.isArray(blocks)) throw new Error("bad blocks shape");
    } catch {
      dispatch({ type: "toast", message: `Could not open ${node.label}: unexpected content from the extractor` });
      return;
    }
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
          // Tag with the opening folder (1.1 option b): Go's absolute root
          // wins (race-free); fall back to the UI-known current root.
          rootPath: opened.root ?? currentRoot ?? undefined,
        },
        blocks,
        highlights: [],
        currentPage: 1,
      },
    });
    const label = source.title;
    dispatch({ type: "reader.doc.open", tabId: uid("rt"), documentId: docId, label: label.length > 18 ? `${label.slice(0, 16)}…` : label });
    dispatch({ type: "toast", message: `Opened ${node.label}` });
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
        currentRoot={currentRoot}
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

      {ui.aiPanelOpen &&
        (doc ? (
          <ResizablePanel
            width={ui.aiPanelWidth}
            min={300}
            max={520}
            onWidth={(w) => dispatch({ type: "reader.ui", patch: { aiPanelWidth: w } })}
          >
            <AIReadingPanel doc={doc} />
          </ResizablePanel>
        ) : (
          <ResizablePanel
            width={ui.aiPanelWidth}
            min={300}
            max={520}
            onWidth={(w) => dispatch({ type: "reader.ui", patch: { aiPanelWidth: w } })}
          >
            <div className="flex h-full min-h-0 flex-col">
              <div className="flex items-center justify-between border-b border-line-soft px-4 py-3">
                <div className="flex items-center gap-2">
                  <Icon name="sparkles" size={15} className="text-iris-600" />
                  <h2 className="font-serif text-[15px] font-semibold text-ink-900">AI Reading</h2>
                </div>
              </div>
              <div className="flex min-h-0 flex-1 flex-col justify-center">
                <EmptyState
                  icon="sparkles"
                  title="No document open"
                  hint="Open a document first to show AI response."
                />
              </div>
            </div>
          </ResizablePanel>
        ))}
    </div>
  );
}
