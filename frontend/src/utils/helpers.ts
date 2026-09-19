import type { DocumentBlock, ID } from "../types/domain";

let counter = 0;

/** Small unique id generator (UI/session scoped, not persistent). */
export function uid(prefix: string): ID {
  counter += 1;
  return `${prefix}-${counter.toString(36)}${Date.now().toString(36).slice(-4)}`;
}

export function clamp(value: number, min: number, max: number): number {
  return Math.min(max, Math.max(min, value));
}

/** "just now", "2m ago", "1h ago" — for AI activity history. */
export function timeLabel(ts: number): string {
  const s = Math.max(0, (Date.now() - ts) / 1000);
  if (s < 45) return "just now";
  if (s < 90) return "1m ago";
  if (s < 3600) return `${Math.round(s / 60)}m ago`;
  if (s < 86400) return `${Math.round(s / 3600)}h ago`;
  return `${Math.round(s / 86400)}d ago`;
}

export function truncate(text: string, max: number): string {
  return text.length > max ? `${text.slice(0, max).trimEnd()}…` : text;
}

export function hostOf(url: string): string {
  try {
    return new URL(url).host.replace(/^www\./, "");
  } catch {
    return url;
  }
}

/**
 * Normalize a workspace root for stale-tab comparison (tasks.md 1.1 option b):
 * slashes unified, trailing separators dropped, case-insensitive (Windows
 * drive letters / UNC paths must not false-mismatch on case alone).
 */
export function normalizeRoot(root: string): string {
  return root.replace(/\\/g, "/").replace(/\/+$/, "").toLowerCase();
}

/**
 * True when a document tagged with docRoot belongs to a different folder
 * than currentRoot. Untagged sides (undefined/null/empty) never count as
 * stale — mock data, summaries and pre-fix documents skip the check.
 */
export function isStaleRoot(docRoot: string | undefined | null, currentRoot: string | undefined | null): boolean {
  if (!docRoot || !currentRoot) return false;
  return normalizeRoot(docRoot) !== normalizeRoot(currentRoot);
}

/**
 * Decode extractor wire JSON ({blocks:[...]}) into blocks, coercing Go nil
 * slices (JSON null) to arrays so every consumer sees arrays. Null when the
 * shape is unusable — callers report an honest error, never a placeholder.
 */
export function parseBlocksWire(blocksJSON: string): DocumentBlock[] | null {
  try {
    const parsed: unknown = JSON.parse(blocksJSON);
    const raw = (parsed as { blocks: DocumentBlock[] }).blocks;
    if (!Array.isArray(raw)) return null;
    return raw.map((b) => ({
      ...b,
      segments: Array.isArray(b.segments) ? b.segments : [],
      listItems: b.type === "list" && Array.isArray(b.listItems) ? b.listItems : b.listItems,
    }));
  } catch {
    return null;
  }
}
