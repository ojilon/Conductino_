/**
 * Mirror-file types — the TS twin of backend/docpipe/types.go.
 *
 * Every document the app opens is mirrored by the Go pipeline into TWO
 * JSON files (kept in the app-data cache and attached to OpenFile):
 *
 *   content.json → MirrorContent — the document text, FREE of pdf/docx
 *                  rules: plain blocks the canvas renders and edits.
 *   meta.json    → MirrorMeta    — the GUIDE: how the content sat in the
 *                  source file (kind, page size, fonts, margins, which
 *                  page each block was on, per-block hashes). The canvas
 *                  uses it to mirror the source LOOK its own way, and the
 *                  save bridge uses the hashes to verify what changed.
 *
 * Keep field names in sync with Go — they cross the bridge as JSON.
 */

/** One run of text with optional simple styling (italic / bold). */
export interface MirrorSegment {
  text: string;
  em?: boolean;
  strong?: boolean;
}

/** One block of the format-free content (mirrors DocumentBlock + page). */
export interface MirrorBlock {
  id: string;
  type: "heading" | "paragraph" | "list" | "page";
  level?: number;
  segments: MirrorSegment[];
  listItems?: MirrorSegment[][];
  /** 1-based page of the source this block sat on. */
  page?: number;
}

/** The whole content.json file. */
export interface MirrorContent {
  version: number;
  kind: string; // docx | pdf | text
  title: string;
  blocks: MirrorBlock[];
  /** Unix ms of the last in-app save (0/undefined = never edited). */
  editedAt?: number;
}

/** A font description in points (like the source formats use). */
export interface FontInfo {
  family: string;
  sizePt: number;
}

/** Page margins in points (72 pt = 1 inch). */
export interface MarginsPt {
  top: number;
  right: number;
  bottom: number;
  left: number;
}

/** One source page's real size in points (PDF pages can differ). */
export interface PageInfo {
  number: number;
  widthPt: number;
  heightPt: number;
}

/** How one block appeared at extraction time (verification data). */
export interface BlockOrigin {
  id: string;
  type: string;
  page?: number;
  /** Short sha256 of the block's plain text at extraction time. */
  textHash: string;
}

/** The whole meta.json file — the rendering guide + verification data. */
export interface MirrorMeta {
  version: number;
  kind: string;
  /** Workspace-relative path of the source file. */
  sourcePath: string;
  /** "<mtimeMillis>-<sizeBytes>" of the source at extraction time. */
  sourceStamp: string;
  extractedAt: number;

  pageWidthPt: number;
  pageHeightPt: number;
  marginsPt: MarginsPt;
  defaultFont: FontInfo;
  pages?: PageInfo[];
  /** Synthetic page budget for formats without real pages (docx/txt/md). */
  charsPerPage: number;
  /** False for PDFs — read + highlight only. */
  editable: boolean;

  blocks: BlockOrigin[];
}

/** What the save bridge (App.SaveDocumentMirror) reports back as JSON. */
export interface MirrorWriteReport {
  written: boolean;
  changed: string[];
  added: string[];
  removed: string[];
  unchangedCount: number;
  kind?: string;
  path?: string;
}

/** The workspace summary target marker (App.GetSummaryTarget JSON). */
export interface SummaryTarget {
  root: string;
  relPath: string;
  title: string;
  setAt?: number;
}

/* ------------------------------------------------------------------ */
/* Highlight colors                                                    */
/*                                                                     */
/* The user picks a color per highlight; yellow is reserved by convention */
/* for AI changes (published diffs render yellow-ish in the summary      */
/* view), green means "read/kept". Classes are Tailwind-safe literals.   */
/* ------------------------------------------------------------------ */

export type HighlightColor = "yellow" | "green" | "blue" | "pink";

export const HIGHLIGHT_COLORS: HighlightColor[] = ["yellow", "green", "blue", "pink"];

/** Tailwind classes for one highlight color (mark background + text). */
export const HIGHLIGHT_CLASS: Record<HighlightColor, string> = {
  yellow: "bg-amber-200/70 text-ink-900",
  green: "bg-moss-200/70 text-ink-900",
  blue: "bg-sky-200/70 text-ink-900",
  pink: "bg-rose-200/70 text-ink-900",
};

/** Swatch dot classes (color picker buttons). */
export const HIGHLIGHT_SWATCH: Record<HighlightColor, string> = {
  yellow: "bg-amber-300",
  green: "bg-moss-300",
  blue: "bg-sky-300",
  pink: "bg-rose-300",
};

/** Human meaning of each color (tooltips). */
export const HIGHLIGHT_MEANING: Record<HighlightColor, string> = {
  yellow: "AI changed / to review",
  green: "Read — done",
  blue: "Important",
  pink: "Question for later",
};

/* ------------------------------------------------------------------ */
/* Small helpers shared by the canvas + save bridge                    */
/* ------------------------------------------------------------------ */

/** Points → CSS pixels ( browsers define 1pt = 96/72 px ). */
export const PX_PER_PT = 96 / 72;

/**
 * Page fit scale: real page sizes (Letter = 612pt ≈ 816px) are wider than
 * a comfortable reading column, so the canvas scales pages down — its own
 * interpretation of the source size, exactly as the mirror concept intends.
 */
export const PAGE_FIT_SCALE = 0.78;
