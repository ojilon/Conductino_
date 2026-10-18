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

import type { ChatMessage, ChatThread, FileTreeNode } from "../types/domain";
import type { MirrorContent, MirrorMeta, MirrorWriteReport, SummaryTarget } from "../types/mirror";

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
  /** Session AI telemetry (per-provider calls + tool counts), "" when idle. */
  AIMeters(): Promise<string>;
  /** Raw workspace bytes (base64) for renderers needing originals (PDF canvas). */
  ReadRawFile(path: string): Promise<string>;
  /** Server-side budgeted context assembly (plan 11 §4). */
  BuildContextPack(
    blocksJSON: string,
    title: string,
    blockId: string,
    selText: string,
    rangeStart: number,
    rangeEnd: number,
    windowBlocks: number,
    includeOutline: boolean,
  ): Promise<string>;
  /** Skill excerpts for an operation+intent, as JSON [{name, excerpt}]. */
  MatchSkills(operation: string, intent: string): Promise<string>;
  /** Chat thread persistence (SQLite in desktop, no-op without bridge). */
  SaveThread(t: ChatThreadDTO): Promise<void>;
  AppendChatMessage(m: ChatMessageDTO): Promise<void>;
  /** Map a document as the workspace primary summary (SQLite, desktop). */
  SetPrimarySummary(summaryId: string): Promise<void>;
  LoadThread(id: string): Promise<ChatThreadDTO | null>;
  ListThreads(workspaceId: string): Promise<ChatThreadDTO[]>;
  ListMessages(threadId: string): Promise<ChatMessageDTO[]>;

  /* ---------------- Mirror pipeline + file ops (backend/docpipe) -------- */
  /** SAVE BRIDGE: edited mirror content JSON → verified write-back to the
   *  source file. Resolves to the WriteReport as a JSON string. */
  SaveDocumentMirror(path: string, contentJSON: string): Promise<string>;
  /** Create an empty file in the workspace (.docx gets a valid empty Word
   *  package; .txt/.md an empty text file). Returns the relative path. */
  CreateFile(path: string): Promise<string>;
  /** Create a folder in the workspace. Returns the relative path. */
  CreateFolder(path: string): Promise<string>;
  /** Rename a file/folder in place (new base name only). */
  RenameFile(path: string, newName: string): Promise<string>;
  /** Mark a .docx as THE workspace summary target (all summaries go there). */
  SetSummaryTarget(path: string): Promise<void>;
  /** The marked summary target as JSON ("" when none). */
  GetSummaryTarget(): Promise<string>;
  /** The AI skills folder as a file tree (same shape as the library). */
  ListSkillsTree(): Promise<FileTreeNode | null>;
  /** Read one skills-folder instruction file. */
  ReadSkillFile(path: string): Promise<string>;
  /** Save one skills-folder instruction file. */
  WriteSkillFile(path: string, content: string): Promise<void>;
  /** Create a file (isDir=false) or folder (isDir=true) inside skills/. */
  CreateSkillEntry(path: string, isDir: boolean): Promise<void>;
  /** Run one basic command in the sandboxed shell (cwd = open folder). */
  RunShell(command: string): Promise<string>;
  /** Push LIVE editor state into the mirror's content.json (no source
   *  write) so the AI's read_mirror view matches what the user sees. */
  UpdateMirrorContent(path: string, contentJSON: string): Promise<void>;
}

/** Wire shapes of Go persistence records (models/storage.go). */
export interface ChatThreadDTO {
  id: string;
  workspaceId: string;
  documentId?: string;
  title?: string;
  createdAt?: number;
  updatedAt?: number;
}

