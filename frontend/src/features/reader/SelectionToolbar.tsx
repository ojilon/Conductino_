/**
 * Floating selection toolbar — the "highlight → AI / summary" hub.
 *
 * Appears whenever the user selects text inside a document view (canvas or
 * PDF page). Actions:
 *   - Ask AI / Explain / Go deeper  → AI runners with the selection as context
 *   - color dots                    → save a colored highlight (yellow is
 *                                      reserved by convention for AI changes)
 *   - To summary                    → append the quoted selection to the
 *                                      workspace summary TARGET document
 *                                      (right-click a .docx in the library →
 *                                      "Make this the target summary document")
 *
 * Lives in its own file so DocumentView, MirrorCanvas and PdfView can all
 * use it without import cycles.
 */

import { useState } from "react";
import { useApp } from "../../state/appState";
import { useAIRunners, type SelectionRef } from "../../state/aiController";
import { getSummaryTargetFile } from "../../services/backend";
import type { Document } from "../../types/domain";
import { HIGHLIGHT_COLORS, HIGHLIGHT_MEANING, HIGHLIGHT_SWATCH, type HighlightColor } from "../../types/mirror";
import { Icon } from "../../components/icons";
import { clamp, uid } from "../../utils/helpers";
import { cn } from "../../utils/cn";

export function SelectionToolbar({ doc }: { doc: Document }) {
  const { state, dispatch } = useApp();
  const { runExplain, runIncludeInSummary } = useAIRunners();
  const [promptOpen, setPromptOpen] = useState(false);
  const [customPrompt, setCustomPrompt] = useState("");
  const sel = state.selection;
  if (!sel || sel.documentId !== doc.id) return null;

  const asRef = (): SelectionRef => ({
    documentId: sel.documentId,
    blockId: sel.blockId,
    text: sel.text,
    range: sel.range,
  });
  const x = clamp(sel.x, 200, window.innerWidth - 200);
  const y = Math.max(70, sel.y - 16);

  const runWithPrompt = (op: "AI_EXPLAIN" | "AI_VERIFY" | "AI_EXPAND") => {
    const prompt = customPrompt.trim() || undefined;
    runExplain(op, asRef(), prompt);
    setPromptOpen(false);
    setCustomPrompt("");
    dispatch({ type: "selection.set", selection: null });
  };

  /** Save the selection as a colored highlight on this document. */
  const saveHighlight = (color: HighlightColor) => {
    dispatch({
      type: "doc.highlight.add",
      documentId: doc.id,
      highlight: {
        id: uid("hl"),
        blockId: sel.blockId,
        text: sel.text,
        range: sel.range,
        color,
        note: HIGHLIGHT_MEANING[color],
        createdAt: Date.now(),
      },
    });
    dispatch({ type: "selection.set", selection: null });
    dispatch({ type: "toast", message: `Highlight saved (${HIGHLIGHT_MEANING[color]})` });
  };

  /**
   * Send the quoted selection to the workspace summary target document.
   * Deterministic path first: when the marked target .docx is OPEN, append
   * the quote as a paragraph (its editor autosaves / the user saves it).
   * Otherwise fall back to the AI "include in summary" flow, which proposes
   * through the harness (and can publish into the target even when closed).
   */
  const sendToSummary = async () => {
    const quote = sel.text.trim();
    const srcTitle = state.documents[sel.documentId]?.metadata.title ?? "a source";
    const target = await getSummaryTargetFile().catch(() => null);
    const targetDoc = target
      ? Object.values(state.documents).find((d) => d.metadata.path === target.relPath)
      : undefined;
    if (targetDoc) {
      dispatch({
        type: "doc.blocks.replace",
        documentId: targetDoc.id,
        blocks: [
          ...targetDoc.blocks,
          {
            id: uid("blk"),
            type: "paragraph",
            segments: [{ text: `“${quote}” — highlighted from ${srcTitle}` }],
          },
        ],
      });
      dispatch({ type: "toast", message: `Sent to "${targetDoc.metadata.title}" — save it to keep the change` });
    } else {
      runIncludeInSummary(asRef());
    }
    dispatch({ type: "selection.set", selection: null });
  };

  const act = (
    label: string,
    icon: "sparkles" | "bulb" | "search" | "list",
    onClick: () => void,
    accent = false,
  ) => (
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
        {act("To summary", "list", () => void sendToSummary())}
        <span className="mx-1 h-4 w-px bg-line" />
        {/* Highlighter colors — click a dot to save the selection in it. */}
        {HIGHLIGHT_COLORS.map((c) => (
          <button
            key={c}
            type="button"
            title={`Highlight: ${HIGHLIGHT_MEANING[c]}`}
            aria-label={`Highlight ${HIGHLIGHT_MEANING[c]}`}
            onClick={() => saveHighlight(c)}
            className={cn("mx-0.5 h-4 w-4 rounded-full border border-line transition-transform hover:scale-110", HIGHLIGHT_SWATCH[c])}
          />
        ))}
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
