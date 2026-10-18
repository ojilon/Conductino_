/**
 * MirrorCanvas — the CUSTOM CANVAS for mirrored documents (docx/txt/md).
 *
 * This is the frontend half of the mirror pipeline (see backend/docpipe):
 * the Go side extracted the source file into content blocks + a metadata
 * GUIDE (meta.json: page size, fonts, margins, page map). This component
 * renders those blocks as PAGES its own way — no Word, no XML — using the
 * guide to stay close to how the source looks:
 *
 *   - page width/height come from meta.pageWidthPt/pageHeightPt (scaled to
 *     fit a comfortable column — the app's own interpretation),
 *   - margins come from meta.marginsPt,
 *   - the body font comes from meta.defaultFont,
 *   - pagination follows meta.charsPerPage (or real "page" blocks for PDFs).
 *
 * Editing (docx/txt/md): each block is a plain-text contentEditable that
 * commits on blur (caret-safe: typing never touches React state) and Enter
 * splits a block — the same proven pattern as the summary editor. PDF stays
 * read-only (meta.editable=false): highlight only.
 *
 * Highlights: colored <mark> runs (yellow = AI/review by convention). A
 * right-click on any highlight opens a small menu: keep it permanently,
 * recolor, remove, or ask the AI about that exact passage (the prompt
 * carries the quoted text + this file's context).
 *
 * Saving: "Save to file" serializes the CURRENT blocks back into the mirror
 * content shape and calls App.SaveDocumentMirror — the Go bridge verifies
 * which blocks actually changed (hashes from meta.json) and writes the
 * source .docx/.txt/.md. Nothing else ever writes to the source file.
 */

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useApp } from "../../state/appState";
import { useAIRunners, type SelectionRef } from "../../state/aiController";
import { saveDocumentMirror, updateMirrorContent } from "../../services/backend";
import type { Document, DocumentBlock, Highlight, ID, Segment } from "../../types/domain";
import {
  HIGHLIGHT_CLASS,
  HIGHLIGHT_COLORS,
  HIGHLIGHT_MEANING,
  HIGHLIGHT_SWATCH,
  PAGE_FIT_SCALE,
  PX_PER_PT,
  type HighlightColor,
  type MirrorBlock,
  type MirrorContent,
} from "../../types/mirror";
import { Icon } from "../../components/icons";
import { uid } from "../../utils/helpers";
import { cn } from "../../utils/cn";

/* ------------------------------------------------------------------ */
/* Pagination (shared with the summary editor)                         */
/* ------------------------------------------------------------------ */

/** Fallback virtual page budget — mirrors extract.SynthesizedPageSize (Go). */
export const PAGE_BUDGET = 2000;

function blockChars(block: DocumentBlock): number {
  if (block.type === "list" && block.listItems?.length) {
    return block.listItems.reduce((n, item) => n + item.reduce((m, s) => m + s.text.length, 0), 0);
  }
  return (block.segments ?? []).reduce((n, s) => n + s.text.length, 0);
}

/**
 * Greedy pagination over canonical blocks: fill a page until the char
 * budget (from meta.charsPerPage when we have it), then start a new one.
 * `page`-type blocks force a break (real PDF pagination); headings move to
 * a fresh page rather than dangle at the bottom. Pure function — blocks
 * keep their ids, so highlights/selection work across pages.
 */
export function paginateBlocks(blocks: DocumentBlock[], budget = PAGE_BUDGET): DocumentBlock[][] {
  const pages: DocumentBlock[][] = [];
  let cur: DocumentBlock[] = [];
  let used = 0;
  const flush = () => {
    if (cur.length > 0) pages.push(cur);
    cur = [];
    used = 0;
  };
  for (const b of blocks) {
    if (b.type === "page") {
      flush();
      continue;
    }
    const cost = blockChars(b);
    if (cur.length > 0 && (used + cost > budget || (b.type === "heading" && used > budget / 2))) {
      flush();
    }
    cur.push(b);
    used += cost;
  }
  flush();
  // Empty document = one empty page (new files stay openable + editable).
  if (pages.length === 0) pages.push([]);
  return pages;
}

