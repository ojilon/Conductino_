/**
 * Reader left side: a slim icon rail + a collapsible secondary panel.
 *
 * Secondary panel views (driven by rail icons, part of ReaderUIState):
 *  - documents: document info + table of contents + file tree
 *               (or "sources feeding the summary" when a summary is open)
 *  - bookmarks / notes: real lists from state
 *  - library / settings: honest placeholders for future backend features
 *
 * Filesystem access goes through backend.filesystem (service boundary) —
 * the component never touches paths directly beyond dispatching intent.
 */

import { useEffect, useState } from "react";
import { useApp } from "../../state/appState";
import { backend } from "../../services/backend";
import {
  createWorkspaceFile,
  createWorkspaceFolder,
  renameWorkspaceFile,
  setSummaryTargetFile,
} from "../../services/backend";
import type { Document, FileTreeNode, RailView } from "../../types/domain";
import { Icon, type IconName } from "../../components/icons";
import { Button, EmptyState } from "../../components/ui";
import { cn } from "../../utils/cn";
import { isStaleRoot, truncate } from "../../utils/helpers";

const RAIL: { id: RailView; icon: IconName; label: string }[] = [
  { id: "documents", icon: "fileText", label: "Document" },
  { id: "bookmarks", icon: "bookmark", label: "Bookmarks" },
  { id: "notes", icon: "note", label: "Notes" },
  { id: "library", icon: "book", label: "Library" },
  { id: "settings", icon: "settings", label: "Settings" },
];

/* ------------------------------------------------------------------ */
/* File tree (Workspace-owned library: backend.library.list)             */
/* ------------------------------------------------------------------ */

/** Join a folder token + a name into a workspace-relative path token. */
function joinToken(folderToken: string, name: string): string {
  const clean = name.trim().replace(/^[/\\]+|[/\\]+$/g, "");
  return folderToken ? `${folderToken}/${clean}` : clean;
}

/**
 * Tree file operations (create / rename / summary target / reveal).
 * Shared by the tree's right-click menu and the Library panel's "+ New"
 * buttons. Everything goes through the Go bridge (resolve-jailed); after a
 * successful op the tree is refreshed by the caller-supplied callbacks.
 */
function useTreeActions(onTreeChanged: () => void, onTargetChanged: () => void) {
  const { dispatch } = useApp();
  const fail = (what: string, e: unknown) =>
    dispatch({ type: "toast", message: `${what}: ${e instanceof Error ? e.message : String(e)}` });
  const noBridge = (what: string) =>
    dispatch({ type: "toast", message: `${what} needs the desktop app — run \`wails dev\`.` });

  return {
    /** Create an empty file (.docx gets a valid empty Word package). */
    createFile: async (folderToken: string, suggested = "Untitled.docx") => {
      const name = window.prompt("New file name (.docx, .txt or .md):", suggested);
      if (!name?.trim()) return;
      try {
        const created = await createWorkspaceFile(joinToken(folderToken, name));
        if (!created) return noBridge("Creating files");
        dispatch({ type: "toast", message: `Created ${created}` });
        onTreeChanged();
      } catch (e) {
        fail("Create failed", e);
      }
    },
    /** Create a folder. */
    createFolder: async (folderToken: string) => {
      const name = window.prompt("New folder name:", "my-notes");
      if (!name?.trim()) return;
      try {
        const created = await createWorkspaceFolder(joinToken(folderToken, name));
        if (!created) return noBridge("Creating folders");
        dispatch({ type: "toast", message: `Created folder ${created}` });
        onTreeChanged();
      } catch (e) {
        fail("Create failed", e);
      }
    },
    /** Rename a file/folder in place (new base name only). */
    rename: async (token: string, label: string) => {
      const name = window.prompt("New name:", label);
      if (!name?.trim() || name.trim() === label) return;
      try {
        const renamed = await renameWorkspaceFile(token, name.trim());
        if (!renamed) return noBridge("Renaming");
        dispatch({ type: "toast", message: `Renamed to ${renamed}` });
        onTreeChanged();
      } catch (e) {
        fail("Rename failed", e);
      }
    },
    /** Mark a .docx as THE workspace summary target (all summaries go there). */
    makeSummaryTarget: async (token: string, label: string) => {
      try {
        const ok = await setSummaryTargetFile(token);
        if (!ok) return noBridge("Setting the summary target");
        dispatch({
          type: "toast",
          message: `"${label}" is now the target summary document — all summaries are siphoned into it`,
        });
        onTargetChanged();
      } catch (e) {
        fail("Could not set the summary target", e);
      }
    },
    /** Reveal in the OS file manager. */
    reveal: async (token: string) => {
      const r = await backend.filesystem.showContainingFolder(token);
      dispatch({ type: "toast", message: r.note });
    },
  };
}

