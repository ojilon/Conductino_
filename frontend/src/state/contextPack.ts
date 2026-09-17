/**
 * Phase 3 — context pack assembly (oneshot AI).
 *
 * Builds a budgeted string the backend appends to the model prompt.
 * Layers (drop lowest first if over budget):
 *   1. Selection core
 *   2. Local window (blocks before/after)
 *   3. Document outline (headings)
 *
 * Document blocks live in AppState today; Phase 6 may move assembly to Go.
 */

import type { Document, DocumentBlock, ID } from "../types/domain";

const DEFAULT_WINDOW = 2;
/** Soft character budget for the whole pack (prompt still has its own token ceiling). */
const CHAR_BUDGET = 6000;

function blockText(b: DocumentBlock): string {
  if (b.type === "list" && b.listItems?.length) {
    return b.listItems.map((item) => "• " + item.map((s) => s.text).join("")).join("\n");
  }
  return b.segments.map((s) => s.text).join("");
}

function findBlockIndex(doc: Document, blockId: ID): number {
  return doc.blocks.findIndex((b) => b.id === blockId);
}

/** Heading outline: level + truncated title. */
export function documentOutline(doc: Document, maxHeadings = 24): string {
  const lines: string[] = [];
  for (const b of doc.blocks) {
    if (b.type !== "heading") continue;
    const level = b.level ?? 1;
    const title = blockText(b).trim().slice(0, 120);
    if (!title) continue;
    lines.push(`${"#".repeat(Math.min(level, 3))} ${title}`);
    if (lines.length >= maxHeadings) break;
  }
  return lines.join("\n");
}

/** Blocks around the selection (exclusive of the selection block itself). */
export function localWindow(
  doc: Document,
  blockId: ID,
  radius = DEFAULT_WINDOW,
): { before: string; after: string } {
  const idx = findBlockIndex(doc, blockId);
  if (idx < 0) return { before: "", after: "" };
  const beforeBlocks = doc.blocks.slice(Math.max(0, idx - radius), idx);
  const afterBlocks = doc.blocks.slice(idx + 1, idx + 1 + radius);
  return {
    before: beforeBlocks.map(blockText).map((t) => t.trim()).filter(Boolean).join("\n\n"),
    after: afterBlocks.map(blockText).map((t) => t.trim()).filter(Boolean).join("\n\n"),
  };
}

export interface ContextPackOpts {
  windowBlocks?: number;
  includeOutline?: boolean;
  charBudget?: number;
}

/**
 * Assemble a structured context pack for explain / expand / verify / merge / chat.
 * Selection is optional (chat may attach only title + outline).
 */
export function buildContextPack(
  doc: Document | undefined,
  selection?: { blockId: ID; text: string },
  opts: ContextPackOpts = {},
): string {
  const radius = opts.windowBlocks ?? DEFAULT_WINDOW;
  const budget = opts.charBudget ?? CHAR_BUDGET;
  const includeOutline = opts.includeOutline !== false;

  const parts: string[] = [];
  const sel = selection?.text?.trim();
  if (sel) {
    parts.push("### Selection\n" + sel);
  }

  if (doc) {
    const title = doc.metadata?.title?.trim();
    if (title) parts.push("### Document\n" + title);

    if (selection?.blockId) {
      const { before, after } = localWindow(doc, selection.blockId, radius);
      if (before) parts.push("### Before selection\n" + before);
      if (after) parts.push("### After selection\n" + after);
    }

    if (includeOutline) {
      const outline = documentOutline(doc);
      if (outline) parts.push("### Outline\n" + outline);
    }
  }

  let pack = parts.join("\n\n");
  if (pack.length > budget) {
    pack = pack.slice(0, budget - 20) + "\n\n[…truncated…]";
  }
  return pack;
}
