/**
 * Summary document view — direct .docx editing on a paginated canvas.
 *
 * No Slate: canonical body is DocumentBlock[], each block a plain-text
 * contentEditable committed on blur (caret-safe), Enter splits the block.
 * Published AI edits decorate matched spans with color highlights;
 * right-click a diff for Accept / Reject / instruction + Send to AI.
 * Autosave writes the .docx; the AI reads/writes it through the mirror.
 */

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useApp } from "../../state/appState";
import { useAIRunners } from "../../state/aiController";
import { writeSummaryDOCX } from "../../services/backend";
import type { Document, DocumentBlock, ID, Segment, SummaryDiff } from "../../types/domain";
import { Icon } from "../../components/icons";
import { Badge } from "../../components/ui";
import { cn } from "../../utils/cn";
import { uid } from "../../utils/helpers";
import { paginateBlocks } from "./DocumentView";

/* ------------------------------------------------------------------ */
/* DocxCanvas — Word-like direct editing (no Slate)                      */
/*                                                                     */
/* Canonical body stays DocumentBlock[]: each block is a plain-text     */
/* contentEditable committed on blur (caret-safe: typing never touches  */
/* React state), Enter splits the block, lists edit per item. Published */
/* diffs decorate matched spans with color highlights; right-click (or  */
/* click) opens the per-diff menu: Accept / Reject / instruction + Send */
/* to AI. The .docx on disk is canonical — autosave writes it, the AI   */
/* reads/writes it through the mirror, reloads re-render here.          */
/* ------------------------------------------------------------------ */

function blockPlainText(b: DocumentBlock): string {
  if (b.type === "list" && b.listItems?.length) {
    return b.listItems.map((it) => it.map((s) => s.text).join("")).join("\n");
  }
  return (b.segments ?? []).map((s) => s.text).join("");
}

/** Split committed text on blank lines (paste/type normalization). */
function splitParagraphs(text: string): string[] {
  return text
    .replace(/\r\n/g, "\n")
    .split(/\n\s*\n/)
    .map((t) => t.trim())
    .filter(Boolean);
}

/** Probe strings for a diff span, strongest first (single source of truth). */
function diffProbes(span: string): string[] {
  const t = span.trim();
  if (!t) return [];
  const probes = [t.slice(0, 120)];
  // Short-span fallback for blocks edited after publish (mirrors
  // findBlockForSpan in aiController.ts).
  if (t.length > 60) probes.push(t.slice(0, 60));
  return [...new Set(probes)];
}

/** Block index holding a diff span; -1 when the file moved on since publish. */
function findBlockIndexForDiff(blocks: DocumentBlock[], diff: SummaryDiff): number {
  const span = (diff.op === "delete" ? (diff.oldText ?? diff.text) : diff.text).trim();
  if (!span) return -1;
  // Deletes have no live span by definition — never match them to a block.
  if (diff.op === "delete") return -1;
  return blocks.findIndex((b) => {
    const full = blockPlainText(b);
    return diffProbes(span).some((p) => p && full.includes(p));
  });
}

/** Split segments into before/mark/after runs for a diff span (marks preserved). */
function diffRuns(
  segments: Segment[],
  span: string,
): { before: Segment[]; mark: string; after: Segment[] } | null {
  const full = segments.map((s) => s.text).join("");
  const t = span.trim();
  if (!t) return null;
  // Prefer the full span so long inserts mark wholly; fall back to the
  // truncated probes when the block was edited after publish.
  let at = full.indexOf(t);
  let len = t.length;
  if (at < 0) {
    const probe = t.slice(0, 120);
    at = full.indexOf(probe);
    len = Math.min(t.length, 120);
  }
  if (at < 0) return null;
  const end = at + len;
  const before: Segment[] = [];
  const after: Segment[] = [];
  let mark = "";
  let acc = 0;
  for (const s of segments) {
    const sEnd = acc + s.text.length;
    if (sEnd <= at) before.push(s);
    else if (acc >= end) after.push(s);
    else {
      const lo = Math.max(0, at - acc);
      const hi = Math.min(s.text.length, end - acc);
      if (lo > 0) before.push({ ...s, text: s.text.slice(0, lo) });
      mark += s.text.slice(lo, hi);
      if (hi < s.text.length) after.push({ ...s, text: s.text.slice(hi) });
    }
    acc = sEnd;
  }
  if (!mark) return null;
  return { before, after, mark };
}