/** One right-click menu entry in the file tree. */
interface TreeMenuTarget {
  token: string; // workspace-relative path token
  label: string;
  kind: "folder" | "file";
  ext?: string;
  x: number;
  y: number;
}

function FileTree({
  node,
  onFile,
  focusPath,
  onFocusDone,
  summaryTarget,
  onTreeChanged,
  onTargetChanged,
}: {
  node: FileTreeNode;
  onFile: (n: FileTreeNode & { path: string }) => void;
  /** Path token to expand + highlight + scroll to (from "Show in library"). */
  focusPath?: string | null;
  onFocusDone?: () => void;
  /** Marked summary-target .docx (starred in the tree). */
  summaryTarget?: string | null;
  /** Refresh callbacks after create/rename/target ops. */
  onTreeChanged: () => void;
  onTargetChanged: () => void;
}) {
  // Open/active state is keyed by TOKEN (n.path ?? label-built prefix), not
  // by id: tokens are the stable cross-world identity (real trees use them
  // as ids anyway; label-built tokens match flatFiles, so ancestors always
  // line up).
  const [open, setOpen] = useState<Record<string, boolean>>({});
  const [activeToken, setActiveToken] = useState<string>("");
  const [menu, setMenu] = useState<TreeMenuTarget | null>(null);
  const actions = useTreeActions(onTreeChanged, onTargetChanged);

  const tokenOf = (n: FileTreeNode, path: string) => n.path ?? (path ? `${path}/${n.label}` : n.label);

  // Open top-level folders whenever a new tree arrives.
  useEffect(() => {
    const initial: Record<string, boolean> = {};
    node.children?.forEach((c) => {
      if (c.kind === "folder") initial[tokenOf(c, "")] = true;
    });
    setOpen(initial);
    setActiveToken("");
  }, [node]);

  // "Show in library": expand every ancestor prefix of the focused token,
  // highlight it, scroll it into view, then report back so the parent clears
  // the request (one-shot — it must not re-fire on later tree refreshes).
  useEffect(() => {
    if (!focusPath) return;
    const parts = focusPath.split("/");
    const openNext: Record<string, boolean> = {};
    let acc = "";
    for (let i = 0; i < parts.length - 1; i++) {
      acc = acc ? `${acc}/${parts[i]}` : parts[i];
      openNext[acc] = true;
    }
    setOpen((o) => ({ ...o, ...openNext }));
    setActiveToken(focusPath);
    requestAnimationFrame(() => {
      document
        .querySelector(`[data-node-token="${CSS.escape(focusPath)}"]`)
        ?.scrollIntoView({ behavior: "smooth", block: "center" });
    });
    onFocusDone?.();
  }, [focusPath, node]);

  // Right-click anywhere else closes the tree menu.
  useEffect(() => {
    if (!menu) return;
    const close = () => setMenu(null);
    window.addEventListener("click", close);
    window.addEventListener("contextmenu", close);
    return () => {
      window.removeEventListener("click", close);
      window.removeEventListener("contextmenu", close);
    };
  }, [menu]);

  const walk = (n: FileTreeNode, path: string, d: number) => {
    const token = tokenOf(n, path);
    const isTarget = !!summaryTarget && token === summaryTarget;
    return (
      <div key={n.id}>
        <button
          type="button"
          data-node-token={token}
          style={{ paddingLeft: `${8 + d * 14}px` }}
          onClick={() => {
            if (n.kind === "folder") setOpen((o) => ({ ...o, [token]: !o[token] }));
            else {
              setActiveToken(token);
              onFile({ ...n, path: token });
            }
          }}
          onContextMenu={(e) => {
            e.preventDefault();
            e.stopPropagation();
            setMenu({ token, label: n.label, kind: n.kind, ext: n.ext, x: e.clientX, y: e.clientY });
          }}
          className={cn(
            "flex w-full items-center gap-1.5 rounded-md py-[5px] pr-2 text-left text-[12.5px] transition-colors",
            activeToken === token ? "bg-iris-100/70 text-iris-700" : "text-ink-700 hover:bg-cream-100",
          )}
        >
          {n.kind === "folder" ? (
            <>
              <Icon name="chevronRight" size={11} className={cn("text-mute transition-transform", open[token] && "rotate-90")} />
              <Icon name="folder" size={13} className="shrink-0 text-mute" />
            </>
          ) : (
            <>
              <span className="w-[11px]" />
              <Icon
                name="fileText"
                size={13}
                className={cn("shrink-0", n.ext === "pdf" ? "text-[#c42b2b]" : "text-iris-500")}
              />
            </>
          )}
          <span className="truncate">{n.label}</span>
          {isTarget && (
            <span title="Workspace summary target — all summaries are siphoned into this file" className="ml-auto shrink-0 text-amber-500">
              ★
            </span>
          )}
        </button>
        {n.kind === "folder" && open[token] && n.children?.map((c) => walk(c, token, d + 1))}
      </div>
    );
  };

  const menuItem = (label: string, onClick: () => void, danger = false) => (
    <button
      type="button"
      onClick={(e) => {
        e.stopPropagation();
        setMenu(null);
        onClick();
      }}
      className={cn(
        "w-full rounded-md px-2.5 py-1.5 text-left text-[12px] font-medium transition-colors",
        danger ? "text-rose-600 hover:bg-rose-50" : "text-ink-700 hover:bg-cream-100",
      )}
    >
      {label}
    </button>
  );

  return (
    <div>
      {node.children?.map((c) => walk(c, "", 0))}
      {menu && (
        <div
          data-tree-menu
          className="fade-in fixed z-50 w-[240px] rounded-xl border border-line bg-white p-1.5 shadow-xl shadow-ink-900/10"
          style={{
            left: Math.max(8, Math.min(menu.x, window.innerWidth - 250)),
            top: Math.max(8, Math.min(menu.y, window.innerHeight - 260)),
          }}
          onClick={(e) => e.stopPropagation()}
          onContextMenu={(e) => e.preventDefault()}
        >
          <p className="truncate px-2.5 pb-1 pt-0.5 text-[10.5px] uppercase tracking-wide text-mute">{menu.label}</p>
          {menu.kind === "folder" ? (
            <>
              {menuItem("New Word document (.docx)", () => void actions.createFile(menu.token, "Summary.docx"))}
              {menuItem("New text file (.txt)", () => void actions.createFile(menu.token, "notes.txt"))}
              {menuItem("New markdown file (.md)", () => void actions.createFile(menu.token, "notes.md"))}
              {menuItem("New folder…", () => void actions.createFolder(menu.token))}
              {menuItem("Rename…", () => void actions.rename(menu.token, menu.label))}
              {menuItem("Show in folder", () => void actions.reveal(menu.token))}
            </>
          ) : (
            <>
              {menuItem("Rename…", () => void actions.rename(menu.token, menu.label))}
              {(menu.ext ?? "").toLowerCase() === "docx" &&
                menuItem("★ Make this the target summary document", () => void actions.makeSummaryTarget(menu.token, menu.label))}
              {menuItem("Show in folder", () => void actions.reveal(menu.token))}
            </>
          )}
        </div>
      )}
    </div>
  );
}