/* ------------------------------------------------------------------ */
/* Text helpers                                                        */
/* ------------------------------------------------------------------ */

/** Plain text of one block (list items joined with newlines). */
export function blockPlainText(b: DocumentBlock): string {
  if (b.type === "list" && b.listItems?.length) {
    return b.listItems.map((it) => it.map((s) => s.text).join("")).join("\n");
  }
  return (b.segments ?? []).map((s) => s.text).join("");
}

/** One render run: a span of chars sharing style + highlight. */
interface Run {
  text: string;
  em?: boolean;
  strong?: boolean;
  hl?: Highlight;
}

/**
 * Split a block's segments into runs decorated with its highlights.
 * Ranged highlights mark exactly their span; a range-less highlight tints
 * the whole block. Overlaps: first highlight wins (kept simple + stable).
 */
function runsForBlock(block: DocumentBlock, highlights: Highlight[]): Run[] {
  const segments = block.segments ?? [];
  const mine = highlights.filter((h) => h.blockId === block.id);
  if (mine.length === 0) {
    return segments.map((s) => ({ text: s.text, em: s.em, strong: s.strong }));
  }

  // Char-level owner map: which highlight (if any) covers each index.
  const full = segments.map((s) => s.text).join("");
  const owner: (Highlight | undefined)[] = new Array(full.length).fill(undefined);
  for (const h of mine) {
    if (h.range && h.range.end > h.range.start && h.range.end <= full.length) {
      for (let i = h.range.start; i < h.range.end; i++) {
        if (!owner[i]) owner[i] = h;
      }
    } else if (!h.range) {
      // Block-anchored highlight: tint everything not already claimed.
      for (let i = 0; i < full.length; i++) {
        if (!owner[i]) owner[i] = h;
      }
    }
  }

  // Walk segments + owners together, grouping consecutive chars that share
  // (em, strong, highlight id) into runs.
  const runs: Run[] = [];
  let idx = 0;
  for (const seg of segments) {
    let i = 0;
    while (i < seg.text.length) {
      const hl = owner[idx + i];
      let j = i + 1;
      while (
        j < seg.text.length &&
        owner[idx + j] === hl
      ) {
        j++;
      }
      runs.push({ text: seg.text.slice(i, j), em: seg.em, strong: seg.strong, hl });
      i = j;
    }
    idx += seg.text.length;
  }
  return runs;
}

/* ------------------------------------------------------------------ */
/* Highlight right-click menu                                          */
/* ------------------------------------------------------------------ */

