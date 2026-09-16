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
 * Real content of one workspace file, as returned by App.OpenFile.
 * blocksJSON decodes to DocumentBlock[] (types/domain.ts). Null = mock mode
 * or unsupported type — the caller falls back to mock extraction.
 */
export interface OpenedFile {
  title: string;
  blocksJSON: string;
  pageCount?: number;
  kind?: string;
  /** Absolute opening root tagging the file's folder (tasks.md 1.1 option b). */
  root?: string;
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
   * Real content for .txt/.md today; null in mock mode or for unsupported
   * types (caller falls back to the mock document factory).
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
    if (!app) return null;
    const picked = await app.SelectFolder();
    return picked || null; // Go returns "" on cancel
  },
  async openFile(path) {
    const app = wailsApp();
    if (!app) return null;
    try {
      return await app.OpenFile(path);
    } catch {
      return null; // unsupported type / read error → caller uses mock fallback
    }
  },
  async showContainingFolder(path) {
    const app = wailsApp();
    if (!app) return { ok: false, note: "Desktop bridge unavailable (browser mock mode)." };
    try {
      const dir = await app.ShowContainingFolder(path);
      return { ok: true, note: `Revealed in file manager: ${dir}` };
    } catch (e) {
      return { ok: false, note: e instanceof Error ? e.message : "Reveal failed" };
    }
  },
  async libraryRoot() {
    const app = wailsApp();
    if (!app) return null;
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
    return wailsApp()?.ListLibraryTree() ?? null;
  },
};

/* ------------------------------------------------------------------ */

export function createBackend(): BackendServices {
  // Desktop build: the Go shell is present — use it for the library +
  // filesystem pipe (storage/sources/workspace stay mocked for now).
  if (wailsApp()) {
    return {
      mode: "wails",
      filesystem: WailsFilesystem,
      library: WailsLibrary,
      storage: MockStorage,
      sources: MockSources,
      workspace: MockWorkspace,
    };
  }
  return {
    mode: "mock",
    filesystem: MockFilesystem,
    library: MockLibrary,
    storage: MockStorage,
    sources: MockSources,
    workspace: MockWorkspace,
  };
}

export const backend: BackendServices = createBackend();