/* ------------------------------------------------------------------ */
/* Document info + TOC                                                 */
/* ------------------------------------------------------------------ */

function DocumentInfoPanel({
  doc,
  onShowInLibrary,
  currentRoot,
}: {
  doc: Document;
  /** Reveal this file's location in the Library view (no-op when unknown). */
  onShowInLibrary: (path: string, rootPath?: string) => void;
  /** Absolute workspace root for stale-tab detection (tasks.md 1.1 option b). */
  currentRoot: string | null;
}) {
  const { dispatch } = useApp();
  const [tocActive, setTocActive] = useState<string | null>(null);
  const meta = doc.metadata;
  const headings = doc.blocks.filter((b) => b.type === "heading" && b.level === 2);

  useEffect(() => {
    setTocActive(headings[0]?.id ?? null);
  }, [doc.id, headings.length]);

  const scrollTo = (blockId: string) => {
    setTocActive(blockId);
    document.getElementById(`blk-${blockId}`)?.scrollIntoView({ behavior: "smooth", block: "start" });
  };

  return (
    <div className="fade-in space-y-5 px-4 py-4">
      {/* info card */}
      <div className="flex gap-3 rounded-lg border border-line-soft bg-cream-50 p-3">
        <div className="flex h-[64px] w-12 shrink-0 flex-col items-center justify-center gap-1 rounded border border-line bg-white">
          <Icon name="fileText" size={16} className={meta.format === "pdf" ? "text-[#c42b2b]" : "text-iris-500"} />
          <span className="text-[8px] font-bold uppercase text-mute">{meta.format}</span>
        </div>
        <div className="min-w-0">
          <p className="font-serif text-[13.5px] font-bold leading-snug text-ink-900">{meta.title}</p>
          {meta.author && (
            <p className="mt-1 text-[11.5px] text-ink-500">
              {meta.author}
              {meta.year ? ` (${meta.year})` : ""}
            </p>
          )}
          {meta.venue && <p className="text-[11px] text-mute">{meta.venue}</p>}
          <p className="mt-1 text-[10.5px] text-mute">
            {meta.pageCount ? `${meta.pageCount} pages · ` : ""}
            {meta.format.toUpperCase()}
          </p>
        </div>
      </div>

      {/* TOC */}
      {headings.length > 0 && (
        <section>
          <h4 className="mb-1.5 px-1 text-[11px] font-semibold uppercase tracking-wide text-mute">
            Table of contents
          </h4>
          <ul className="space-y-0.5">
            {headings.map((h) => (
              <li key={h.id}>
                <button
                  type="button"
                  onClick={() => scrollTo(h.id)}
                  className={cn(
                    "flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-left text-[12.5px] transition-colors",
                    tocActive === h.id ? "bg-iris-100/80 font-medium text-iris-700" : "text-ink-700 hover:bg-cream-100",
                  )}
                >
                  <span
                    className={cn(
                      "h-1.5 w-1.5 shrink-0 rounded-full",
                      tocActive === h.id ? "bg-iris-500" : "border border-line",
                    )}
                  />
                  <span className="truncate">{h.segments.map((s) => s.text).join("")}</span>
                </button>
              </li>
            ))}
          </ul>
        </section>
      )}

      {/* file location: the tree itself lives in the Library view now */}
      <section>
        <h4 className="mb-1.5 px-1 text-[11px] font-semibold uppercase tracking-wide text-mute">File</h4>
        {isStaleRoot(meta.rootPath, currentRoot) ? (
          <div className="rounded-lg border border-amber-300 bg-amber-50 px-3 py-2">
            <p className="text-[12px] font-medium text-amber-800">This file belongs to another folder.</p>
            <p className="mt-0.5 truncate text-[11px] text-amber-700" title={meta.rootPath}>
              Opened from {meta.rootPath}
            </p>
          </div>
        ) : meta.path ? (
          <Button variant="soft" icon="book" className="w-full" onClick={() => onShowInLibrary(meta.path!, meta.rootPath)}>
            Show in library
          </Button>
        ) : (
          <p className="px-1 py-1 text-[12px] text-mute">No file location for this document.</p>
        )}
        <Button
          variant="soft"
          icon="folder"
          className="mt-2 w-full"
          onClick={() => {
            // Stale guard (tasks.md 1.1 option b): never resolve a tab tagged
            // with another folder's root against the current root — Go's
            // Resolve would silently target new-root/old-relative-path.
            if (isStaleRoot(meta.rootPath, currentRoot)) {
              dispatch({ type: "toast", message: "This file belongs to another folder — switch back to reveal it." });
              return;
            }
            const path = meta.path ?? "research_workspace";
            backend.filesystem.showContainingFolder(path).then((r) => {
              dispatch({ type: "toast", message: r.note });
            });
          }}
        >
          Show containing folder
        </Button>
      </section>
    </div>
  );
}