function HighlightMenu({
  doc,
  highlight,
  x,
  y,
  onClose,
}: {
  doc: Document;
  highlight: Highlight;
  x: number;
  y: number;
  onClose: () => void;
}) {
  const { dispatch } = useApp();
  const { runExplain } = useAIRunners();
  const [askOpen, setAskOpen] = useState(false);
  const [draft, setDraft] = useState("");

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [onClose]);

  const askAI = () => {
    // The prompt carries the highlighted text as the selection; the harness
    // attaches this document's context pack, so the AI knows which file and
    // page it is looking at.
    const ref: SelectionRef = {
      documentId: doc.id,
      blockId: highlight.blockId,
      text: highlight.text,
      range: highlight.range,
    };
    runExplain("AI_EXPLAIN", ref, draft.trim() || undefined);
    onClose();
  };

  return (
    <div
      data-highlight-menu
      className="fade-in fixed z-50 w-[260px] rounded-xl border border-line bg-white p-2 shadow-xl shadow-ink-900/10"
      style={{ left: x, top: y }}
      onMouseDown={(e) => e.stopPropagation()}
    >
      <p className="mb-1.5 max-h-[70px] overflow-y-auto rounded-md bg-cream-50 px-2 py-1 font-serif text-[12px] leading-relaxed text-ink-700">
        “{highlight.text.slice(0, 180)}”
      </p>
      <div className="mb-1.5 flex items-center gap-1">
        <span className="mr-1 text-[10.5px] uppercase tracking-wide text-mute">Color</span>
        {HIGHLIGHT_COLORS.map((c) => (
          <button
            key={c}
            type="button"
            title={HIGHLIGHT_MEANING[c]}
            className={cn(
              "h-4 w-4 rounded-full border transition-transform hover:scale-110",
              HIGHLIGHT_SWATCH[c],
              highlight.color === c ? "border-ink-700" : "border-line",
            )}
            onClick={() => {
              dispatch({ type: "doc.highlight.update", documentId: doc.id, highlightId: highlight.id, patch: { color: c } });
              onClose();
            }}
          />
        ))}
      </div>
      <div className="flex flex-col gap-1">
        <button
          type="button"
          className="rounded-lg px-2 py-1.5 text-left text-[12px] font-medium text-ink-700 hover:bg-cream-100"
          onClick={() => {
            // "Save permanently": keep the highlight, clear the pending flag
            // (AI-made yellow marks become a normal, kept highlight).
            dispatch({
              type: "doc.highlight.update",
              documentId: doc.id,
              highlightId: highlight.id,
              patch: { pending: false, color: highlight.color ?? ("green" as HighlightColor) },
            });
            dispatch({ type: "toast", message: "Highlight kept permanently" });
            onClose();
          }}
        >
          Save permanently
        </button>
        <button
          type="button"
          className="rounded-lg px-2 py-1.5 text-left text-[12px] font-medium text-ink-700 hover:bg-cream-100"
          onClick={() => setAskOpen((v) => !v)}
        >
          Ask the AI about this…
        </button>
        <button
          type="button"
          className="rounded-lg px-2 py-1.5 text-left text-[12px] font-medium text-rose-600 hover:bg-rose-50"
          onClick={() => {
            dispatch({ type: "doc.highlight.remove", documentId: doc.id, highlightId: highlight.id });
            onClose();
          }}
        >
          Remove highlight
        </button>
      </div>
      {askOpen && (
        <div className="mt-1.5 flex flex-col gap-1.5 border-t border-line-soft pt-1.5">
          <textarea
            autoFocus
            rows={2}
            value={draft}
            onChange={(e) => setDraft(e.target.value)}
            placeholder="Custom prompt — the highlighted text + this file's context are attached…"
            className="w-full resize-none rounded-lg border border-line bg-cream-50 px-2 py-1.5 text-[12px] text-ink-900 placeholder:text-mute focus:border-iris-400 focus:outline-none"
            onKeyDown={(e) => {
              if (e.key === "Enter" && !e.shiftKey) {
                e.preventDefault();
                askAI();
              }
            }}
          />
          <div className="flex justify-end gap-1">
            <button type="button" className="rounded-lg px-2 py-1 text-[11px] text-mute hover:bg-cream-100" onClick={() => setAskOpen(false)}>
              Cancel
            </button>
            <button type="button" className="rounded-lg bg-iris-600 px-2.5 py-1 text-[11px] font-medium text-white hover:bg-iris-700" onClick={askAI}>
              Send to AI
            </button>
          </div>
        </div>
      )}
    </div>
  );
}

/* ------------------------------------------------------------------ */
/* One block (read-only or editable)                                   */
/* ------------------------------------------------------------------ */

