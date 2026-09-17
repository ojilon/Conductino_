/**
 * Source document view (read-only research documents).
 *
 * Renders the structured DocumentModel (headings / paragraphs / lists /
 * highlights). A FUTURE real renderer (pdf.js, docx-preview, …) replaces
 * the block rendering only — selection, AI actions and highlights are
 * driven by the DocumentModel + app state, not by this file.
 *
 * Selection workflow:
 *   native text selection → TextSelection in state → floating
 *   SelectionToolbar → AI operation → streaming activity → result.
 */

import { useRef } from "react";
import { useApp } from "../../state/appState";
import { useAIRunners, type SelectionRef } from "../../state/aiController";
import type { Document, DocumentBlock } from "../../types/domain";
import { Icon } from "../../components/icons";
import { clamp, uid } from "../../utils/helpers";
import { cn } from "../../utils/cn";

/* ------------------------------------------------------------------ */
/* Blocks                                                              */
/* ------------------------------------------------------------------ */

export function BlockText({ block, docId }: { block: DocumentBlock; docId: string }) {
  const { state } = useApp();
  const doc = state.documents[docId];
  const noted = doc?.highlights.some((h) => h.blockId === block.id) ?? false;

  if (block.type === "heading") {
    return (
      <h2 className="mt-8 font-serif text-[19px] font-bold text-ink-900 first:mt-0">
        {block.segments.map((s, i) => (
          <span key={i}>{s.text}</span>
        ))}
      </h2>
    );
  }

  if (block.type === "list") {
    return (
      <ul className="my-4 list-disc space-y-2 pl-6 doc-body">
        {(block.listItems ?? []).map((item, i) => (
          <li key={i}>
            {item.map((s, j) => (
              <span key={j} className={s.em ? "italic" : undefined}>
                {s.text}
              </span>
            ))}
          </li>
        ))}
      </ul>
    );
  }

  return (
    <p className={cn("my-4 doc-body", noted && "underline decoration-hay-300 decoration-[3px] underline-offset-[6px]")}>
      {block.segments.map((s, i) =>
        s.highlightId ? (
          <mark key={i} className="rounded-[3px] bg-moss-100 px-0.5 text-moss-700">
            {s.text}
          </mark>
        ) : (
          <span key={i} className={cn(s.em && "italic", s.strong && "font-semibold")}>
            {s.text}
          </span>
        ),
      )}
    </p>
  );
}

/* ------------------------------------------------------------------ */
/* Floating selection toolbar                                          */
/* ------------------------------------------------------------------ */

export function SelectionToolbar({ doc }: { doc: Document }) {
  const { state, dispatch } = useApp();
  const { runExplain, runIncludeInSummary } = useAIRunners();
  const sel = state.selection;
  if (!sel || sel.documentId !== doc.id) return null;

  const asRef = (): SelectionRef => ({ documentId: sel.documentId, blockId: sel.blockId, text: sel.text });
  const x = clamp(sel.x, 200, window.innerWidth - 200);
  const y = Math.max(70, sel.y - 16);

  const act = (label: string, icon: "sparkles" | "bulb" | "search" | "list" | "bookmark", onClick: () => void, accent = false) => (
    <button
      type="button"
      onClick={onClick}
      className={cn(
        "flex items-center gap-1.5 rounded-lg px-2.5 py-1.5 text-[12px] font-medium transition-colors",
        accent ? "text-iris-700 hover:bg-iris-50" : "text-ink-700 hover:bg-cream-100",
      )}
    >
      <Icon name={icon} size={13} className={accent ? "text-iris-600" : "text-mute"} />
      {label}
    </button>
  );

  return (
    <div
      data-sel-toolbar
      className="fade-in fixed z-50 flex items-center gap-0.5 rounded-xl border border-line bg-white px-1.5 py-1 shadow-xl shadow-ink-900/10"
      style={{ left: x, top: y, transform: "translate(-50%, -100%)" }}
    >
      {act("Ask AI", "sparkles", () => runExplain("AI_EXPLAIN", asRef()), true)}
      {act("Explain", "bulb", () => runExplain("AI_VERIFY", asRef()))}
      {act("Go deeper", "search", () => runExplain("AI_EXPAND", asRef()))}
      {act("Include in summary", "list", () => runIncludeInSummary(asRef()))}
      {act("Save note", "bookmark", () => {
        dispatch({
          type: "doc.highlight.add",
          documentId: doc.id,
          highlight: { id: uid("hl"), blockId: sel.blockId, text: sel.text, note: "Saved from selection", createdAt: Date.now() },
        });
        dispatch({ type: "selection.set", selection: null });
        dispatch({ type: "toast", message: "Note saved to this document" });
      })}
      <span className="mx-0.5 h-4 w-px bg-line" />
      <button
        type="button"
        title="Dismiss"
        className="rounded-lg p-1.5 text-mute transition-colors hover:bg-cream-100 hover:text-ink-900"
        onClick={() => dispatch({ type: "selection.set", selection: null })}
      >
        <Icon name="x" size={12} />
      </button>
    </div>
  );
}

