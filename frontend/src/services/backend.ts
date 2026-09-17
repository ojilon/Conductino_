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
  /** Phase 8: write summary blocks as .docx under library root. */
  WriteSummaryDOCX(relPath: string, blocksJSON: string): Promise<string>;
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
  selectFolder(): Promise<string | null>;
  openFile(path: string): Promise<OpenedFile | null>;
  showContainingFolder(path: string): Promise<{ ok: boolean; note: string }>;
  libraryRoot(): Promise<string | null>;
}

export interface LibraryService {
  list(): Promise<FileTreeNode | null>;
}

export interface StorageService {
  readonly engine: "memory" | "sqlite";
  init(): Promise<void>;
}

export interface SourceService {
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

const delay = (ms: number) => new Promise((r) => setTimeout(r, ms));

const MockFilesystem: FilesystemService = {
  async selectFolder() {
    await delay(50);
    return null;
  },
  async openFile() {
    await delay(50);
    return null;
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

const WailsFilesystem: FilesystemService = {
  async selectFolder() {
    const app = wailsApp();
    if (!app || typeof app.SelectFolder !== "function") return null;
    try {
      const picked = await app.SelectFolder();
      return picked || null;
    } catch {
      return null;
    }
  },
  async openFile(path) {
    const app = wailsApp();
    if (!app || typeof app.OpenFile !== "function") return null;
    return await app.OpenFile(path);
  },
  async showContainingFolder(path) {
    const app = wailsApp();
    if (!app) return { ok: false, note: "Desktop bridge unavailable (browser mock mode)." };
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
    const fn = app && (app as unknown as { ListLibraryTree?: unknown }).ListLibraryTree;
    if (typeof fn !== "function") {
      throw new Error("Library bridge outdated — restart `wails dev` to rebind Go methods.");
    }
    const raw = (await (fn as () => Promise<FileTreeNode | FileTreeNode[] | null>)()) ?? null;
    if (Array.isArray(raw)) return raw[0] ?? null;
    return raw;
  },
};

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
  if (wailsApp()) return wailsBackend;
  return mockBackend;
}

export const backend: BackendServices = new Proxy({} as BackendServices, {
  get(_t, p: keyof BackendServices) {
    const live = wailsApp() ? wailsBackend : mockBackend;
    if (p === "mode") return live.mode;
    return live[p];
  },
});

/**
 * Phase 8: persist canonical summary blocks as a .docx under the workspace.
 * relPath empty → auto name under summaries/. Returns absolute path, or null
 * when the Wails bridge is unavailable (browser/mock mode).
 */
export async function writeSummaryDOCX(
  blocksJSON: string,
  relPath = "",
): Promise<string | null> {
  const app = wailsApp();
  if (!app || typeof app.WriteSummaryDOCX !== "function") return null;
  return app.WriteSummaryDOCX(relPath, blocksJSON);
}