const DIFF_MARK_CLASS: Record<SummaryDiff["op"], string> = {
  insert: "cursor-pointer rounded-[3px] bg-moss-200/70 px-0.5 text-ink-900",
  modify: "cursor-pointer rounded-[3px] bg-hay-100 px-0.5 underline decoration-hay-400 decoration-2 underline-offset-4",
  delete: "",
};

function MarkedText({
  segments,
  diff,
  onOpenMenu,
}: {
  segments: Segment[];
  diff: SummaryDiff | undefined;
  onOpenMenu: (e: React.MouseEvent, diff: SummaryDiff) => void;
}) {
  if (!diff || !diff.text.trim()) {
    return (
      <>
        {segments.map((s, i) => (
          <span key={i} className={cn(s.em && "italic", s.strong && "font-semibold")}>
            {s.text}
          </span>
        ))}
      </>
    );
  }
  const runs = diffRuns(segments, diff.text);
  if (!runs) {
    return (
      <>
        {segments.map((s, i) => (
          <span key={i} className={cn(s.em && "italic", s.strong && "font-semibold")}>
            {s.text}
          </span>
        ))}
      </>
    );
  }
  const open = (e: React.MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();
    onOpenMenu(e, diff);
  };
  return (
    <>
      {runs.before.map((s, i) => (
        <span key={`b${i}`} className={cn(s.em && "italic", s.strong && "font-semibold")}>
          {s.text}
        </span>
      ))}
      <mark
        key="mark"
        data-diff-id={diff.id}
        title="Published AI edit — right-click to accept, reject, or instruct"
        className={DIFF_MARK_CLASS[diff.op]}
        onContextMenu={open}
        onClick={open}
      >
        {runs.mark}
      </mark>
      {runs.after.map((s, i) => (
        <span key={`a${i}`} className={cn(s.em && "italic", s.strong && "font-semibold")}>
          {s.text}
        </span>
      ))}
    </>
  );
}

function EditableBlock({
  block,
  diff,
  onCommitText,
  onCommitList,
  onSplit,
  onOpenMenu,
}: {
  block: DocumentBlock;
  diff: SummaryDiff | undefined;
  onCommitText: (blockId: ID, text: string) => void;
  onCommitList: (blockId: ID, rows: string[]) => void;
  onSplit: (blockId: ID, before: string, after: string) => void;
  onOpenMenu: (e: React.MouseEvent, diff: SummaryDiff) => void;
}) {
  const elRef = useRef<HTMLElement | null>(null);
  const setRef = useCallback(
    (el: HTMLDivElement | HTMLHeadingElement | HTMLParagraphElement | null) => {
      elRef.current = el;
    },
    [],
  );

  // Commit on blur only — typing never touches React state, so the caret
  // is never yanked by a re-render.
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

  // Enter splits the block (Word-like): text before the caret stays, the
  // rest becomes a new paragraph below. Shift+Enter keeps native behavior.
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
    // Focus lands in the new block after React commits (see DocxCanvas).
  };

  const editable = {
    ref: setRef,
    // plaintext-only: rich paste lands as plain text (Chromium/WebView2);
    // single newlines render via pre-wrap, blank-line runs split on commit.
    contentEditable: "plaintext-only",
    suppressContentEditableWarning: true,
    spellCheck: false,
    onBlur: commit,
    onKeyDown,
  } as const;

  if (block.type === "heading") {
    const level = block.level ?? 2;
    const cls =
      level === 1
        ? "mb-7 font-serif text-[26px] font-bold leading-tight text-ink-900 outline-none whitespace-pre-wrap"
        : "mt-8 font-serif text-[19px] font-bold text-ink-900 outline-none whitespace-pre-wrap";
    const Tag = level === 1 ? "h1" : "h2";
    return (
      <Tag data-block-id={block.id} className={cls} {...editable}>
        <MarkedText segments={block.segments ?? []} diff={diff} onOpenMenu={onOpenMenu} />
      </Tag>
    );
  }

  if (block.type === "list") {
    const items = block.listItems ?? [];
    // Diff may target one row: decorate that row only.
    const diffRow = diff?.text.trim()
      ? items.findIndex((it) => {
          const row = it.map((s) => s.text).join("");
          return diffProbes(diff.text).some((p) => p && row.includes(p));
        })
      : -1;
    return (
      <div data-block-id={block.id} className="outline-none doc-body whitespace-pre-wrap" {...editable}>
        {items.map((item, i) => (
          <div key={i} className="my-1 flex gap-2">
            <span className="select-none text-mute">•</span>
            <span className="flex-1">
              <MarkedText
                segments={item}
                diff={i === diffRow ? diff : undefined}
                onOpenMenu={onOpenMenu}
              />
            </span>
          </div>
        ))}
      </div>
    );
  }

  return (
    <p data-block-id={block.id} className="my-4 doc-body outline-none whitespace-pre-wrap" {...editable}>
      <MarkedText segments={block.segments ?? []} diff={diff} onOpenMenu={onOpenMenu} />
    </p>
  );
}