/* ------------------------------------------------------------------ */
/* Sources feeding the summary                                         */
/* ------------------------------------------------------------------ */

const DOT_COLORS = ["bg-iris-500", "bg-violet-500", "bg-sky-500", "bg-moss-600", "bg-amber-500"];

function SummarySourcesPanel({ doc }: { doc: Document }) {
  const { state } = useApp();
  const sourceIds = doc.sourceIds ?? [];
  return (
    <div className="fade-in space-y-5 px-4 py-4">
      <section>
        <h4 className="mb-1.5 px-1 text-[11px] font-semibold uppercase tracking-wide text-mute">Sources</h4>
        <ul className="space-y-2">
          {sourceIds.map((sid, i) => {
            const src = state.sources[sid];
            if (!src) return null;
            return (
              <li key={sid} className="flex items-center gap-2.5 rounded-md border border-line-soft bg-cream-50 px-3 py-2">
                <Icon name="fileText" size={14} className="shrink-0 text-ink-400" />
                <div className="min-w-0 flex-1">
                  <p className="truncate text-[12.5px] font-medium text-ink-900">{src.title}</p>
                  <p className="truncate text-[11px] text-mute">{truncate(src.abstract, 34)}</p>
                </div>
                <span className={cn("h-2 w-2 shrink-0 rounded-full", DOT_COLORS[i % DOT_COLORS.length])} />
              </li>
            );
          })}
        </ul>
      </section>

      <section>
        <h4 className="mb-1.5 flex items-center gap-1.5 px-1 text-[11px] font-semibold uppercase tracking-wide text-mute">
          <Icon name="link" size={12} /> Cited sources
        </h4>
        <ul className="space-y-1 px-1">
          {sourceIds.map((sid, i) => {
            const src = state.sources[sid];
            return (
              <li key={sid} className="flex items-center gap-2 py-1 text-[12.5px] text-ink-700">
                <span className={cn("h-2 w-2 rounded-full", DOT_COLORS[i % DOT_COLORS.length])} />
                <span className="truncate">{src?.title ?? sid}</span>
              </li>
            );
          })}
        </ul>
        <p className="mt-2 px-1 font-serif text-[12px] italic text-mute">Feed this summary</p>
      </section>
    </div>
  );
}

