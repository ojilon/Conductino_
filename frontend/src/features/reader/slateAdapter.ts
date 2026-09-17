/**
 * Canonical block/segment ↔ Slate value adapters.
 *
 * Rules (docs/plans/03-document-model-and-slate.md):
 *   - Block/segment stays the source of truth (persist + AI + extract).
 *   - Slate value is a transient view for the summary editor only.
 *   - Never store Slate JSON in SQLite or ship it over Wails as body.
 *   - blockId is preserved on every element so changes/highlights stay
 *     joinable after fromSlate.
 */

import type { Descendant } from "slate";
import type { DocumentBlock, Segment, BlockType, ID } from "../../types/domain";

/** Slate text leaf — marks map 1:1 with Segment flags. */
export type SlateText = {
  text: string;
  em?: boolean;
  strong?: boolean;
  highlightId?: ID;
};

/** Slate element — one per DocumentBlock. */
export type SlateElement = {
  type: BlockType;
  level?: 1 | 2 | 3;
  blockId: ID;
  changeId?: ID;
  children: SlateText[];
};

export type SlateValue = Descendant[];

function leavesToSegments(nodes: Descendant[]): Segment[] {
  const out: Segment[] = [];
  for (const n of nodes) {
    if ("text" in n && typeof (n as { text: string }).text === "string") {
      const t = n as SlateText;
      if (!t.text && out.length) continue;
      out.push({
        text: t.text,
        ...(t.em ? { em: true } : {}),
        ...(t.strong ? { strong: true } : {}),
        ...(t.highlightId ? { highlightId: t.highlightId } : {}),
      });
    } else if ("children" in n) {
      out.push(...leavesToSegments((n as { children: Descendant[] }).children));
    }
  }
  if (!out.length) out.push({ text: "" });
  return out;
}

/** Canonical blocks → Slate value (editor hydration). */
export function toSlate(blocks: DocumentBlock[]): SlateValue {
  if (!blocks.length) {
    return [
      {
        type: "paragraph",
        blockId: "empty",
        children: [{ text: "" }],
      } as SlateElement,
    ];
  }

  return blocks.map((b) => {
    const base: SlateElement = {
      type: b.type === "page" ? "paragraph" : b.type,
      blockId: b.id,
      children: [{ text: "" }],
    };
    if (b.level) base.level = b.level;
    if (b.changeId) base.changeId = b.changeId;

    if (b.type === "list" && b.listItems?.length) {
      const joined = b.listItems
        .map((row) => row.map((s) => s.text).join(""))
        .join("\n");
      base.children = [{ text: joined }];
      return base;
    }

    if (b.segments?.length) {
      base.children = b.segments.map((s) => ({
        text: s.text,
        ...(s.em ? { em: true } : {}),
        ...(s.strong ? { strong: true } : {}),
        ...(s.highlightId ? { highlightId: s.highlightId } : {}),
      }));
    }
    return base;
  });
}

/** Slate value → canonical blocks (on save / blur / debounced change). */
export function fromSlate(value: SlateValue): DocumentBlock[] {
  const blocks: DocumentBlock[] = [];
  for (const node of value) {
    if (!("type" in node) || !("blockId" in node)) continue;
    const el = node as unknown as SlateElement;
    const type: BlockType =
      el.type === "heading" || el.type === "list" || el.type === "page" || el.type === "paragraph"
        ? el.type
        : "paragraph";

    const block: DocumentBlock = {
      id: el.blockId || cryptoRandomId(),
      type,
      segments: leavesToSegments(el.children as Descendant[]),
    };
    if (el.level) block.level = el.level;
    if (el.changeId) block.changeId = el.changeId;

    if (type === "list") {
      const text = block.segments.map((s) => s.text).join("");
      block.listItems = text.split("\n").map((line) => [{ text: line }]);
    }

    blocks.push(block);
  }
  return blocks.length ? blocks : [{ id: cryptoRandomId(), type: "paragraph", segments: [{ text: "" }] }];
}

function cryptoRandomId(): string {
  if (typeof crypto !== "undefined" && "randomUUID" in crypto) {
    return crypto.randomUUID();
  }
  return `blk-${Date.now()}-${Math.random().toString(36).slice(2, 9)}`;
}

/** True when two Slate values serialize to the same canonical blocks. */
export function sameBlocks(a: DocumentBlock[], b: DocumentBlock[]): boolean {
  if (a.length !== b.length) return false;
  for (let i = 0; i < a.length; i++) {
    if (a[i].id !== b[i].id || a[i].type !== b[i].type) return false;
    const ta = a[i].segments.map((s) => s.text).join("");
    const tb = b[i].segments.map((s) => s.text).join("");
    if (ta !== tb) return false;
  }
  return true;
}