function CanvasBlock({
  block,
  doc,
  editable,
  onCommitText,
  onCommitList,
  onSplit,
  onHighlightMenu,
}: {
  block: DocumentBlock;
  doc: Document;
  editable: boolean;
  onCommitText: (blockId: ID, text: string) => void;
  onCommitList: (blockId: ID, rows: string[]) => void;
  onSplit: (blockId: ID, before: string, after: string) => void;
  onHighlightMenu: (e: React.MouseEvent, hl: Highlight) => void;
}) {
  const elRef = useRef<HTMLElement | null>(null);
  const setRef = useCallback((el: HTMLDivElement | HTMLHeadingElement | HTMLParagraphElement | null) => {
    elRef.current = el;
  }, []);

  // Render the styled + highlighted runs. Marks are <mark> elements with a
  // right-click handler; inside contentEditable they are inert for typing
  // (commit reads innerText, which ignores markup).
  const renderRuns = (segments: Segment[], highlights: Highlight[]) => {
    const runs = runsForBlock({ ...block, segments }, highlights);
    return runs.map((r, i) =>
      r.hl ? (
        <mark
          key={i}
          data-highlight-id={r.hl.id}
          title={`Highlight: ${r.hl.note ?? HIGHLIGHT_MEANING[r.hl.color ?? "yellow"]} — right-click for options`}
          className={cn(
            "cursor-pointer rounded-[3px] px-0.5",
            HIGHLIGHT_CLASS[r.hl.color ?? "yellow"],
            r.hl.pending && "ring-1 ring-amber-400",
          )}
          onContextMenu={(e) => {
            e.preventDefault();
            e.stopPropagation();
            onHighlightMenu(e, r.hl!);
          }}
        >
          {r.text}
        </mark>
      ) : (
        <span key={i} className={cn(r.em && "italic", r.strong && "font-semibold")}>
          {r.text}
        </span>
      ),
    );
  };

  const highlights = doc.highlights;

  // ---- editing mechanics (only when editable) ----
  const commit = () => {
    const el = elRef.current;
    if (!el) return;
    if (block.type === "list") {
      const rows = el.innerText.replace(/\n+$/, "").split("\n");
      const cur = (block.listItems ?? []).map((it) => it.map((s) => s.text).join(""));
      if (rows.length === cur.length && rows.every((r, i) => r === cur[i])) return;
      onCommitList(block.id, rows);
      return;
    }
    const text = el.innerText.replace(/\n+$/, "");
    if (text === blockPlainText(block)) return;
    onCommitText(block.id, text);
  };

  // Enter splits the block (Word-like); Shift+Enter stays native.
  const onKeyDown = (e: React.KeyboardEvent) => {
    if (e.key !== "Enter" || e.shiftKey) return;
    const el = elRef.current;
    const sel = window.getSelection();
    if (!el || !sel || sel.rangeCount === 0) return;
    e.preventDefault();
    const range = sel.getRangeAt(0);
    const pre = range.cloneRange();
    pre.selectNodeContents(el);
    try {
      pre.setEnd(range.startContainer, range.startOffset);
    } catch {
      return;
    }
    const offset = pre.toString().length;
    const full = el.innerText.replace(/\n+$/, "");
    onSplit(block.id, full.slice(0, offset), full.slice(offset));
  };

  const editableProps = editable
    ? ({
        ref: setRef,
        // plaintext-only: rich paste lands as plain text; single newlines
        // render via pre-wrap, blank-line runs split into blocks on commit.
        contentEditable: "plaintext-only",
        suppressContentEditableWarning: true,
        spellCheck: false,
        onBlur: commit,
        onKeyDown,
      } as const)
    : {};

  if (block.type === "heading") {
    const level = block.level ?? 2;
    const cls = cn(
      "font-serif font-bold leading-tight text-ink-900 whitespace-pre-wrap",
      level === 1 ? "mb-5 mt-2 text-[24px]" : "mb-3 mt-6 text-[18px]",
      editable && "outline-none focus:bg-iris-50/40",
    );
    const Tag = (level === 1 ? "h1" : "h2") as "h1" | "h2";
    return (
      <Tag data-block-id={block.id} className={cls} {...editableProps}>
        {renderRuns(block.segments ?? [], highlights)}
      </Tag>
    );
  }

  if (block.type === "list") {
    const items = block.listItems ?? [];
    if (editable) {
      // Editable list: one line per bullet, plain text (same commit path).
      return (
        <div data-block-id={block.id} className="my-3 whitespace-pre-wrap outline-none doc-body" {...editableProps}>
          {items.map((item, i) => (
            <div key={i} className="my-1 flex gap-2">
              <span className="select-none text-mute">•</span>
              <span className="flex-1">{item.map((s, j) => <span key={j}>{s.text}</span>)}</span>
            </div>
          ))}
        </div>
      );
    }
    return (
      <ul data-block-id={block.id} className="my-3 list-disc space-y-1.5 pl-6 doc-body">
        {items.map((item, i) => (
          <li key={i}>{item.map((s, j) => <span key={j} className={cn(s.em && "italic", s.strong && "font-semibold")}>{s.text}</span>)}</li>
        ))}
      </ul>
    );
  }

  return (
    <p
      data-block-id={block.id}
      id={`blk-${block.id}`}
      className={cn("my-3.5 whitespace-pre-wrap doc-body", editable && "outline-none focus:bg-iris-50/40")}
      {...editableProps}
    >
      {renderRuns(block.segments ?? [], highlights)}
    </p>
  );
}