/* ------------------------------------------------------------------ */
/* Library (Workspace-owned file tree)                                   */
/* ------------------------------------------------------------------ */

function LibraryPanel({
  tree,
  treeError,
  onPickFolder,
  onRefreshTree,
  onFile,
  focusPath,
  onFocusDone,
  summaryTarget,
  onTargetChanged,
}: {
  /** undefined = loading, null = no folder yet — owned by ReaderMode. */
  tree: FileTreeNode | null | undefined;
  treeError: string | null;
  onPickFolder: () => void;
  onRefreshTree: () => void;
  onFile: (n: FileTreeNode & { path: string }) => void;
  focusPath: string | null;
  onFocusDone: () => void;
  /** Marked summary-target .docx path (starred in the tree). */
  summaryTarget: string | null;
  onTargetChanged: () => void;
}) {
  const actions = useTreeActions(onRefreshTree, onTargetChanged);
  return (
    <div className="fade-in space-y-4 px-4 py-4">
      <section>
        <div className="mb-1.5 flex items-center justify-between px-1">
          <h4 className="text-[11px] font-semibold uppercase tracking-wide text-mute">Library</h4>
          {/* Create at the workspace root; right-click any folder for more. */}
          {tree && (
            <span className="flex gap-1">
              <button
                type="button"
                title="New file in the workspace root"
                onClick={() => void actions.createFile("", "Untitled.docx")}
                className="rounded-md px-1.5 py-0.5 text-[11px] text-mute hover:bg-cream-100 hover:text-ink-900"
              >
                + File
              </button>
              <button
                type="button"
                title="New folder in the workspace root"
                onClick={() => void actions.createFolder("")}
                className="rounded-md px-1.5 py-0.5 text-[11px] text-mute hover:bg-cream-100 hover:text-ink-900"
              >
                + Folder
              </button>
            </span>
          )}
        </div>
        {treeError ? (
          <div className="rounded-lg border border-line-soft bg-cream-50 px-3 py-3">
            <p className="text-[12px] text-ink-700">Couldn’t load the library: {treeError}</p>
            <Button variant="soft" icon="refresh" className="mt-2 w-full" onClick={onRefreshTree}>
              Retry
            </Button>
          </div>
        ) : tree === undefined ? (
          <p className="px-1 py-2 text-[12px] text-mute">Loading library…</p>
        ) : tree === null ? (
          <div className="rounded-lg border border-line-soft bg-cream-50 px-3 py-3">
            <p className="text-[12px] text-ink-700">No library folder selected.</p>
            <Button variant="soft" icon="folder" className="mt-2 w-full" onClick={onPickFolder}>
              Choose folder
            </Button>
          </div>
        ) : (
          <>
            <p className="mb-1 truncate px-1 text-[11px] text-mute">{tree.label}</p>
            <FileTree
              node={tree}
              onFile={onFile}
              focusPath={focusPath}
              onFocusDone={onFocusDone}
              summaryTarget={summaryTarget}
              onTreeChanged={onRefreshTree}
              onTargetChanged={onTargetChanged}
            />
            <p className="mt-2 px-1 text-[10.5px] leading-relaxed text-mute">
              Right-click a file or folder to create, rename, or mark the ★ summary target.
            </p>
          </>
        )}
      </section>
      <Button variant="soft" icon="folder" className="w-full" onClick={onPickFolder}>
        Choose folder
      </Button>
    </div>
  );
}

