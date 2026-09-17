/**
 * Go backend service boundary (Wails).
 *
 * The React app never touches the filesystem, database or network
 * directly. It calls these interfaces; library + filesystem are served by
 * Wails-bound Go methods when the desktop shell is present
 * (window.go.frontend.App.* — bound in frontend/app.go, bridged via
 * backend/main.go which stays pure Go with no Wails imports), and by
 * in-memory mocks in browser mode. Storage/sources/workspace are still
 * mocked in both modes.
 *
 * To wire another service to Go: add its methods to the WailsApp interface
 * and Wails* implementation below — nothing else in the app changes.
 *
 * See docs/architecture.md §"React ↔ Go" and backend/README.md.
 */

import type { FileTreeNode } from "../types/domain";
import { fileTreeMock } from "../mock/data";

/**
 * Wails-bound Go shell (frontend/app.go, generated at runtime — NOT a build
 * artifact to import). Method names match the Go exported methods exactly;
 * Go `(T, error)` becomes a Promise<T> that rejects on error, and a nil
 * `*FileTreeNode` arrives as null.
 */
interface WailsApp {
  ListLibraryTree(): Promise<FileTreeNode | null>;
  SelectFolder(): Promise<string>;
  OpenFile(path: string): Promise<OpenedFile>;
  ShowContainingFolder(path: string): Promise<string>;
  LibraryRoot(): Promise<string>;
}

declare global {
  interface Window {
    go?: { frontend?: { App?: WailsApp } };
  }
}

/** Non-null only inside the desktop build (wails dev / wails build). */
function wailsApp(): WailsApp | null {
  return window.go?.frontend?.App ?? null;
}

/**
 * Typed open failure (tasks.md 1.2). Mirrors models.OpenFailureReason in
 * backend/models/models.go — keep in sync.
 */
export type OpenFailureReason = "unsupported" | "not_found" | "permission_denied" | "too_large" | "parse_error";

/**
 * Real content of one workspace file, as returned by App.OpenFile.
 * blocksJSON decodes to DocumentBlock[] (types/domain.ts).
 * A classified failure arrives as a VALUE with `reason` set and blocksJSON
 * empty — never a rejection — so the caller can tell "unsupported type"
 * apart from real read failures. Null = no bridge (mock/browser mode).
 */
export interface OpenedFile {
  title: string;
  blocksJSON: string;
  pageCount?: number;
  kind?: string;
  /** Absolute opening root tagging the file's folder (tasks.md 1.1 option b). */
  root?: string;
  /** Empty on success; set on classified failure (tasks.md 1.2). */
  reason?: OpenFailureReason;
  detail?: string;
}

export interface FilesystemService {
  /**
   * OS folder dialog (App.SelectFolder). Returns the picked absolute path,
   * or null when cancelled. The Go side repoints the library itself, so
   * callers just re-call library.list() afterwards. Null in mock/browser
   * mode (no dialog exists there) — callers treat it as "keep current tree".
   */
  selectFolder(): Promise<string | null>;
  /**
   * Open one workspace file by its Path token (App.OpenFile → Documents).
   * Real content for .txt/.md today. Null only when no bridge exists
   * (mock/browser mode). Classified failures (unsupported type, missing
   * file, access denied, too large, unreadable) resolve with `reason`
   * set — the caller shows an honest error and opens nothing. A rejection
   * means an unexpected bridge failure, not a known file problem.
   */
  openFile(path: string): Promise<OpenedFile | null>;
  /** OS "show in folder" — a Go-only capability. */
  showContainingFolder(path: string): Promise<{ ok: boolean; note: string }>;
  /**
   * Absolute workspace root (App.LibraryRoot). Null in mock/browser mode.
   * Used to tag opened documents and detect stale tabs (tasks.md 1.1 b).
   */
  libraryRoot(): Promise<string | null>;
}

/**
 * Curated library view over the chosen folder (Workspace service:
 * App.ListLibraryTree). Separate from FilesystemService on purpose — raw
 * OS capability (walk/reveal/dialog) vs. the research library (root choice,
 * nesting, later persistence + file↔document mapping).
 */
export interface LibraryService {
  /**
   * Library tree (ONE nested root node). Null means "no folder yet" — the
   * UI shows its empty state, not an error.
   */
  list(): Promise<FileTreeNode | null>;
}

export interface StorageService {
  readonly engine: "memory" | "sqlite";
  init(): Promise<void>;
}

export interface SourceService {
  /** Fetch/extract the full text of a source for AI operations. */
  fetchContent(sourceId: string): Promise<{ text: string; note?: string }>;
}

export interface WorkspaceService {
  save(): Promise<{ ok: boolean }>;
  open(path: string): Promise<{ ok: boolean }>;
}

export interface BackendServices {
  readonly mode: "mock" | "wails";
  filesystem: FilesystemService;
  library: LibraryService;
  storage: StorageService;
  sources: SourceService;
  workspace: WorkspaceService;
}

/* ------------------------------------------------------------------ */
/* Mock implementation (in-memory)                                     */
/* ------------------------------------------------------------------ */

const delay = (ms: number) => new Promise((r) => setTimeout(r, ms));