/* ------------------------------------------------------------------ */
/* DocxCanvas — paginated direct editing + right-click diff menu        */
/* ------------------------------------------------------------------ */

function DocxCanvas({ doc }: { doc: Document }) {
  const { dispatch } = useApp();
  const { runChat } = useAIRunners();
  const [menu, setMenu] = useState<{ diff: SummaryDiff; x: number; y: number } | null>(null);
  const [menuDraft, setMenuDraft] = useState("");
  const focusNewRef = useRef<ID | null>(null);

  // One diff per block (first match wins): keeps decorations unambiguous.
  const diffByBlock = useMemo(() => {
    const m = new Map<ID, SummaryDiff>();
    for (const d of doc.diffs ?? []) {
      if (d.op === "delete" || !d.text.trim()) continue;
      const idx = findBlockIndexForDiff(doc.blocks, d);
      if (idx >= 0) {
        const blk = doc.blocks[idx];
        if (blk && !m.has(blk.id)) m.set(blk.id, d);
      }
    }
    return m;
  }, [doc.diffs, doc.blocks]);

  const openMenu = useCallback((e: React.MouseEvent, diff: SummaryDiff) => {
    setMenuDraft("");
    setMenu({
      diff,
      x: Math.max(8, Math.min(e.clientX, window.innerWidth - 280)),
      y: Math.max(8, Math.min(e.clientY, window.innerHeight - 340)),
    });
  }, []);

  useEffect(() => {
    if (!menu) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") setMenu(null);
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [menu]);

  // Focus a freshly split block at its start.
  useEffect(() => {
    if (!focusNewRef.current) return;
    const id = focusNewRef.current;
    focusNewRef.current = null;
    const el = document.querySelector(`[data-block-id="${id}"]`) as HTMLElement | null;
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
      // Focus is enough; caret placement is best-effort.
    }
  });

  const commitText = (blockId: ID, text: string) => {
    // The empty-document placeholder mints a real block on first input.
    if (blockId === "__new") {
      if (!text.trim()) return;
      dispatch({
        type: "doc.blocks.replace",
        documentId: doc.id,
        blocks: splitParagraphs(text).map((t) => ({ id: uid("blk"), type: "paragraph" as const, segments: [{ text: t }] })),
      });
      return;
    }
    // Buffer/line normalization (block-tree housekeeping): pasted or typed
    // multi-paragraph text splits into real blocks so the model stays one
    // paragraph per block (rich paste lands as plain paragraphs).
    const paras = splitParagraphs(text);
    if (paras.length > 1) {
      const idx = doc.blocks.findIndex((b) => b.id === blockId);
      if (idx >= 0) {
        const cur = doc.blocks[idx];
        const made = paras.map((t, i) =>
          i === 0
            ? { ...cur, segments: [{ text: t }] }
            : { id: uid("blk"), type: "paragraph" as const, segments: [{ text: t }] },
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
      blocks: doc.blocks.map((b) =>
        b.id === blockId ? { ...b, listItems: rows.map((t) => [{ text: t }]) } : b,
      ),
    });
  };

  const splitBlock = (blockId: ID, before: string, after: string) => {
    if (blockId === "__new") {
      commitText(blockId, `${before}\n${after}`.trim());
      return;
    }
    const idx = doc.blocks.findIndex((b) => b.id === blockId);
    if (idx < 0) return;
    const newId = uid("blk");
    const cur = doc.blocks[idx];
    const next = [
      ...doc.blocks.slice(0, idx),
      { ...cur, segments: [{ text: before }] },
      { id: newId, type: "paragraph" as const, segments: [{ text: after }] },
      ...doc.blocks.slice(idx + 1),
    ];
    dispatch({ type: "doc.blocks.replace", documentId: doc.id, blocks: next });
    focusNewRef.current = newId;
  };

  const acceptDiff = (diff: SummaryDiff) => {
    // Already written by publish — accept clears the decoration.
    dispatch({ type: "doc.diffs.dismiss", documentId: doc.id, diffId: diff.id });
    setMenu(null);
    dispatch({ type: "toast", message: "Diff accepted" });
  };

  const rejectDiff = (diff: SummaryDiff) => {
    if (diff.op !== "delete") {
      const span = diff.text.trim();
      const replacement = diff.op === "modify" ? (diff.oldText ?? "") : "";
      const idx = findBlockIndexForDiff(doc.blocks, diff);
      if (idx < 0) {
        dispatch({ type: "toast", message: "Span not found — the file changed since publish." });
        return;
      }
      const full = blockPlainText(doc.blocks[idx]);
      const matched = diffProbes(span).find((p) => p && full.includes(p)) ?? "";
      if (!matched) {
        dispatch({ type: "toast", message: "Span not found — the file changed since publish." });
        return;
      }
      const cut = full.replace(matched, replacement);
      // Rejecting an insert that leaves an empty block removes the block
      // (mirrors card insert-reject); modify keeps the block with pre-image.
      const next =
        diff.op === "insert" && !cut.trim()
          ? doc.blocks.filter((_, i) => i !== idx)
          : doc.blocks.map((b, i) => (i !== idx ? b : { ...b, segments: [{ text: cut }] }));
      dispatch({ type: "doc.blocks.replace", documentId: doc.id, blocks: next });
    } else {
      const restored = (diff.oldText ?? diff.text).trim();
      if (restored) {
        dispatch({
          type: "doc.blocks.replace",
          documentId: doc.id,
          blocks: [...doc.blocks, { id: uid("blk"), type: "paragraph", segments: [{ text: restored }] }],
        });
      }
    }
    // Autosave writes the .docx; the mirror re-syncs from the new snapshot.
    dispatch({ type: "doc.diffs.dismiss", documentId: doc.id, diffId: diff.id });
    setMenu(null);
    dispatch({ type: "toast", message: "Diff rejected — pre-image restored" });
  };

  const sendDiffToAI = (diff: SummaryDiff, instruction: string) => {
    const span = (diff.op === "delete" ? (diff.oldText ?? diff.text) : diff.text).trim();
    // Exact block anchor (same finder as decoration/reject): runChat scopes
    // the context pack window + anchored region to it. Deletes have no live
    // span, so they send text-only context (title + outline + snapshot).
    const idx = findBlockIndexForDiff(doc.blocks, diff);
    const blk = idx >= 0 ? doc.blocks[idx] : undefined;
    void runChat(instruction || `Improve this passage: ${span.slice(0, 200)}`, {
      documentId: doc.id,
      selection: { documentId: doc.id, blockId: blk?.id ?? "", text: span.slice(0, 2000) },
    });
    setMenu(null);
    dispatch({ type: "toast", message: "Sent to AI — watch the chat for the revision" });
  };

  const pages = doc.blocks.length > 0 ? paginateBlocks(doc.blocks) : [];

  return (
    <div>
      {doc.blocks.length === 0 ? (
        <section
          aria-label="Page 1 of 1"
          className="rounded-sm border border-line-soft bg-white px-8 py-6 shadow-sm"
          onContextMenu={(e) => e.preventDefault()}
        >
          <EditableBlock
            block={{ id: "__new", type: "paragraph", segments: [{ text: "" }] }}
            diff={undefined}
            onCommitText={commitText}
            onCommitList={commitList}
            onSplit={splitBlock}
            onOpenMenu={openMenu}
          />
          <p className="mt-6 text-center text-[10.5px] text-mute">— 1 —</p>
        </section>
      ) : (
        <div className="space-y-6">
          {pages.map((page, i) => (
            <section
              key={i}
              aria-label={`Page ${i + 1} of ${pages.length}`}
              className="rounded-sm border border-line-soft bg-white px-8 py-6 shadow-sm"
              // Wails shows the webview menu on right-click — suppress it on
              // pages so diff marks own the gesture (menu marks stop
              // propagation; the textarea menu is outside these sections).
              onContextMenu={(e) => e.preventDefault()}
            >
              {page.map((block) => (
                <EditableBlock
                  key={block.id}
                  block={block}
                  diff={diffByBlock.get(block.id)}
                  onCommitText={commitText}
                  onCommitList={commitList}
                  onSplit={splitBlock}
                  onOpenMenu={openMenu}
                />
              ))}
              <p className="mt-6 text-center text-[10.5px] text-mute">— {i + 1} —</p>
            </section>
          ))}
        </div>
      )}

      {menu && (
        <>
          <div
            className="fixed inset-0 z-40"
            onClick={() => setMenu(null)}
            onContextMenu={(e) => {
              e.preventDefault();
              setMenu(null);
            }}
          />
          <div
            className="fixed z-50 w-[260px] rounded-lg border border-line bg-white p-3 shadow-lg"
            style={{ left: menu.x, top: menu.y }}
            onClick={(e) => e.stopPropagation()}
          >
            <p className="mb-1 flex items-center gap-1.5 text-[11px] font-semibold uppercase tracking-wide text-sky-700">
              <Icon name="sparkles" size={12} /> Published {menu.diff.op}
              <button
                type="button"
                className="ml-auto font-normal normal-case tracking-normal text-mute hover:text-ink-700"
                onClick={() => setMenu(null)}
              >
                dismiss
              </button>
            </p>
            <p className="max-h-[120px] overflow-y-auto font-serif text-[12.5px] leading-relaxed text-ink-700">
              {menu.diff.text.slice(0, 280)}
            </p>
            {menu.diff.note && <p className="mt-1 text-[11px] text-mute">{menu.diff.note}</p>}
            <div className="mt-2 flex gap-1.5">
              <button
                type="button"
                onClick={() => acceptDiff(menu.diff)}
                className="rounded-md bg-moss-600 px-2.5 py-1 text-[11.5px] font-medium text-white hover:bg-moss-700"
              >
                Accept
              </button>
              <button
                type="button"
                onClick={() => rejectDiff(menu.diff)}
                className="rounded-md border border-line bg-white px-2.5 py-1 text-[11.5px] font-medium text-ink-700 hover:border-rose-300"
              >
                Reject
              </button>
            </div>
            <div className="mt-2 flex items-end gap-1.5">
              <textarea
                value={menuDraft}
                onChange={(e) => setMenuDraft(e.target.value)}
                rows={2}
                placeholder="Custom instruction, e.g. shorten but keep numbers…"
                className="min-h-[40px] flex-1 resize-none rounded-lg border border-line bg-white px-2.5 py-1.5 text-[12px] text-ink-800 outline-none focus:border-iris-300"
              />
              <button
                type="button"
                onClick={() => {
                  sendDiffToAI(menu.diff, menuDraft.trim());
                  setMenuDraft("");
                }}
                className="rounded-md border border-iris-300 bg-iris-50 px-2.5 py-1.5 text-[11.5px] font-medium text-iris-700 hover:bg-iris-100"
              >
                Send to AI
              </button>
            </div>
          </div>
        </>
      )}
    </div>
  );
}

export default function SummaryDocumentView({ doc }: { doc: Document }) {
  const { state, dispatch } = useApp();

  // Published write-through diffs (ephemeral, over live blocks). Deletes
  // have no span left in the file — they render as cards; inserts/modifies
  // decorate inline in the canvas. Single review path: no proposal cards.
  const diffs = doc.diffs ?? [];
  const diffDeletes = diffs.filter((d) => d.op === "delete");

  const saveDocx = async () => {
    // Save back to the open file itself (folder/summary.docx stays
    // folder/summary.docx). Only pathless summaries fall back to the
    // auto-named summaries/ copy.
    const blocksJSON = JSON.stringify({ blocks: doc.blocks });
    try {
      const path = await writeSummaryDOCX(blocksJSON, doc.metadata.path ?? "");
      if (path) {
        dispatch({ type: "toast", message: `Saved summary DOCX · ${path}` });
      } else {
        dispatch({ type: "toast", message: "DOCX save needs the desktop app (Wails)." });
      }
    } catch (e) {
      dispatch({
        type: "toast",
        message: `DOCX save failed: ${e instanceof Error ? e.message : String(e)}`,
      });
    }
  };

  // Gated autosave (issues 16+29): debounced 3s after blocks settle, written
  // back to the open file itself (metadata.path) — never a summaries/ copy.
  // Silent on success; honest toast only on failure. No-op in browser mode.
  const lastSaved = useRef<string>("");
  const [autoState, setAutoState] = useState<"idle" | "saving" | "saved">("idle");
  const savePath = doc.metadata.path ?? "";
  useEffect(() => {
    const committed = doc.blocks.filter((b) => {
      const ch = state.changes[b.changeId ?? ""];
      return !(ch && ch.type === "insert" && ch.status === "pending");
    });
    const snapshot = JSON.stringify({ blocks: committed });
    if (snapshot === lastSaved.current) return;
    const t = setTimeout(() => {
      lastSaved.current = snapshot;
      setAutoState("saving");
      writeSummaryDOCX(snapshot, savePath)
        .then((path) => setAutoState(path ? "saved" : "idle"))
        .catch((e: unknown) => {
          setAutoState("idle");
          dispatch({
            type: "toast",
            message: `Autosave failed: ${e instanceof Error ? e.message : String(e)}`,
          });
        });
    }, 3000);
    return () => clearTimeout(t);
  }, [doc.blocks, state.changes, savePath, dispatch]);

  return (
    <div className="mx-auto max-w-[700px] px-8 py-6">
      <div className="mb-4 flex items-center justify-between border-b border-line pb-2.5 text-[12px] text-mute">
        <span className="flex items-center gap-3">
          <span>Editable summary · {doc.sourceIds?.length ?? 0} sources</span>
          <button
            type="button"
            onClick={() => void saveDocx()}
            className="rounded-md border border-line bg-white px-2 py-0.5 text-[11px] font-medium text-ink-700 hover:border-moss-300 hover:text-moss-700"
            title="Write summary as .docx under the workspace (summaries/)"
          >
            Save DOCX
          </button>
        </span>
        <span className="flex items-center gap-1.5">
          <span className={cn("h-1.5 w-1.5 rounded-full", diffs.length ? "bg-sky-400" : "bg-moss-200")} />
          {diffs.length
            ? `${diffs.length} diff${diffs.length === 1 ? "" : "s"} to review`
            : autoState === "saving"
              ? "saving…"
              : autoState === "saved"
                ? "all changes reviewed · saved"
                : "all changes reviewed"}
        </span>
      </div>

      {/* Published delete diffs: the span is gone from the file, so they
          render as cards (accept dismisses; reject re-inserts the pre-image). */}
      {diffDeletes.map((diff) => (
        <div key={`diffdel-${diff.id}`} className="fade-in mb-2 rounded-lg border border-sky-200 bg-sky-50/60 px-4 py-3">
          <div className="mb-1 flex items-center gap-2 text-[11px] text-sky-700">
            <Icon name="sparkles" size={12} />
            <span className="font-semibold uppercase tracking-wide">Published deletion</span>
            <Badge tone="blue">in-file</Badge>
            <span className="ml-auto flex gap-1.5">
              <button
                type="button"
                onClick={() => dispatch({ type: "doc.diffs.dismiss", documentId: doc.id, diffId: diff.id })}
                className="rounded-md bg-moss-600 px-2 py-0.5 text-[11px] font-medium text-white hover:bg-moss-700"
              >
                Accept
              </button>
              <button
                type="button"
                onClick={() => {
                  const restored = (diff.oldText ?? diff.text).trim();
                  if (restored) {
                    dispatch({
                      type: "doc.blocks.replace",
                      documentId: doc.id,
                      blocks: [...doc.blocks, { id: uid("blk"), type: "paragraph", segments: [{ text: restored }] }],
                    });
                  }
                  dispatch({ type: "doc.diffs.dismiss", documentId: doc.id, diffId: diff.id });
                  dispatch({ type: "toast", message: "Diff rejected — pre-image restored" });
                }}
                className="rounded-md border border-line bg-white px-2 py-0.5 text-[11px] font-medium text-ink-700 hover:border-rose-300"
              >
                Reject
              </button>
            </span>
          </div>
          <p className="font-serif text-[14px] leading-[1.7] text-ink-500 line-through decoration-rose-300">
            {(diff.oldText ?? diff.text).slice(0, 280) || "(empty block)"}
          </p>
        </div>
      ))}

      <DocxCanvas doc={doc} />

      {diffs.length === 0 && (
        <p className="mt-8 flex items-center gap-2 border-t border-line-soft pt-4 text-[11.5px] text-mute">
          <Icon name="shieldCheck" size={13} className="text-moss-600" />
          Edit directly — Enter splits a paragraph. AI edits appear highlighted in-file: right-click one to accept, reject, or instruct.
        </p>
      )}
    </div>
  );
}