/* ------------------------------------------------------------------ */
/* Sidebar shell                                                       */
/* ------------------------------------------------------------------ */

export default function ReaderSidebar({
  doc,
  onOpenFile,
  tree,
  treeError,
  onPickFolder,
  onRefreshTree,
  locatePath,
  onLocateDone,
  onShowInLibrary,
  currentRoot,
  summaryTarget,
  onSummaryTargetChanged,
}: {
  doc: Document | undefined;
  onOpenFile: (n: FileTreeNode & { path: string }) => void;
  /** Library tree owned by ReaderMode: undefined = loading, null = none. */
  tree: FileTreeNode | null | undefined;
  treeError: string | null;
  onPickFolder: () => void;
  onRefreshTree: () => void;
  /** Path token the Library tree should reveal (one-shot, see FileTree). */
  locatePath: string | null;
  onLocateDone: () => void;
  onShowInLibrary: (path: string, rootPath?: string) => void;
  /** Absolute workspace root for stale-tab detection (tasks.md 1.1 option b). */
  currentRoot: string | null;
  /** Marked summary-target .docx (starred in the tree). */
  summaryTarget: string | null;
  /** Re-fetch the target marker after the user marks a new one. */
  onSummaryTargetChanged: () => void;
}) {
  const { state, dispatch } = useApp();
  const ui = state.reader.ui;
  const view = ui.railView;

  const highlights = Object.entries(state.documents).flatMap(([docId, d]) =>
    d.highlights.map((h) => ({ h, docId, title: d.metadata.title })),
  );
  const saved = Object.values(state.sources).filter((s) => s.saved);

  return (
    <>
      {/* icon rail */}
      <div className="flex w-12 shrink-0 flex-col items-center border-r border-line bg-paper py-2.5">
        <div className="flex flex-col gap-1">
          {RAIL.map((r) => (
            <button
              key={r.id}
              type="button"
              title={r.label}
              onClick={() => {
                if (view === r.id && ui.sidebarOpen) {
                  dispatch({ type: "reader.ui", patch: { sidebarOpen: false } });
                } else {
                  dispatch({ type: "reader.ui", patch: { railView: r.id, sidebarOpen: true } });
                }
              }}
              className={cn(
                "flex h-9 w-9 items-center justify-center rounded-lg transition-colors",
                view === r.id && ui.sidebarOpen
                  ? "bg-iris-100 text-iris-700"
                  : "text-ink-400 hover:bg-cream-100 hover:text-ink-900",
              )}
            >
              <Icon name={r.icon} size={17} />
            </button>
          ))}
        </div>
        <div className="mt-auto">
          <button
            type="button"
            title={ui.sidebarOpen ? "Collapse panel" : "Expand panel"}
            onClick={() => dispatch({ type: "reader.ui", patch: { sidebarOpen: !ui.sidebarOpen } })}
            className="flex h-9 w-9 items-center justify-center rounded-lg text-ink-400 transition-colors hover:bg-cream-100 hover:text-ink-900"
          >
            <Icon name={ui.sidebarOpen ? "chevronsLeft" : "chevronsRight"} size={15} />
          </button>
        </div>
      </div>

      {/* secondary panel */}
      {ui.sidebarOpen && (
        <div className="flex h-full w-[250px] shrink-0 flex-col border-r border-line bg-paper">
          <div className="min-h-0 flex-1 overflow-y-auto">
            {view === "documents" &&
              (doc ? (
                doc.kind === "summary" ? (
                  <SummarySourcesPanel doc={doc} />
                ) : (
                  <DocumentInfoPanel doc={doc} onShowInLibrary={onShowInLibrary} currentRoot={currentRoot} />
                )
              ) : (
                <EmptyState icon="fileText" title="No document open" hint="Open a document from the tabs bar." />
              ))}

            {view === "library" && (
              <LibraryPanel
                tree={tree}
                treeError={treeError}
                onPickFolder={onPickFolder}
                onRefreshTree={onRefreshTree}
                onFile={onOpenFile}
                focusPath={locatePath}
                onFocusDone={onLocateDone}
                summaryTarget={summaryTarget}
                onTargetChanged={onSummaryTargetChanged}
              />
            )}

            {view === "bookmarks" &&
              (saved.length ? (
                <div className="fade-in px-4 py-4">
                  <h4 className="mb-2 px-1 text-[11px] font-semibold uppercase tracking-wide text-mute">
                    Saved sources
                  </h4>
                  <ul className="space-y-1.5">
                    {saved.map((s) => (
                      <li key={s.id} className="flex items-center gap-2 rounded-md border border-line-soft bg-cream-50 px-3 py-2">
                        <Icon name="bookmark" size={13} className="shrink-0 text-iris-600" />
                        <div className="min-w-0">
                          <p className="truncate text-[12px] font-medium text-ink-900">{s.title}</p>
                          <p className="truncate text-[10.5px] text-mute">{s.origin}</p>
                        </div>
                      </li>
                    ))}
                  </ul>
                </div>
              ) : (
                <EmptyState icon="bookmark" title="No saved sources" hint="Use “Save” on any AI Browse result." />
              ))}

            {view === "notes" &&
              (highlights.length ? (
                <div className="fade-in px-4 py-4">
                  <h4 className="mb-2 px-1 text-[11px] font-semibold uppercase tracking-wide text-mute">Saved notes</h4>
                  <ul className="space-y-2">
                    {highlights.map(({ h, title }) => (
                      <li key={h.id} className="rounded-md border border-line-soft bg-cream-50 px-3 py-2">
                        <p className="text-[11px] font-medium text-iris-700">{title}</p>
                        <p className="mt-0.5 text-[12px] leading-snug text-ink-500">{truncate(h.text, 120)}</p>
                      </li>
                    ))}
                  </ul>
                </div>
              ) : (
                <EmptyState icon="note" title="No notes yet" hint="Select text in a source document and choose “Save note”." />
              ))}



            {view === "settings" && (
              <div className="fade-in px-4 py-4">
                <EmptyState icon="settings" title="Workspace settings" hint="Global settings (AI provider, storage, browser engine) live in the settings dialog." />
                <Button variant="soft" icon="settings" className="mx-auto" onClick={() => dispatch({ type: "settings.set", open: true })}>
                  Open settings
                </Button>
              </div>
            )}
          </div>
        </div>
      )}
    </>
  );
}
