/**
 * SkillsPanel — the AI's instruction folder, browsable like normal files.
 *
 * Opens from the AI panel ("Skills" button). It shows the INTERNAL skills/
 * folder (backend/skills in dev, app-data skills installed) with the same
 * tree interaction as the library, so the user can:
 *
 *   - read every instruction file (bundled skills + the AI user guide),
 *   - create custom folders, e.g. skills/my_instructions/, and write their
 *     own .md instructions for the AI,
 *   - edit + save any file (the AI picks bundled ones up at startup, and
 *     can read ANY of them on demand with the read_skill tool — mention
 *     "@skills/my_instructions/rules.md" in chat to point it there),
 *   - try the AI's sandboxed shell (same "shell" tool the AI uses: runs in
 *     the open research folder, 10s timeout, capped output).
 */

import { useCallback, useEffect, useState } from "react";
import { useApp } from "../../state/appState";
import {
  createSkillEntry,
  listSkillsTree,
  readSkillFile,
  runShellCommand,
  writeSkillFile,
} from "../../services/backend";
import type { FileTreeNode } from "../../types/domain";
import { Icon } from "../../components/icons";
import { Button, Modal } from "../../components/ui";
import { cn } from "../../utils/cn";

/* ------------------------------------------------------------------ */
/* Skills tree (same shape as the library tree, smaller)               */
/* ------------------------------------------------------------------ */

function SkillsTree({
  node,
  selected,
  onSelect,
}: {
  node: FileTreeNode;
  selected: string | null;
  onSelect: (path: string) => void;
}) {
  const [open, setOpen] = useState<Record<string, boolean>>({});

  // Start with every folder open (skills trees are small).
  useEffect(() => {
    const initial: Record<string, boolean> = {};
    const mark = (n: FileTreeNode) => {
      if (n.kind === "folder") {
        initial[n.path ?? ""] = true;
        n.children?.forEach(mark);
      }
    };
    mark(node);
    setOpen(initial);
  }, [node]);

  const walk = (n: FileTreeNode, depth: number): React.ReactNode => {
    const token = n.path ?? "";
    if (n.kind === "folder" && depth >= 0 && token === "" && n !== node) return null;
    return (
      <div key={n.id}>
        <button
          type="button"
          style={{ paddingLeft: `${6 + depth * 12}px` }}
          onClick={() => {
            if (n.kind === "folder") setOpen((o) => ({ ...o, [token]: !o[token] }));
            else onSelect(token);
          }}
          className={cn(
            "flex w-full items-center gap-1.5 rounded-md py-[4px] pr-2 text-left text-[12px] transition-colors",
            selected === token && n.kind === "file"
              ? "bg-iris-100/70 text-iris-700"
              : "text-ink-700 hover:bg-cream-100",
          )}
        >
          {n.kind === "folder" ? (
            <>
              <Icon name="chevronRight" size={10} className={cn("text-mute transition-transform", open[token] && "rotate-90")} />
              <Icon name="folder" size={12} className="shrink-0 text-mute" />
            </>
          ) : (
            <>
              <span className="w-[10px]" />
              <Icon name="fileText" size={12} className="shrink-0 text-iris-500" />
            </>
          )}
          <span className="truncate">{n.label}</span>
        </button>
        {n.kind === "folder" && open[token] && n.children?.map((c) => walk(c, depth + 1))}
      </div>
    );
  };

  return <div>{node.children?.map((c) => walk(c, 0)) ?? <p className="px-2 py-1 text-[11.5px] text-mute">(empty)</p>}</div>;
}

/* ------------------------------------------------------------------ */
/* Panel                                                               */
/* ------------------------------------------------------------------ */