const MockFilesystem: FilesystemService = {
  async selectFolder() {
    await delay(50);
    return null; // no OS dialog in mock/browser mode — caller keeps current tree
  },
  async openFile() {
    await delay(50);
    return null; // no extractor in mock mode — caller uses the mock factory
  },
  async showContainingFolder(path) {
    await delay(200);
    return {
      ok: true,
      note: `Would reveal “${path}” in the OS file manager (Go: os.StartProcess — backend/services/filesystem.go).`,
    };
  },
  async libraryRoot() {
    await delay(10);
    return null;
  },
};

const MockLibrary: LibraryService = {
  async list() {
    await delay(120);
    return fileTreeMock;
  },
};

const MockStorage: StorageService = {
  engine: "memory" as const,
  async init() {
    await delay(50);
  },
};

const MockSources: SourceService = {
  async fetchContent(sourceId) {
    await delay(150);
    return {
      text: "",
      note: `Mock extraction for ${sourceId} — real extraction lives in Go (backend/services/sources.go).`,
    };
  },
};

const MockWorkspace: WorkspaceService = {
  async save() {
    await delay(80);
    return { ok: true };
  },
  async open() {
    await delay(80);
    return { ok: true };
  },
};

/* ------------------------------------------------------------------ */
/* Wails implementation (desktop build only)                           */
/* ------------------------------------------------------------------ */

// Temporary, direct push-through to the Go shell — no caching, no shaping.
// Its only job is proving the extraction pipe end-to-end: dialog → walkDir
// tree → .txt bytes → rendered tab. Rendering stays exactly as-is.
const WailsFilesystem: FilesystemService = {
  async selectFolder() {
    const app = wailsApp();
    // Guard method existence: an older bound shell (ListTree era) has no
    // SelectFolder — return null so the caller keeps the current tree
    // instead of throwing "app.SelectFolder is not a function".
    if (!app || typeof app.SelectFolder !== "function") return null;
    try {
      const picked = await app.SelectFolder();
      return picked || null; // Go returns "" on cancel
    } catch {
      return null;
    }
  },
  async openFile(path) {
    const app = wailsApp();
    if (!app || typeof app.OpenFile !== "function") return null;
    // No catch-and-null here (tasks.md 1.2): a read failure must reach the
    // caller as a typed `reason` value or a rejection — never collapse into
    // the mock path. Only truly unexpected bridge errors reject.
    return await app.OpenFile(path);
  },
  async showContainingFolder(path) {
    const app = wailsApp();
    if (!app) return { ok: false, note: "Desktop bridge unavailable (browser mock mode)." };
    // Older shells never bound ShowContainingFolder — report it instead of
    // throwing a TypeError the caller never expects.
    if (typeof app.ShowContainingFolder !== "function") {
      return { ok: false, note: "Reveal needs a newer desktop build — restart `wails dev`." };
    }
    try {
      const dir = await app.ShowContainingFolder(path);
      return { ok: true, note: `Revealed in file manager: ${dir}` };
    } catch (e) {
      return { ok: false, note: e instanceof Error ? e.message : "Reveal failed" };
    }
  },
  async libraryRoot() {
    const app = wailsApp();
    if (!app || typeof app.LibraryRoot !== "function") return null;
    try {
      const root = await app.LibraryRoot();
      return root || null;
    } catch {
      return null;
    }
  },
};

const WailsLibrary: LibraryService = {
  async list() {
    const app = wailsApp();
    // Method-missing guard: pre-fix shells only bound ListTree (slice), so
    // calling a missing ListLibraryTree would throw and the panel would
    // show "Couldn't load the library / Retry" instead of the tree.
    const fn = app && (app as unknown as { ListLibraryTree?: unknown }).ListLibraryTree;
    if (typeof fn !== "function") {
      throw new Error("Library bridge outdated — restart `wails dev` to rebind Go methods.");
    }
    const raw = (await (fn as () => Promise<FileTreeNode | FileTreeNode[] | null>)()) ?? null;
    // Defensive unwrap: very old shells returned a one-element slice.
    if (Array.isArray(raw)) return raw[0] ?? null;
    return raw;
  },
};

/* ------------------------------------------------------------------ */

const mockBackend: BackendServices = {
  mode: "mock",
  filesystem: MockFilesystem,
  library: MockLibrary,
  storage: MockStorage,
  sources: MockSources,
  workspace: MockWorkspace,
};

const wailsBackend: BackendServices = {
  mode: "wails",
  filesystem: WailsFilesystem,
  library: WailsLibrary,
  storage: MockStorage,
  sources: MockSources,
  workspace: MockWorkspace,
};

export function createBackend(): BackendServices {
  // Probe per call-site via the `backend` proxy below — this snapshot is
  // only for tests / one-shot checks. Desktop build uses Wails impls for
  // the library + filesystem pipe (storage/sources/workspace stay mocked).
  if (wailsApp()) return wailsBackend;
  return mockBackend;
}

// Live probe: window.go is injected async by the Wails runtime, AFTER this
// module first loads. A single `createBackend()` snapshot would freeze
// mode="mock" even under `wails dev`. The proxy re-probes on every property
// access, so library/folder calls use the desktop bridge as soon as it
// appears — and `backend.mode` flips without a reload.
export const backend: BackendServices = new Proxy({} as BackendServices, {
  get(_t, p: keyof BackendServices) {
    const live = wailsApp() ? wailsBackend : mockBackend;
    if (p === "mode") return live.mode;
    return live[p];
  },
});
