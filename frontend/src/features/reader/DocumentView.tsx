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

import { useRef, useState } from "react";
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
        {(block.segments ?? []).map((s, i) => (
          <span key={i}>{s.text}</span>
        ))}
      </h2>
    );
  }

  // Page-break anchor (PDF extraction emits one per page): a thin rule the
  // canvas leaf and chat both key pages by. Carries data-block-id like text.
  if (block.type === "page") {
    return (
      <div className="my-6 flex items-center gap-3 text-mute" aria-hidden>
        <span className="h-px flex-1 bg-line-soft" />
        <span className="text-[10.5px] font-medium uppercase tracking-wide">page</span>
        <span className="h-px flex-1 bg-line-soft" />
      </div>
    );
  }

  if (block.type === "list") {
    return (
      <ul className="my-4 list-disc space-y-2 pl-6 doc-body">
        {(block.listItems ?? []).map((item, i) => (
          <li key={i}>
            {(item ?? []).map((s, j) => (
              <span key={j} className={s.em ? "italic" : undefined}>
                {s.text}
              </span>
            ))}
          </li>
        ))}
      </ul>
    );
  }

  // Range-precise highlight (issue 8): a saved note with offsets underlines
  // exactly the span. Single-segment blocks split cleanly; multi-segment
  // blocks keep the whole-block treatment below (no style loss).
  const full = (block.segments ?? []).map((s) => s.text).join("");
  const ranged = doc?.highlights.find(
    (h) => h.blockId === block.id && h.range && h.range.end > h.range.start,
  );
  const r = ranged?.range;
  if (r && block.segments.length === 1 && r.end <= full.length) {
    return (
      <p className="my-4 doc-body">
        {full.slice(0, r.start)}
        <mark className="rounded-[3px] bg-moss-100 px-0.5 text-moss-700">{full.slice(r.start, r.end)}</mark>
        {full.slice(r.end)}
      </p>
    );
  }

  return (
    <p className={cn("my-4 doc-body", noted && "underline decoration-hay-300 decoration-[3px] underline-offset-[6px]")}>
      {(block.segments ?? []).map((s, i) =>
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
  const [promptOpen, setPromptOpen] = useState(false);
  const [customPrompt, setCustomPrompt] = useState("");
  const sel = state.selection;
  if (!sel || sel.documentId !== doc.id) return null;

  const asRef = (): SelectionRef => ({ documentId: sel.documentId, blockId: sel.blockId, text: sel.text, range: sel.range });
  const x = clamp(sel.x, 200, window.innerWidth - 200);
  const y = Math.max(70, sel.y - 16);

  const runWithPrompt = (op: "AI_EXPLAIN" | "AI_VERIFY" | "AI_EXPAND") => {
    const prompt = customPrompt.trim() || undefined;
    runExplain(op, asRef(), prompt);
    setPromptOpen(false);
    setCustomPrompt("");
    dispatch({ type: "selection.set", selection: null });
  };

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
      className="fade-in fixed z-50 flex flex-col gap-1 rounded-xl border border-line bg-white px-1.5 py-1 shadow-xl shadow-ink-900/10"
      style={{ left: x, top: y, transform: "translate(-50%, -100%)", minWidth: promptOpen ? 320 : undefined }}
    >
      <div className="flex items-center gap-0.5">
        {act("Ask AI", "sparkles", () => setPromptOpen((v) => !v), true)}
        {act("Explain", "bulb", () => runWithPrompt("AI_VERIFY"))}
        {act("Go deeper", "search", () => runWithPrompt("AI_EXPAND"))}
        {act("Include in summary", "list", () => {
          runIncludeInSummary(asRef());
          dispatch({ type: "selection.set", selection: null });
        })}
        {act("Save note", "bookmark", () => {
          dispatch({
            type: "doc.highlight.add",
            documentId: doc.id,
            highlight: { id: uid("hl"), blockId: sel.blockId, text: sel.text, range: sel.range, note: "Saved from selection", createdAt: Date.now() },
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
      {promptOpen && (
        <div className="flex flex-col gap-1.5 border-t border-line-soft px-1.5 pb-1.5 pt-1">
          <textarea
            autoFocus
            rows={2}
            value={customPrompt}
            onChange={(e) => setCustomPrompt(e.target.value)}
            placeholder="Optional instruction (e.g. focus on the mechanism)…"
            className="w-full resize-none rounded-lg border border-line bg-cream-50 px-2 py-1.5 text-[12px] text-ink-900 placeholder:text-mute focus:border-iris-400 focus:outline-none"
            onKeyDown={(e) => {
              if (e.key === "Enter" && !e.shiftKey) {
                e.preventDefault();
                runWithPrompt("AI_EXPLAIN");
              }
            }}
          />
          <div className="flex justify-end gap-1">
            <button
              type="button"
              className="rounded-lg px-2 py-1 text-[11px] text-mute hover:bg-cream-100"
              onClick={() => setPromptOpen(false)}
            >
              Cancel
            </button>
            <button
              type="button"
              className="rounded-lg bg-iris-600 px-2.5 py-1 text-[11px] font-medium text-white hover:bg-iris-700"
              onClick={() => runWithPrompt("AI_EXPLAIN")}
            >
              Ask AI
            </button>
          </div>
        </div>
      )}
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
    const domRange = sel && sel.rangeCount > 0 ? sel.getRangeAt(0) : null;
    const rect = domRange?.getBoundingClientRect();
    if (!rect) return;
    // Range offsets within the block (issue 8): measure by cloning a range
    // over the block contents and ending it at the selection start. Only
    // when the whole selection sits inside this block element; otherwise
    // stay block-anchored (range undefined).
    let range: { start: number; end: number } | undefined;
    if (domRange) {
      try {
        const inside = (n: Node | null) => !!n && (blockEl === n || blockEl.contains(n));
        if (inside(domRange.startContainer) && inside(domRange.endContainer)) {
          const pre = domRange.cloneRange();
          pre.selectNodeContents(blockEl);
          pre.setEnd(domRange.startContainer, domRange.startOffset);
          const start = pre.toString().length;
          const full = blockEl.textContent?.length ?? 0;
          const end = Math.min(full, start + domRange.toString().length);
          if (end > start) range = { start, end };
        }
      } catch {
        range = undefined; // measuring must never break selection
      }
    }
    dispatch({
      type: "selection.set",
      selection: {
        documentId: doc.id,
        blockId: blockEl.getAttribute("data-block-id") as string,
        text,
        range,
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