export default function SkillsPanel({ onClose }: { onClose: () => void }) {
  const { dispatch } = useApp();
  const [tree, setTree] = useState<FileTreeNode | null | undefined>(undefined);
  const [selected, setSelected] = useState<string | null>(null);
  const [content, setContent] = useState("");
  const [dirty, setDirty] = useState(false);
  const [shellCmd, setShellCmd] = useState("");
  const [shellOut, setShellOut] = useState<string[]>([]);
  const [shellBusy, setShellBusy] = useState(false);

  const refresh = useCallback(async () => {
    setTree(await listSkillsTree());
  }, []);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  const toast = (message: string) => dispatch({ type: "toast", message });

  const openFile = async (path: string) => {
    if (dirty && !window.confirm("Discard unsaved changes to the current file?")) return;
    const text = await readSkillFile(path);
    if (text === null) {
      toast("Reading skills files needs the desktop app — run `wails dev`.");
      return;
    }
    setSelected(path);
    setContent(text);
    setDirty(false);
  };

  const save = async () => {
    if (!selected) return;
    const ok = await writeSkillFile(selected, content);
    if (ok) {
      setDirty(false);
      toast(`Saved skills/${selected}`);
      void refresh();
    } else {
      toast("Saving needs the desktop app — run `wails dev`.");
    }
  };

  const newFile = async () => {
    const path = window.prompt(
      "New instruction file (path inside skills/, e.g. my_instructions/reading-rules.md):",
      "my_instructions/my-rules.md",
    );
    if (!path?.trim()) return;
    const ok = await createSkillEntry(path.trim(), false);
    if (ok) {
      await refresh();
      void openFile(path.trim());
      toast(`Created skills/${path.trim()} — mention it in chat as @skills/${path.trim()}`);
    } else {
      toast("Could not create the file (needs the desktop app, .md/.txt only, no overwriting).");
    }
  };

  const newFolder = async () => {
    const path = window.prompt("New folder (path inside skills/):", "my_instructions");
    if (!path?.trim()) return;
    const ok = await createSkillEntry(path.trim(), true);
    if (ok) {
      await refresh();
      toast(`Created skills/${path.trim()}/`);
    } else {
      toast("Could not create the folder (needs the desktop app).");
    }
  };

  const runShell = async () => {
    const cmd = shellCmd.trim();
    if (!cmd || shellBusy) return;
    setShellBusy(true);
    setShellOut((prev) => [...prev, `$ ${cmd}`]);
    const out = await runShellCommand(cmd);
    setShellOut((prev) => [...prev, out === null ? "(shell needs the desktop app — run `wails dev`)" : out || "(no output)"]);
    setShellBusy(false);
    setShellCmd("");
  };

  return (
    <Modal
      title="AI skills & instructions"
      subtitle="The AI's instruction folder — read it, extend it, and point the AI at your own files with @skills/… in chat."
      onClose={onClose}
      width={880}
    >
      <div className="grid grid-cols-[240px_1fr] gap-4">
        {/* left: tree + create buttons */}
        <div className="flex min-h-[320px] flex-col rounded-lg border border-line-soft bg-cream-50 p-2">
          <div className="mb-2 flex items-center justify-between px-1">
            <h4 className="text-[11px] font-semibold uppercase tracking-wide text-mute">skills/</h4>
            <span className="flex gap-1">
              <button type="button" title="New file" onClick={() => void newFile()} className="rounded-md px-1.5 py-0.5 text-[11px] text-mute hover:bg-cream-100 hover:text-ink-900">
                + File
              </button>
              <button type="button" title="New folder" onClick={() => void newFolder()} className="rounded-md px-1.5 py-0.5 text-[11px] text-mute hover:bg-cream-100 hover:text-ink-900">
                + Folder
              </button>
            </span>
          </div>
          <div className="min-h-0 flex-1 overflow-y-auto">
            {tree === undefined ? (
              <p className="px-2 py-1 text-[11.5px] text-mute">Loading skills folder…</p>
            ) : tree === null ? (
              <p className="px-2 py-1 text-[11.5px] text-mute">
                Needs the desktop app — run <code className="rounded bg-cream-100 px-1">wails dev</code>.
              </p>
            ) : (
              <SkillsTree node={tree} selected={selected} onSelect={(p) => void openFile(p)} />
            )}
          </div>
        </div>

        {/* right: editor */}
        <div className="flex min-h-[320px] flex-col gap-2">
          {selected ? (
            <>
              <div className="flex items-center justify-between gap-2">
                <p className="truncate text-[12px] font-medium text-ink-900">skills/{selected}</p>
                <span className="flex items-center gap-2">
                  {dirty && <span className="text-[11px] text-amber-600">unsaved</span>}
                  <Button variant="soft" onClick={() => void save()} disabled={!dirty}>
                    Save
                  </Button>
                </span>
              </div>
              <textarea
                value={content}
                onChange={(e) => {
                  setContent(e.target.value);
                  setDirty(true);
                }}
                spellCheck={false}
                className="min-h-[260px] flex-1 resize-none rounded-lg border border-line bg-white p-3 font-mono text-[12px] leading-relaxed text-ink-800 outline-none focus:border-iris-300"
              />
              <p className="text-[11px] leading-relaxed text-mute">
                Bundled skills use frontmatter (--- skill: name … ---). Custom instructions in your own subfolder
                can be plain markdown — tell the AI to read them with{" "}
                <code className="rounded bg-cream-100 px-1">@skills/{selected}</code> in chat.
              </p>
            </>
          ) : (
            <div className="flex flex-1 flex-col items-center justify-center gap-2 rounded-lg border border-dashed border-line px-6 py-10 text-center">
              <Icon name="sparkles" size={18} className="text-iris-500" />
              <p className="text-[12.5px] text-ink-700">Pick an instruction file to read or edit it.</p>
              <p className="max-w-[380px] text-[11.5px] leading-relaxed text-mute">
                Start with <code className="rounded bg-cream-100 px-1">app-guide.md</code> — the AI's internal user
                guide (tools, mirrors, summary target, shell). Create <code className="rounded bg-cream-100 px-1">my_instructions/</code>{" "}
                for your own workflows.
              </p>
            </div>
          )}
        </div>
      </div>

      {/* bottom: the AI's sandboxed shell (same tool the AI calls) */}
      <div className="mt-4 rounded-lg border border-line-soft bg-ink-900 p-3">
        <div className="mb-2 flex items-center justify-between">
          <h4 className="text-[11px] font-semibold uppercase tracking-wide text-cream-100/70">
            Sandbox shell <span className="normal-case text-cream-100/40">— the AI's "shell" tool, cwd = open research folder, 10s timeout</span>
          </h4>
          {shellOut.length > 0 && (
            <button type="button" className="text-[11px] text-cream-100/50 hover:text-cream-100" onClick={() => setShellOut([])}>
              clear
            </button>
          )}
        </div>
        <pre className="mb-2 max-h-[140px] min-h-[36px] overflow-y-auto whitespace-pre-wrap font-mono text-[11.5px] leading-relaxed text-moss-200">
          {shellOut.length ? shellOut.join("\n") : "# try: ls -la   ·   wc -w notes.txt   ·   grep -ri \"method\" ."}
        </pre>
        <div className="flex gap-2">
          <input
            value={shellCmd}
            onChange={(e) => setShellCmd(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter") void runShell();
            }}
            placeholder="Run a basic command…"
            spellCheck={false}
            className="flex-1 rounded-md border border-cream-100/20 bg-ink-900/60 px-2.5 py-1.5 font-mono text-[12px] text-cream-50 placeholder:text-cream-100/30 focus:border-iris-400 focus:outline-none"
          />
          <button
            type="button"
            onClick={() => void runShell()}
            disabled={shellBusy || !shellCmd.trim()}
            className={cn(
              "rounded-md px-3 py-1.5 text-[12px] font-medium",
              shellBusy || !shellCmd.trim() ? "bg-cream-100/10 text-cream-100/30" : "bg-iris-600 text-white hover:bg-iris-500",
            )}
          >
            {shellBusy ? "Running…" : "Run"}
          </button>
        </div>
      </div>
    </Modal>
  );
}