/* ------------------------------------------------------------------ */
/* The canvas                                                          */
/* ------------------------------------------------------------------ */

export default function MirrorCanvas({ doc }: { doc: Document }) {
  const { state, dispatch } = useApp();
  const meta = doc.metadata.mirror;
  const rootRef = useRef<HTMLDivElement>(null);
  const focusNewRef = useRef<ID | null>(null);
  const [hlMenu, setHlMenu] = useState<{ hl: Highlight; x: number; y: number } | null>(null);

  // Editable when the mirror says so (docx/txt/md with a real file path).
  // Without a mirror (mock/browser mode) fall back to the format.
  const editable =
    !!doc.metadata.path &&
    (meta ? meta.editable : doc.metadata.format === "docx" || doc.metadata.format === "text");

  // Pagination follows the metadata guide (charsPerPage) when present.
  const budget = meta?.charsPerPage && meta.charsPerPage > 0 ? meta.charsPerPage : PAGE_BUDGET;
  const pages = useMemo(() => paginateBlocks(doc.blocks, budget), [doc.blocks, budget]);

  /* ---- page geometry from the metadata guide (the "mirror the look" part) ---- */
  const pageWPt = meta?.pageWidthPt ?? 612; // US Letter fallback
  const pageHPt = meta?.pageHeightPt ?? 792;
  const fontPt = meta?.defaultFont?.sizePt ?? 11;
  const fontFamily = meta?.defaultFont?.family ?? "Georgia, 'Times New Roman', serif";
  const pageStyle: React.CSSProperties = {
    width: Math.round(pageWPt * PX_PER_PT * PAGE_FIT_SCALE),
    minHeight: Math.round(pageHPt * PX_PER_PT * PAGE_FIT_SCALE),
    paddingTop: Math.round((meta?.marginsPt?.top ?? 72) * PX_PER_PT * PAGE_FIT_SCALE),
    paddingRight: Math.round((meta?.marginsPt?.right ?? 72) * PX_PER_PT * PAGE_FIT_SCALE),
    paddingBottom: Math.round((meta?.marginsPt?.bottom ?? 72) * PX_PER_PT * PAGE_FIT_SCALE),
    paddingLeft: Math.round((meta?.marginsPt?.left ?? 72) * PX_PER_PT * PAGE_FIT_SCALE),
    fontFamily,
    fontSize: `${Math.round(fontPt * PX_PER_PT * 0.92 * 10) / 10}px`,
    lineHeight: 1.65,
  };

  /* ---- editing commits (same semantics as the summary editor) ---- */
  const commitText = (blockId: ID, text: string) => {
    // Pasted/typed multi-paragraph text splits into real blocks so the
    // mirror stays one paragraph per block.
    const paras = text
      .replace(/\r\n/g, "\n")
      .split(/\n\s*\n/)
      .map((t) => t.trim())
      .filter(Boolean);
    // The empty-page placeholder ("__new") mints real blocks on first input.
    if (blockId === "__new") {
      if (paras.length === 0) return;
      dispatch({
        type: "doc.blocks.replace",
        documentId: doc.id,
        blocks: paras.map((t) => ({ id: uid("blk"), type: "paragraph" as const, segments: [{ text: t }] })),
      });
      return;
    }
    if (paras.length > 1) {
      const idx = doc.blocks.findIndex((b) => b.id === blockId);
      if (idx >= 0) {
        const cur = doc.blocks[idx];
        const made: DocumentBlock[] = paras.map((t, i) =>
          i === 0
            ? { ...cur, segments: [{ text: t }] }
            : { id: uid("blk"), type: "paragraph", segments: [{ text: t }] },
        );
        dispatch({
          type: "doc.blocks.replace",
          documentId: doc.id,
          blocks: [...doc.blocks.slice(0, idx), ...made, ...doc.blocks.slice(idx + 1)],
        });
        return;
      }
    }
    dispatch({ type: "doc.block.text", documentId: doc.id, blockId, text });
  };

  const commitList = (blockId: ID, rows: string[]) => {
    dispatch({
      type: "doc.blocks.replace",
      documentId: doc.id,
      blocks: doc.blocks.map((b) => (b.id === blockId ? { ...b, listItems: rows.map((t) => [{ text: t }]) } : b)),
    });
  };

  const splitBlock = (blockId: ID, before: string, after: string) => {
    if (blockId === "__new") {
      commitText(blockId, `${before}\n\n${after}`);
      return;
    }
    const idx = doc.blocks.findIndex((b) => b.id === blockId);
    if (idx < 0) return;
    const newId = uid("blk");
    const cur = doc.blocks[idx];
    dispatch({
      type: "doc.blocks.replace",
      documentId: doc.id,
      blocks: [
        ...doc.blocks.slice(0, idx),
        { ...cur, segments: [{ text: before }] },
        { id: newId, type: "paragraph", segments: [{ text: after }] },
        ...doc.blocks.slice(idx + 1),
      ],
    });
    focusNewRef.current = newId;
  };

  // Focus a freshly split block at its start.
  useEffect(() => {
    if (!focusNewRef.current) return;
    const id = focusNewRef.current;
    focusNewRef.current = null;
    const el = rootRef.current?.querySelector(`[data-block-id="${CSS.escape(id)}"]`) as HTMLElement | null;
    if (!el) return;
    el.focus();
    try {
      const r = document.createRange();
      r.selectNodeContents(el);
      r.collapse(true);
      const s = window.getSelection();
      s?.removeAllRanges();
      s?.addRange(r);
    } catch {
      // Focus alone is fine; caret placement is best-effort.
    }
  });

  /* ---- save bridge (mirror content → Go → source file) ---- */
  const snapshot = JSON.stringify(doc.blocks);
  const lastSaved = useRef<string>(snapshot);
  const [saveState, setSaveState] = useState<"clean" | "dirty" | "saving">("clean");
  useEffect(() => {
    setSaveState((s) => (s === "saving" ? s : snapshot === lastSaved.current ? "clean" : "dirty"));
  }, [snapshot]);

  /**
   * Serialize the CURRENT blocks + pagination back into the mirror content
   * shape (content.json). Used by both the live push (below) and Save.
   */
  const buildContent = useCallback((): MirrorContent => {
    const blocks: MirrorBlock[] = [];
    pages.forEach((page, pi) => {
      for (const b of page) {
        blocks.push({
          id: b.id,
          type: b.type,
          level: b.level,
          segments: (b.segments ?? []).map((s) => ({ text: s.text, em: s.em, strong: s.strong })),
          listItems: b.listItems?.map((row) => row.map((s) => ({ text: s.text, em: s.em, strong: s.strong }))),
          page: pi + 1,
        });
      }
    });
    return {
      version: 1,
      kind: meta?.kind ?? (doc.metadata.format === "docx" ? "docx" : "text"),
      title: doc.metadata.title,
      blocks,
    };
  }, [pages, meta?.kind, doc.metadata.format, doc.metadata.title]);

  // LIVE MIRROR: while editing, push the current content to content.json
  // (debounced ~1.2s). No source-file write happens here — but the AI's
  // read_mirror view now matches what the user sees, mid-edit.
  useEffect(() => {
    const path = doc.metadata.path;
    if (!editable || !path) return;
    const t = setTimeout(() => {
      void updateMirrorContent(path, buildContent());
    }, 1200);
    return () => clearTimeout(t);
  }, [snapshot, editable, doc.metadata.path, buildContent]);

  const save = useCallback(async () => {
    const path = doc.metadata.path;
    if (!path) {
      dispatch({ type: "toast", message: "This document has no file path — nothing to save to." });
      return;
    }
    const content = buildContent();
    setSaveState("saving");
    try {
      const report = await saveDocumentMirror(path, content);
      if (!report) {
        setSaveState("dirty");
        dispatch({ type: "toast", message: "Saving needs the desktop app — run `wails dev`." });
        return;
      }
      lastSaved.current = snapshot;
      setSaveState("clean");
      dispatch({
        type: "toast",
        message: report.written
          ? `Saved ${doc.metadata.title} — ${report.changed.length} block(s) changed, ${report.added.length} added, ${report.removed.length} removed`
          : "Already saved — nothing had changed",
      });
    } catch (e) {
      setSaveState("dirty");
      dispatch({ type: "toast", message: `Save failed: ${e instanceof Error ? e.message : String(e)}` });
    }
  }, [doc.id, doc.metadata.path, doc.metadata.title, buildContent, snapshot, dispatch]);

  // Ctrl/Cmd+S saves the visible canvas (multiple docs stay mounted hidden —
  // the offsetParent check keeps hidden tabs from stealing the shortcut).
  useEffect(() => {
    if (!editable) return;
    const onKey = (e: KeyboardEvent) => {
      if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === "s") {
        if (!rootRef.current || rootRef.current.offsetParent === null) return;
        e.preventDefault();
        void save();
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [editable, save]);

  /* ---- selection → floating toolbar (same coordinate logic as before) ---- */
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
    if (!blockEl || !blockEl.closest("[data-doc-area]") || el?.closest("[data-highlight-menu]")) {
      return;
    }
    const domRange = sel && sel.rangeCount > 0 ? sel.getRangeAt(0) : null;
    const rect = domRange?.getBoundingClientRect();
    if (!rect) return;
    // Precise offsets within the block (for range-accurate highlights).
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

  const openHighlightMenu = (e: React.MouseEvent, hl: Highlight) => {
    dispatch({ type: "selection.set", selection: null });
    setHlMenu({
      hl,
      x: Math.max(8, Math.min(e.clientX, window.innerWidth - 280)),
      y: Math.max(8, Math.min(e.clientY, window.innerHeight - 320)),
    });
  };

  // Click anywhere outside the menu closes it.
  useEffect(() => {
    if (!hlMenu) return;
    const close = (e: MouseEvent) => {
      const t = e.target as HTMLElement;
      if (!t.closest("[data-highlight-menu]") && !t.closest("[data-highlight-id]")) setHlMenu(null);
    };
    window.addEventListener("mousedown", close);
    return () => window.removeEventListener("mousedown", close);
  }, [hlMenu]);

  /* ---- render ---- */
  return (
    <div ref={rootRef} className="flex flex-col items-center pb-10" onMouseUp={onMouseUp}>
      {/* Canvas toolbar: mirror info + save state */}
      <div
        className="mb-4 flex w-full max-w-[820px] items-center justify-between gap-3 rounded-lg border border-line-soft bg-white/70 px-3 py-1.5 text-[11.5px] text-mute"
        style={{ width: pageStyle.width }}
      >
        <span className="flex items-center gap-1.5 truncate">
          <Icon name="fileText" size={12} />
          {meta
            ? `mirror canvas · ${Math.round(pageWPt)}×${Math.round(pageHPt)}pt · ${fontFamily.split(",")[0]} ${fontPt}pt · ${pages.length} page${pages.length === 1 ? "" : "s"}`
            : `canvas · ${pages.length} page${pages.length === 1 ? "" : "s"}`}
        </span>
        {editable && (
          <span className="flex shrink-0 items-center gap-2">
            <span className={cn("h-1.5 w-1.5 rounded-full", saveState === "dirty" ? "bg-amber-400" : saveState === "saving" ? "bg-sky-400" : "bg-moss-300")} />
            <span>{saveState === "dirty" ? "unsaved edits" : saveState === "saving" ? "saving…" : "saved"}</span>
            <button
              type="button"
              onClick={() => void save()}
              disabled={saveState !== "dirty"}
              className={cn(
                "rounded-md border px-2.5 py-0.5 text-[11px] font-medium transition-colors",
                saveState === "dirty"
                  ? "border-moss-300 bg-moss-50 text-moss-700 hover:bg-moss-100"
                  : "border-line-soft bg-white text-mute",
              )}
              title="Verify changed blocks and write them back to the source file (Ctrl+S)"
            >
              Save to file
            </button>
          </span>
        )}
        {!editable && <span className="shrink-0">read-only · select text to highlight</span>}
      </div>

      {/* The pages */}
      <div className="flex flex-col items-center gap-6" data-doc-area>
        {pages.map((page, i) => (
          <section
            key={i}
            aria-label={`Page ${i + 1} of ${pages.length}`}
            style={pageStyle}
            className="select-text rounded-sm border border-line-soft bg-white shadow-sm"
          >
            {page.length === 0 && editable && (
              // Empty page of a new/blank file: a placeholder block the user
              // can type into — real blocks are minted on the first commit.
              <p
                data-block-id="__new"
                data-placeholder="Start typing…"
                contentEditable="plaintext-only"
                suppressContentEditableWarning
                spellCheck={false}
                className="my-3.5 whitespace-pre-wrap outline-none focus:bg-iris-50/40 doc-body"
                onBlur={(e) => commitText("__new", e.currentTarget.innerText)}
                onKeyDown={(e) => {
                  if (e.key === "Enter") {
                    e.preventDefault();
                    commitText("__new", e.currentTarget.innerText);
                  }
                }}
              />
            )}
            {page.map((block) => (
              <CanvasBlock
                key={block.id}
                block={block}
                doc={doc}
                editable={editable}
                onCommitText={commitText}
                onCommitList={commitList}
                onSplit={splitBlock}
                onHighlightMenu={openHighlightMenu}
              />
            ))}
            <p className="mt-8 text-center text-[10.5px] text-mute/80">— {i + 1} —</p>
          </section>
        ))}
      </div>

      {/* Footer: where this mirror came from */}
      <p className="mt-8 flex items-center gap-2 border-t border-line-soft pt-4 text-[11.5px] text-mute" style={{ width: pageStyle.width }}>
        <Icon name="info" size={12} />
        {meta?.sourcePath ?? doc.metadata.path ?? "Workspace file"} · {(doc.metadata.format ?? "").toUpperCase()} ·
        mirror: content.json + meta.json
      </p>

      {hlMenu && (
        <HighlightMenu
          doc={doc}
          highlight={hlMenu.hl}
          x={hlMenu.x}
          y={hlMenu.y}
          onClose={() => setHlMenu(null)}
        />
      )}
    </div>
  );
}
