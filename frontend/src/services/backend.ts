/**
 * Go backend service boundary (Wails).
 *
 * The React app never touches the filesystem, database or network
 * directly. It calls these interfaces; today they are served by
 * in-memory mocks, tomorrow by Wails-bound Go methods
 * (window.go.main.App.*).
 *
 * To replace a mock with the real Go implementation:
 *   1. Generate Wails bindings for the matching Go method
 *      (backend/services/*.go).
 *   2. Swap the mock for a `WailsBackend` implementation of
 *      `BackendServices` in `createBackend()` below — nothing else
 *      in the app changes.
 *
 * See docs/architecture.md §"React ↔ Go" and backend/README.md.
 */

import type { FileTreeNode } from "../types/domain";
import { fileTreeMock } from "../mock/data";

declare global {
  interface Window {
    go?: { main?: { App?: unknown } };
  }
}

export interface FilesystemService {
  /** Root tree of the workspace folder. */
  listRoot(): Promise<FileTreeNode>;
  /** OS "show in folder" — a Go-only capability. */
  showContainingFolder(path: string): Promise<{ ok: boolean; note: string }>;
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
  storage: StorageService;
  sources: SourceService;
  workspace: WorkspaceService;
}

/* ------------------------------------------------------------------ */
/* Mock implementation (in-memory)                                     */
/* ------------------------------------------------------------------ */

const delay = (ms: number) => new Promise((r) => setTimeout(r, ms));

const MockFilesystem: FilesystemService = {
  async listRoot() {
    await delay(120);
    return fileTreeMock;
  },
  async showContainingFolder(path) {
    await delay(200);
    return {
      ok: true,
      note: `Would reveal “${path}” in the OS file manager (Go: os.StartProcess — backend/services/filesystem.go).`,
    };
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

export function createBackend(): BackendServices {
  // Future: if (window.go?.main?.App) return new WailsBackend(...);
  return {
    mode: "mock",
    filesystem: MockFilesystem,
    storage: MockStorage,
    sources: MockSources,
    workspace: MockWorkspace,
  };
}

export const backend: BackendServices = createBackend();
