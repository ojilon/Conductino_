/**
 * Reader subtabs. Open documents stay mounted (hidden) in ReaderMode,
 * so scroll position, highlights and edits survive tab switches.
 */

import { useApp, pendingChangesFor } from "../../state/appState";
import { Icon, type IconName } from "../../components/icons";
import { Menu, MenuItem } from "../../components/ui";
import { cn } from "../../utils/cn";
import { uid } from "../../utils/helpers";
import { fileTreeMock } from "../../mock/data";
import type { FileTreeNode } from "../../types/domain";

export function flatFiles(node: FileTreeNode, prefix = ""): (FileTreeNode & { path: string })[] {
  const out: (FileTreeNode & { path: string })[] = [];
  const walk = (n: FileTreeNode, p: string) => {
    if (n.kind === "file") out.push({ ...n, path: p ? `${p}/${n.label}` : n.label });
    n.children?.forEach((c) => walk(c, p ? `${p}/${n.label}` : n.label));
  };
  node.children?.forEach((c) => walk(c, prefix));
  return out;
}

function docIcon(docKind: "source" | "summary"): IconName {
  if (docKind === "summary") return "layers";
  return "fileText";
}

export default function ReaderTabs({ onOpenFile }: { onOpenFile: (node: FileTreeNode & { path: string }) => void }) {
  const { state, dispatch } = useApp();
  const session = state.reader.session;
  const files = flatFiles(fileTreeMock);

  return (
    <div className="flex items-center gap-1 border-b border-line bg-cream-50/60 px-3 pt-2">
      <div className="flex min-w-0 flex-1 items-end gap-1 overflow-x-auto">
        {session.tabIds.map((tabId) => {
          const tab = state.reader.tabs[tabId];
          if (!tab) return null;
          const doc = state.documents[tab.documentId];
          const active = tabId === session.activeTabId;
          const pending = pendingChangesFor(state, tab.documentId).length;
          return (
            <div
              key={tabId}
              onClick={() => dispatch({ type: "reader.tab.select", tabId })}
              className={cn(
                "group flex shrink-0 cursor-pointer items-center gap-1.5 rounded-t-lg border border-b-0 px-3.5 py-2 text-[12.5px] transition-colors",
                active
                  ? "border-line bg-white font-medium text-ink-900"
                  : "border-transparent text-ink-500 hover:bg-cream-100/70",
              )}
            >
              <Icon name={docIcon(doc?.kind ?? "source")} size={13} className={active ? "text-iris-600" : "opacity-50"} />
              <span className="max-w-[150px] truncate">{tab.label}</span>
              {pending > 0 && (
                <span className="flex h-4 min-w-4 items-center justify-center rounded-full bg-hay-100 px-1 text-[10px] font-bold text-hay-700">
                  {pending}
                </span>
              )}
              <button
                type="button"
                title="Close document"
                className="rounded p-0.5 text-ink-400 opacity-0 transition-opacity hover:bg-cream-100 hover:text-ink-900 group-hover:opacity-100"
                onClick={(e) => {
                  e.stopPropagation();
                  dispatch({ type: "reader.tab.close", tabId });
                }}
              >
                <Icon name="x" size={11} />
              </button>
            </div>
          );
        })}
      </div>

      <div className="flex shrink-0 items-center gap-1 pb-1.5">
        <Menu
          align="right"
          width={250}
          button={(open) => (
            <button
              type="button"
              className={cn(
                "flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-[12.5px] transition-colors",
                open ? "bg-cream-100 text-ink-900" : "text-ink-500 hover:bg-cream-100 hover:text-ink-900",
              )}
            >
              <Icon name="plus" size={13} /> Open document
            </button>
          )}
        >
          {(close) => (
            <>
              <p className="px-3 pb-1 pt-1.5 text-[10.5px] font-semibold uppercase tracking-wide text-mute">
                From workspace
              </p>
              {files.map((f) => (
                <MenuItem
                  key={f.id}
                  icon={f.ext === "pdf" ? "fileText" : "file"}
                  label={f.label}
                  hint={f.path.split("/")[0]}
                  onClick={() => {
                    onOpenFile(f);
                    close();
                  }}
                />
              ))}
            </>
          )}
        </Menu>
        <button
          type="button"
          title={state.reader.ui.aiPanelOpen ? "Hide AI panel" : "Show AI panel"}
          onClick={() =>
            dispatch({ type: "reader.ui", patch: { aiPanelOpen: !state.reader.ui.aiPanelOpen } })
          }
          className={cn(
            "flex h-7 w-7 items-center justify-center rounded-md transition-colors",
            state.reader.ui.aiPanelOpen
              ? "bg-iris-100 text-iris-700"
              : "text-ink-400 hover:bg-cream-100 hover:text-ink-900",
          )}
        >
          <Icon name="sparkles" size={15} />
        </button>
      </div>
    </div>
  );
}

export function labelForPath(path: string): string {
  const name = path.split("/").pop() ?? path;
  return name
    .replace(/\.(pdf|docx|html|txt)$/i, "")
    .replace(/[_-]+/g, " ")
    .replace(/\b\w/g, (c) => c.toUpperCase());
}

export function newTabId(): string {
  return uid("rt");
}
