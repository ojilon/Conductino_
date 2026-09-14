/**
 * Reader mode layout:
 *   [rail + secondary panel] [document tabs + document view] [AI Reading panel]
 *
 * All open documents stay mounted (hidden) so per-document scroll
 * position, edits, highlights and the selection state survive switching
 * subtabs — a requirement, not a nicety, for research workflows.
 */

import { useApp, activeReaderDocument } from "../../state/appState";
import { ResizablePanel } from "../../components/ui";
import { uid } from "../../utils/helpers";
import { makeDocumentFromSource } from "../../mock/data";
import type { FileTreeNode, Source } from "../../types/domain";
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

  /** Open a workspace file as a reader document (mock document factory). */
  const openFile = (node: FileTreeNode & { path: string }) => {
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
    // File without a loaded document → mock-extract it into the DocumentModel.
    const sourceId = uid("src");
    const docId = uid("doc");
    const ext = node.ext ?? "pdf";
    const title = labelForPath(node.label);
    const source: Source = {
      id: sourceId,
      kind: ext === "docx" ? "docx" : ext === "html" ? "html" : ext === "txt" ? "text" : "pdf",
      title,
      origin: `research_workspace/${node.path}`,
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
      <ReaderSidebar doc={doc} onOpenFile={openFile} />

      <div className="flex min-w-0 flex-1 flex-col">
        <ReaderTabs onOpenFile={openFile} />
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