/* ------------------------------------------------------------------ */
/* View                                                                */
/* ------------------------------------------------------------------ */

export default function SourceDocumentView({ doc }: { doc: Document }) {
  const { state, dispatch } = useApp();
  const scrollRef = useRef<HTMLDivElement>(null);
  const meta = doc.metadata;

  const onMouseUp = () => {
    const sel = window.getSelection();
    const text = sel?.toString().trim() ?? "";
    if (!text || text.length < 4) {
      if (state.selection) dispatch({ type: "selection.set", selection: null });
      return;
    }
    const node = sel?.anchorNode;
    const el = node instanceof Element ? node : node?.parentElement;
    const blockEl = el?.closest("[data-block-id]");
    if (!blockEl || !blockEl.closest("[data-doc-area]")) {
      dispatch({ type: "selection.set", selection: null });
      return;
    }
    const rect = sel?.getRangeAt(0).getBoundingClientRect();
    if (!rect) return;
    dispatch({
      type: "selection.set",
      selection: {
        documentId: doc.id,
        blockId: blockEl.getAttribute("data-block-id") as string,
        text,
        x: rect.left + rect.width / 2,
        y: rect.top,
      },
    });
  };

  const onMouseDown = (e: React.MouseEvent) => {
    if (!state.selection) return;
    const target = e.target as HTMLElement;
    if (target.closest("[data-sel-toolbar]") || target.closest("[data-doc-area]")) return;
    dispatch({ type: "selection.set", selection: null });
  };

  return (
    <div className="mx-auto max-w-[700px] px-8 py-6" onMouseDown={onMouseDown}>
      {/* page header */}
      <div className="mb-4 flex items-center justify-between border-b border-line pb-2.5 text-[12px] text-mute">
        <span className="truncate">
          {meta.author ? `${meta.author} · ` : ""}
          {meta.title}
        </span>
        {meta.pageCount && (
          <span className="shrink-0 pl-3">
            Page {doc.currentPage ?? 1} of {meta.pageCount}
          </span>
        )}
      </div>

      <h1 className="mb-6 font-serif text-[28px] font-bold leading-tight text-ink-900">{meta.title}</h1>

      <div
        ref={scrollRef}
        data-doc-area
        className="select-text"
        onMouseUp={onMouseUp}
        onScroll={() => {
          if (state.selection?.documentId === doc.id) dispatch({ type: "selection.set", selection: null });
        }}
      >
        {doc.blocks.map((block) => (
          <div key={block.id} data-block-id={block.id} id={`blk-${block.id}`}>
            <BlockText block={block} docId={doc.id} />
          </div>
        ))}
        <p className="mt-10 flex items-center gap-2 border-t border-line-soft pt-4 text-[11.5px] text-mute">
          <Icon name="info" size={12} />
          {meta.path ?? "Workspace file"} · {meta.format.toUpperCase()} document
        </p>
      </div>

      <SelectionToolbar doc={doc} />
    </div>
  );
}