export interface ChatMessageDTO {
  id: string;
  threadId: string;
  role: string;
  content: string;
  documentId?: string;
  /** JSON [{tool, ok, ms}] thinking log (assistant turns that ran tools). */
  toolTrace?: string;
  createdAt?: number;
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
  /* ---- Mirror pair (backend/docpipe): compact JSON strings ---- */
  /** content.json — the format-free document content (MirrorContent). */
  mirrorJSON?: string;
  /** meta.json — the rendering guide / verification data (MirrorMeta). */
  metaJSON?: string;
  /** Folder on disk holding content.json + meta.json (diagnostics). */
  mirrorDir?: string;
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
    // No mock tree: browser mode has no folder. Null is the honest
    // "no folder yet" signal — the Library panel shows Choose folder.
    return null;
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

/**
 * AI session telemetry for Settings (per-provider calls/errors/token
 * estimates/RPM + tool call counts). Null in browser/mock mode or when
 * nothing ran yet.
 */
export async function getAIMeters(): Promise<string | null> {
  const app = wailsApp();
  if (!app || typeof app.AIMeters !== "function") return null;
  try {
    const s = await app.AIMeters();
    return s?.trim() ? s : null;
  } catch {
    return null;
  }
}

/**
 * Raw workspace bytes for renderers needing originals (PDF canvas leaf).
 * Resolve-jailed on the Go side; null without the desktop bridge.
 */
export async function openRawFile(path: string): Promise<Uint8Array | null> {
  const app = wailsApp();
  if (!app || typeof app.ReadRawFile !== "function") return null;
  const b64 = await app.ReadRawFile(path);
  const bin = atob(b64);
  const out = new Uint8Array(bin.length);
  for (let i = 0; i < bin.length; i++) out[i] = bin.charCodeAt(i);
  return out;
}

/**
 * Server-side context pack (plan 11 §4). Bridge-first; null without the
 * desktop bridge (callers fall back to the local builder).
 */
export async function buildContextPackRemote(
  blocksJSON: string,
  title: string,
  blockId: string,
  selText: string,
  rangeStart: number,
  rangeEnd: number,
  windowBlocks = 2,
  includeOutline = true,
): Promise<string | null> {
  const app = wailsApp();
  if (!app || typeof app.BuildContextPack !== "function") return null;
  try {
    const pack = await app.BuildContextPack(
      blocksJSON,
      title,
      blockId,
      selText,
      rangeStart,
      rangeEnd,
      windowBlocks,
      includeOutline,
    );
    return pack?.trim() ? pack : null;
  } catch {
    return null;
  }
}

/**
 * Skill excerpts for an operation+intent (plan 11 §1: future Skills UI,
 * debugging). Empty array without the bridge or when nothing matches.
 */
export async function matchSkills(operation: string, intent: string): Promise<{ name: string; excerpt: string }[]> {
  const app = wailsApp();
  if (!app || typeof app.MatchSkills !== "function") return [];
  try {
    const raw = await app.MatchSkills(operation, intent);
    const parsed: unknown = JSON.parse(raw || "[]");
    if (!Array.isArray(parsed)) return [];
    return parsed.filter(
      (e): e is { name: string; excerpt: string } =>
        !!e && typeof (e as { name?: unknown }).name === "string" && typeof (e as { excerpt?: unknown }).excerpt === "string",
    );
  } catch {
    return [];
  }
}

/* ------------------------------------------------------------------ */
/* Chat thread persistence (SQLite in desktop; silent no-op in browser) */
/* ------------------------------------------------------------------ */

function toThreadDTO(t: ChatThread): ChatThreadDTO {
  return {
    id: t.id,
    workspaceId: t.workspaceId,
    documentId: t.documentId,
    title: t.title,
    createdAt: t.createdAt,
    updatedAt: t.updatedAt,
  };
}

function toMessageDTO(threadId: string, m: ChatMessage): ChatMessageDTO {
  return {
    id: m.id,
    threadId,
    role: m.role,
    content: m.content,
    documentId: m.documentId,
    toolTrace: m.toolTrace?.length ? JSON.stringify(m.toolTrace) : undefined,
    createdAt: m.createdAt,
  };
}

function fromMessageDTO(m: ChatMessageDTO): ChatMessage {
  let toolTrace: ChatMessage["toolTrace"];
  if (m.toolTrace) {
    try {
      const parsed: unknown = JSON.parse(m.toolTrace);
      if (Array.isArray(parsed)) {
        toolTrace = parsed.filter(
          (e): e is NonNullable<ChatMessage["toolTrace"]>[number] =>
            !!e &&
            typeof (e as { tool?: unknown }).tool === "string" &&
            typeof (e as { ok?: unknown }).ok === "boolean" &&
            typeof (e as { ms?: unknown }).ms === "number",
        );
      }
    } catch {
      toolTrace = undefined;
    }
  }
  return {
    id: m.id,
    role: m.role as ChatMessage["role"],
    content: m.content,
    createdAt: m.createdAt ?? Date.now(),
    documentId: m.documentId,
    toolTrace,
  };
}

/** Persist a thread (upsert). False without the bridge or on error. */
export async function saveChatThread(t: ChatThread): Promise<boolean> {
  const app = wailsApp();
  if (!app || typeof app.SaveThread !== "function") return false;
  try {
    await app.SaveThread(toThreadDTO(t));
    return true;
  } catch {
    return false;
  }
}

/** Map a document as the workspace primary summary (desktop SQLite). */
export async function setPrimarySummary(summaryId: string): Promise<boolean> {
  const app = wailsApp();
  if (!app || typeof app.SetPrimarySummary !== "function") return false;
  try {
    await app.SetPrimarySummary(summaryId);
    return true;
  } catch {
    return false;
  }
}

/** Append one message. False without the bridge or on error. */
export async function appendChatMessage(threadId: string, m: ChatMessage): Promise<boolean> {
  const app = wailsApp();
  if (!app || typeof app.AppendChatMessage !== "function") return false;
  try {
    await app.AppendChatMessage(toMessageDTO(threadId, m));
    return true;
  } catch {
    return false;
  }
}

/** Load all threads of a workspace with their messages (desktop only). */
export async function loadWorkspaceThreads(workspaceId: string): Promise<ChatThread[]> {
  const app = wailsApp();
  if (!app || typeof app.ListThreads !== "function" || typeof app.ListMessages !== "function") return [];
  try {
    const dtos = (await app.ListThreads(workspaceId)) ?? [];
    const out: ChatThread[] = [];
    for (const d of dtos) {
      const msgs = (await app.ListMessages(d.id)) ?? [];
      out.push({
        id: d.id,
        workspaceId: d.workspaceId,
        documentId: d.documentId,
        title: d.title ?? "Research chat",
        messages: msgs.map(fromMessageDTO),
        createdAt: d.createdAt ?? Date.now(),
        updatedAt: d.updatedAt ?? Date.now(),
      });
    }
    return out.sort((a, b) => b.updatedAt - a.updatedAt);
  } catch {
    return [];
  }
}

/* ------------------------------------------------------------------ */
/* Mirror pipeline + library file ops + skills folder + AI shell       */
/*                                                                     */
/* Every helper is bridge-first: without the desktop shell (browser     */
/* mode) they resolve to honest empty values, never throw.             */
/* ------------------------------------------------------------------ */

/**
 * SAVE BRIDGE: send the edited mirror content back to Go. The backend
 * verifies which blocks changed (hashes from meta.json) and writes the
 * source file (.docx/.txt/.md). Null without the desktop bridge.
 */
export async function saveDocumentMirror(
  path: string,
  content: MirrorContent,
): Promise<MirrorWriteReport | null> {
  const app = wailsApp();
  if (!app || typeof app.SaveDocumentMirror !== "function") return null;
  const raw = await app.SaveDocumentMirror(path, JSON.stringify(content));
  try {
    return JSON.parse(raw) as MirrorWriteReport;
  } catch {
    return { written: true, changed: [], added: [], removed: [], unchangedCount: 0 };
  }
}

/**
 * LIVE MIRROR PUSH: update content.json from the editor without saving the
 * source file. Fire-and-forget (debounced by the caller); silently no-ops
 * without the desktop bridge. The AI's read_mirror reads these files, so
 * this is what lets it "watch" the user edit.
 */
export async function updateMirrorContent(path: string, content: MirrorContent): Promise<void> {
  const app = wailsApp();
  if (!app || typeof app.UpdateMirrorContent !== "function") return;
  try {
    await app.UpdateMirrorContent(path, JSON.stringify(content));
  } catch {
    // Best-effort: the mirror catch-up happens on the next open anyway.
  }
}

/** Create an empty workspace file (.docx/.txt/.md). Null without bridge. */
export async function createWorkspaceFile(path: string): Promise<string | null> {
  const app = wailsApp();
  if (!app || typeof app.CreateFile !== "function") return null;
  return app.CreateFile(path);
}

/** Create a workspace folder. Null without bridge. */
export async function createWorkspaceFolder(path: string): Promise<string | null> {
  const app = wailsApp();
  if (!app || typeof app.CreateFolder !== "function") return null;
  return app.CreateFolder(path);
}

/** Rename a workspace file/folder in place. Null without bridge. */
export async function renameWorkspaceFile(path: string, newName: string): Promise<string | null> {
  const app = wailsApp();
  if (!app || typeof app.RenameFile !== "function") return null;
  return app.RenameFile(path, newName);
}

/** Mark a .docx as the workspace summary target. False without bridge. */
export async function setSummaryTargetFile(path: string): Promise<boolean> {
  const app = wailsApp();
  if (!app || typeof app.SetSummaryTarget !== "function") return false;
  try {
    await app.SetSummaryTarget(path);
    return true;
  } catch (e) {
    throw e instanceof Error ? e : new Error(String(e));
  }
}

/** The marked summary target for the open workspace (null = none). */
export async function getSummaryTargetFile(): Promise<SummaryTarget | null> {
  const app = wailsApp();
  if (!app || typeof app.GetSummaryTarget !== "function") return null;
  try {
    const raw = await app.GetSummaryTarget();
    if (!raw) return null;
    return JSON.parse(raw) as SummaryTarget;
  } catch {
    return null;
  }
}

/** The AI skills folder as a file tree (null without bridge). */
export async function listSkillsTree(): Promise<FileTreeNode | null> {
  const app = wailsApp();
  if (!app || typeof app.ListSkillsTree !== "function") return null;
  try {
    return (await app.ListSkillsTree()) ?? null;
  } catch {
    return null;
  }
}

/** Read one skills-folder file (null without bridge / on error). */
export async function readSkillFile(path: string): Promise<string | null> {
  const app = wailsApp();
  if (!app || typeof app.ReadSkillFile !== "function") return null;
  try {
    return await app.ReadSkillFile(path);
  } catch {
    return null;
  }
}

/** Save one skills-folder file. False without bridge. */
export async function writeSkillFile(path: string, content: string): Promise<boolean> {
  const app = wailsApp();
  if (!app || typeof app.WriteSkillFile !== "function") return false;
  try {
    await app.WriteSkillFile(path, content);
    return true;
  } catch {
    return false;
  }
}

/** Create a file/folder inside skills/. False without bridge. */
export async function createSkillEntry(path: string, isDir: boolean): Promise<boolean> {
  const app = wailsApp();
  if (!app || typeof app.CreateSkillEntry !== "function") return false;
  try {
    await app.CreateSkillEntry(path, isDir);
    return true;
  } catch {
    return false;
  }
}

/** Run one command in the sandboxed AI shell (null without bridge). */
export async function runShellCommand(command: string): Promise<string | null> {
  const app = wailsApp();
  if (!app || typeof app.RunShell !== "function") return null;
  try {
    return await app.RunShell(command);
  } catch (e) {
    return `[shell error] ${e instanceof Error ? e.message : String(e)}`;
  }
}

/** Parse the mirror pair attached to an OpenedFile (null-safe). */
export function parseMirrorPair(opened: OpenedFile): { content: MirrorContent | null; meta: MirrorMeta | null } {
  let content: MirrorContent | null = null;
  let meta: MirrorMeta | null = null;
  try {
    if (opened.mirrorJSON) content = JSON.parse(opened.mirrorJSON) as MirrorContent;
  } catch {
    content = null;
  }
  try {
    if (opened.metaJSON) meta = JSON.parse(opened.metaJSON) as MirrorMeta;
  } catch {
    meta = null;
  }
  return { content, meta };
}
