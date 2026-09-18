/**
 * Reader mode layout:
 *   [rail + secondary panel] [document tabs + document view] [AI Reading panel]
 *
 * All open documents stay mounted (hidden) so per-document scroll
 * position, edits, highlights and the selection state survive switching
 * subtabs — a requirement, not a nicety, for research workflows.
 */

import { useCallback, useEffect, useState } from "react";
import { useApp, activeReaderDocument, workspaceIdFromRoot } from "../../state/appState";
import { backend, type OpenedFile } from "../../services/backend";
import { EmptyState, ResizablePanel } from "../../components/ui";
import { Icon } from "../../components/icons";
import { isStaleRoot, uid } from "../../utils/helpers";
import type { DocumentBlock, FileTreeNode, Source } from "../../types/domain";
import ReaderSidebar from "./ReaderSidebar";
import ReaderTabs, { labelForPath } from "./ReaderTabs";
import SourceDocumentView from "./DocumentView";
import SummaryDocumentView from "./SummaryDocumentView";
import PdfView from "./PdfView";
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

  const [tree, setTree] = useState<FileTreeNode | null | undefined>(undefined);
  const [treeError, setTreeError] = useState<string | null>(null);
  const [locatePath, setLocatePath] = useState<string | null>(null);
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

  // Files created/renamed outside the app (explorer, another program) must
  // appear without a manual refresh: re-list when the window regains focus.
  // Visible-only guard keeps background tabs from walking large folders.
  useEffect(() => {
    if (typeof window === "undefined") return;
    let pending = false;
    const onFocus = () => {
      if (document.visibilityState !== "visible" || pending) return;
      pending = true;
      refreshTree().finally(() => {
        pending = false;
      });
    };
    window.addEventListener("focus", onFocus);
    return () => window.removeEventListener("focus", onFocus);
  }, [refreshTree]);

  const pickFolder = useCallback(async () => {
    const picked = await backend.filesystem.selectFolder().catch(() => null);
    if (picked == null) {
      if (backend.mode === "mock") {
        dispatch({ type: "toast", message: "Folder picking needs the desktop app — run `wails dev`." });
      }
      return;
    }
    await refreshTree();
    const root = await backend.filesystem.libraryRoot().catch(() => null);
    const abs = root ?? picked;
    setCurrentRoot(abs);
    // Phase 2: each library root is its own workspace session so summaries
    // never cross folders.
    const wsId = workspaceIdFromRoot(abs);
    dispatch({
      type: "workspace.ensure",
      session: {
        id: wsId,
        rootPath: abs,
        primarySummaryId: null,
        label: abs.split(/[/\\]/).filter(Boolean).pop() ?? abs,
      },
    });
    dispatch({ type: "workspace.setActive", id: wsId });
    dispatch({ type: "toast", message: `Library: ${picked}` });
  }, [refreshTree, dispatch]);

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

  const openFile = async (node: FileTreeNode & { path: string }) => {
    // Already open? Focus the existing tab — never a duplicate tab.
    // Match by path token + opening root, so a same-named file from another
    // folder still opens fresh (stale-root guard, tasks.md §1.1).
    const openTabId = session.tabIds.find((t) => {
      const d = state.documents[state.reader.tabs[t]?.documentId ?? ""];
      if (!d || d.metadata.path !== node.path) return false;
      const root = d.metadata.rootPath;
      return !root || !currentRoot || root === currentRoot;
    });
    if (openTabId) {
      dispatch({ type: "reader.tab.select", tabId: openTabId });
      return;
    }
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
    let firstText = "";
    try {
      const parsed: unknown = JSON.parse(opened.blocksJSON);
      const raw = (parsed as { blocks: DocumentBlock[] }).blocks;
      if (!Array.isArray(raw)) throw new Error("bad blocks shape");
      // Normalize: Go nil slices marshal to JSON null, and any single null
      // (segments/listItems) used to throw past this try as an uncaught
      // promise rejection with no toast. Coerce here so every consumer below
      // (firstText, store, renderers) sees arrays.
      blocks = raw.map((b) => ({
        ...b,
        segments: Array.isArray(b.segments) ? b.segments : [],
        listItems: b.type === "list" && Array.isArray(b.listItems) ? b.listItems : b.listItems,
      }));
      firstText = blocks
        .map((b) => [...(b.segments ?? []), ...(b.listItems ?? []).flat()].map((s) => s.text).join(""))
        .find((t) => t.trim()) ?? "";
    } catch {
      dispatch({ type: "toast", message: `Could not open ${node.label}: unexpected content from the extractor` });
      return;
    }
    const sourceId = uid("src");
    const docId = uid("doc");
    const workspaceId = state.workspace.activeId ?? undefined;
    // Format follows the extractor (Go Kind), falling back to the file
    // extension: pdf → canvas leaf, docx/md/txt → block views.
    const format = (opened.kind === "pdf" || opened.kind === "docx" || opened.kind === "text")
      ? opened.kind
      : ext === "pdf" ? "pdf" : ext === "docx" ? "docx" : "text";
    const source: Source = {
      id: sourceId,
      workspaceId,
      kind: format,
      title: opened.title || labelForPath(node.label),
      origin: node.path,
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
        workspaceId,
        kind: "source",
        sourceId,
        metadata: {
          title: source.title,
          format,
          pageCount: opened.pageCount ?? 1,
          path: node.path,
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
                  {d.kind === "summary" ? (
                    <SummaryDocumentView doc={d} />
                  ) : d.metadata.format === "pdf" ? (
                    <PdfView doc={d} />
                  ) : (
                    <SourceDocumentView doc={d} />
                  )}
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
