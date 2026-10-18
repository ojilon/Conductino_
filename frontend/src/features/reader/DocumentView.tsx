/**
 * Source document view — header + MIRROR CANVAS.
 *
 * The heavy lifting moved to two focused components:
 *   - MirrorCanvas.tsx    — the custom paginated canvas (page size/fonts
 *                           from the mirror metadata, direct editing for
 *                           docx/txt/md, colored highlights + right-click
 *                           menu, Save-to-file bridge).
 *   - SelectionToolbar.tsx— the floating "Ask AI / highlight / To summary"
 *                           toolbar shared with the PDF view.
 *
 * This file keeps the document header (title, page count, "Make summary")
 * and glues the canvas + toolbar together.
 */

import { useApp, activeWorkspace } from "../../state/appState";
import { setPrimarySummary, setSummaryTargetFile } from "../../services/backend";
import type { Document } from "../../types/domain";
import { Icon } from "../../components/icons";
import MirrorCanvas from "./MirrorCanvas";
import { SelectionToolbar } from "./SelectionToolbar";

export { SelectionToolbar };

export default function SourceDocumentView({ doc }: { doc: Document }) {
  const { state, dispatch } = useApp();
  const meta = doc.metadata;

  /**
   * Designate this document as the workspace summary target:
   *  1. in-app: it becomes the AI merge/propose target (workspace state),
   *  2. on disk: when it is a real .docx, mark it in the backend so ALL
   *     summaries are siphoned into it — including publishes while it is
   *     closed, and the AI finds it via the get_summary_target tool.
   */
  const makeSummary = async () => {
    const wsId = activeWorkspace(state)?.id ?? doc.workspaceId;
    if (!wsId) {
      dispatch({ type: "toast", message: "Open a workspace folder first." });
      return;
    }
    dispatch({ type: "workspace.setPrimarySummary", workspaceId: wsId, summaryId: doc.id });
    void setPrimarySummary(doc.id);
    if (meta.path && meta.format === "docx") {
      try {
        const ok = await setSummaryTargetFile(meta.path);
        if (ok) {
          dispatch({ type: "toast", message: `"${meta.title}" is now the workspace summary target — summaries are siphoned into it` });
          return;
        }
      } catch (e) {
        dispatch({ type: "toast", message: `Summary target failed: ${e instanceof Error ? e.message : String(e)}` });
        return;
      }
    }
    dispatch({ type: "toast", message: `"${meta.title}" is now the workspace summary` });
  };

  return (
    <div className="mx-auto max-w-[860px] px-6 py-6">
      <div className="mb-4 flex items-center justify-between border-b border-line pb-2.5 text-[12px] text-mute">
        <span className="truncate">
          {meta.author ? `${meta.author} · ` : ""}
          {meta.title}
        </span>
        <span className="flex shrink-0 items-center gap-3 pl-3">
          <button
            type="button"
            onClick={() => void makeSummary()}
            title="Make this document the workspace summary target (editable, AI edit target)"
            className="rounded-full border border-line-soft px-2.5 py-0.5 text-[10.5px] text-mute hover:border-iris-300 hover:text-iris-700"
          >
            Make summary target
          </button>
          {meta.pageCount && <span>{meta.pageCount} pages</span>}
        </span>
      </div>

      <h1 className="mb-6 font-serif text-[26px] font-bold leading-tight text-ink-900">{meta.title}</h1>

      <MirrorCanvas doc={doc} />

      <p className="mx-auto mt-6 flex max-w-[640px] items-start gap-2 text-[11.5px] leading-relaxed text-mute">
        <Icon name="info" size={12} className="mt-0.5 shrink-0" />
        {meta.editable === false
          ? "Read-only mirror — select text to highlight, ask the AI, or send passages to the summary target."
          : "Editable mirror — type right on the page, Enter splits a paragraph, Ctrl+S (or “Save to file”) verifies the changed blocks and writes them back to the source file."}
      </p>

      <SelectionToolbar doc={doc} />
    </div>
  );
}
