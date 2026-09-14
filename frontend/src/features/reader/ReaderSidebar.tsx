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
import { fileTreeMock } from "../../mock/data";
import type { Document, FileTreeNode, RailView } from "../../types/domain";
import { Icon, type IconName } from "../../components/icons";
import { Button, EmptyState } from "../../components/ui";
import { cn } from "../../utils/cn";
import { truncate } from "../../utils/helpers";

const RAIL: { id: RailView; icon: IconName; label: string }[] = [
  { id: "documents", icon: "fileText", label: "Document" },
  { id: "bookmarks", icon: "bookmark", label: "Bookmarks" },
  { id: "notes", icon: "note", label: "Notes" },
  { id: "library", icon: "book", label: "Library" },
  { id: "settings", icon: "settings", label: "Settings" },
];

/* ------------------------------------------------------------------ */
/* File tree (mock of backend.filesystem.listRoot)                     */
/* ------------------------------------------------------------------ */

function FileTree({
  node,
  onFile,
}: {
  node: FileTreeNode;
  onFile: (n: FileTreeNode & { path: string }) => void;
}) {
  const [open, setOpen] = useState<Record<string, boolean>>({ "ft-papers": true, "ft-notes": true });
  const [activeId, setActiveId] = useState<string>("ft-mitchell");

  const walk = (n: FileTreeNode, path: string, d: number) => {
    const prefix = path ? `${path}/${n.label}` : n.label;
    return (
      <div key={n.id}>
        <button
          type="button"
          style={{ paddingLeft: `${8 + d * 14}px` }}
          onClick={() => {
            if (n.kind === "folder") setOpen((o) => ({ ...o, [n.id]: !o[n.id] }));
            else {
              setActiveId(n.id);
              onFile({ ...n, path: prefix });
            }
          }}
          className={cn(
            "flex w-full items-center gap-1.5 rounded-md py-[5px] pr-2 text-left text-[12.5px] transition-colors",
            activeId === n.id ? "bg-iris-100/70 text-iris-700" : "text-ink-700 hover:bg-cream-100",
          )}
        >
          {n.kind === "folder" ? (
            <>
              <Icon name="chevronRight" size={11} className={cn("text-mute transition-transform", open[n.id] && "rotate-90")} />
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
        </button>
        {n.kind === "folder" && open[n.id] && n.children?.map((c) => walk(c, prefix, d + 1))}
      </div>
    );
  };

  return <div>{node.children?.map((c) => walk(c, "", 0))}</div>;
}

/* ------------------------------------------------------------------ */
/* Document info + TOC                                                 */
/* ------------------------------------------------------------------ */

function DocumentInfoPanel({ doc, onOpenFile }: { doc: Document; onOpenFile: (n: FileTreeNode & { path: string }) => void }) {
  const { dispatch } = useApp();
  const [tree, setTree] = useState<FileTreeNode>(fileTreeMock);
  const [tocActive, setTocActive] = useState<string | null>(null);
  const meta = doc.metadata;
  const headings = doc.blocks.filter((b) => b.type === "heading" && b.level === 2);

  useEffect(() => {
    let live = true;
    backend.filesystem.listRoot().then((t) => live && setTree(t));
    return () => {
      live = false;
    };
  }, []);

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

      {/* file tree */}
      <section>
        <h4 className="mb-1.5 px-1 text-[11px] font-semibold uppercase tracking-wide text-mute">Files</h4>
        <FileTree node={tree} onFile={onOpenFile} />
        <Button
          variant="soft"
          icon="folder"
          className="mt-3 w-full"
          onClick={() => {
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
/* Sidebar shell                                                       */
/* ------------------------------------------------------------------ */

export default function ReaderSidebar({
  doc,
  onOpenFile,
}: {
  doc: Document | undefined;
  onOpenFile: (n: FileTreeNode & { path: string }) => void;
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
                  <DocumentInfoPanel doc={doc} onOpenFile={onOpenFile} />
                )
              ) : (
                <EmptyState icon="fileText" title="No document open" hint="Open a document from the tabs bar." />
              ))}

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

            {view === "library" && (
              <EmptyState
                icon="book"
                title="Local document library"
                hint="Listed by the Go backend (backend/services/workspace.go). Browse the workspace via the Files tree."
              />
            )}

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
