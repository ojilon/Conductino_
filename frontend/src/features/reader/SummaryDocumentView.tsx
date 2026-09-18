/**
 * Summary document view — EDITABLE research summary via Slate.
 *
 * Phase 7: Slate is a transient view only. Canonical body remains
 * DocumentBlock[]; toSlate / fromSlate are the sole conversion boundary.
 * Pending AI inserts/modifies still render as cards/decorations until
 * accept/reject merges them into blocks and re-hydrates the editor.
 *
 * Phase 8: Save DOCX writes blocks to summaries/*.docx under the workspace.
 */

import { useCallback, useEffect, useMemo, useRef, useState, type ReactNode } from "react";
import {
  createEditor,
  type Descendant,
  type BaseEditor,
  Editor,
  Node,
  Range as SlateRange,
  type NodeEntry,
} from "slate";
import {
  Slate,
  Editable,
  withReact,
  type RenderElementProps,
  type RenderLeafProps,
  type ReactEditor,
} from "slate-react";
import { withHistory, type HistoryEditor } from "slate-history";
import { useApp, pendingChangesFor } from "../../state/appState";
import { useAIRunners } from "../../state/aiController";
import { writeSummaryDOCX } from "../../services/backend";
import type { Document, DocumentBlock, DocumentChange, ID } from "../../types/domain";
import { Icon } from "../../components/icons";
import { Badge } from "../../components/ui";
import { cn } from "../../utils/cn";
import {
  toSlate,
  fromSlate,
  sameBlocks,
  type SlateElement as ConductinoSlateElement,
  type SlateText as ConductinoSlateText,
} from "./slateAdapter";

declare module "slate" {
  interface CustomTypes {
    Editor: BaseEditor & ReactEditor & HistoryEditor;
    Element: ConductinoSlateElement;
    Text: ConductinoSlateText;
  }
}

function InsertCard({ change, sourceTitle }: { change: DocumentChange; sourceTitle: string }) {
  return (
    <div className="fade-in my-4 rounded-lg border border-hay-200 bg-hay-50 px-4 py-3.5">
      <div className="mb-1.5 flex items-center gap-2">
        <Icon name="sparkles" size={13} className="text-hay-600" />
        <span className="text-[11.5px] font-semibold uppercase tracking-wide text-hay-600">AI change</span>
        <Badge tone="hay">pending</Badge>
        <span className="ml-auto flex items-center gap-1 text-[10.5px] text-mute">
          <Icon name="link" size={10} /> {sourceTitle}
        </span>
      </div>
      <p className="font-serif text-[15px] leading-[1.75] text-[#4c7a48]">{change.newContent}</p>
    </div>
  );
}

function Element({ attributes, children, element }: RenderElementProps) {
  const el = element as ConductinoSlateElement;
  const pending = Boolean(el.changeId);

  if (el.type === "heading") {
    const level = el.level ?? 2;
    if (level === 1) {
      return (
        <h1
          {...attributes}
          className={cn(
            "mb-7 font-serif text-[26px] font-bold leading-tight text-ink-900",
            pending && "rounded-md border border-hay-200 bg-hay-50/60 px-2",
          )}
          data-block-id={el.blockId}
        >
          {children}
        </h1>
      );
    }
    return (
      <h2
        {...attributes}
        className={cn(
          "mt-8 font-serif text-[19px] font-bold text-ink-900",
          pending && "rounded-md border border-hay-200 bg-hay-50/60 px-2",
        )}
        data-block-id={el.blockId}
      >
        {children}
      </h2>
    );
  }

  if (el.type === "list") {
    return (
      <ul
        {...attributes}
        className={cn("my-4 list-disc pl-6 doc-body", pending && "rounded-md border border-hay-200 bg-hay-50/60")}
        data-block-id={el.blockId}
      >
        {children}
      </ul>
    );
  }

  return (
    <p
      {...attributes}
      className={cn(
        "my-4 doc-body",
        pending && "rounded-md border border-hay-200 bg-hay-50/40 px-2 py-1",
      )}
      data-block-id={el.blockId}
    >
      {children}
    </p>
  );
}

function Leaf({ attributes, children, leaf }: RenderLeafProps) {
  const l = leaf as ConductinoSlateText;
  let node: ReactNode = children;
  if (l.strong) node = <strong>{node}</strong>;
  if (l.em) node = <em>{node}</em>;
  if (l.highlightId) {
    node = (
      <mark className="rounded-[2px] bg-sky-100/80 px-0.5 text-ink-900" data-highlight-id={l.highlightId}>
        {node}
      </mark>
    );
  }
  return <span {...attributes}>{node}</span>;
}

/** Locate a fragment inside element text (first occurrence). */
function fragmentRange(text: string, fragment: string): { start: number; end: number } | null {
  const frag = fragment.trim();
  if (!frag) return null;
  const idx = text.indexOf(frag.length > 120 ? frag.slice(0, 120) : frag);
  if (idx < 0) return null;
  return { start: idx, end: idx + Math.min(frag.length, 120) };
}

function SummarySlateEditor({
  docId,
  blocks,
  modifies,
  externalKey,
}: {
  docId: ID;
  blocks: DocumentBlock[];
  /** Pending modify changes by block id — rendered as inline decorations. */
  modifies: Map<ID, DocumentChange>;
  externalKey: string;
}) {
  const { dispatch } = useApp();
  const { runRevise } = useAIRunners();
  const editor = useMemo(() => withHistory(withReact(createEditor())), [externalKey]);
  const initial = useMemo(() => toSlate(blocks), [externalKey]); // eslint-disable-line react-hooks/exhaustive-deps
  const [focusId, setFocusId] = useState<ID | null>(null);
  const focused = (focusId && modifies.get(focusId)) || null;

  const onChange = useCallback(
    (next: Descendant[]) => {
      const nextBlocks = fromSlate(next);
      if (sameBlocks(nextBlocks, blocks)) return;

      dispatch({ type: "doc.blocks.replace", documentId: docId, blocks: nextBlocks });

      const first = nextBlocks[0];
      if (first?.type === "heading" && first.level === 1) {
        const title = first.segments.map((s) => s.text).join("").trim();
        if (title) dispatch({ type: "doc.title", documentId: docId, text: title });
      }
    },
    [blocks, dispatch, docId],
  );

  // Inline pending-modify decorations (issue 28): the targeted span inside
  // the committed block gets a pendingChangeId mark instead of a separate
  // card above the editor. fromSlate drops the mark, so user edits can
  // never persist decoration state into canonical blocks.
  const decorate = useCallback(
    ([node, path]: NodeEntry) => {
      const ranges: SlateRange[] = [];
      if (!("blockId" in node)) return ranges;
      const el = node as ConductinoSlateElement;
      const change = modifies.get(el.blockId);
      if (!change || change.type !== "modify") return ranges;
      const text = Node.string(node);
      const span = fragmentRange(text, change.highlightFragment || change.oldContent);
      if (!span) return ranges;
      ranges.push({
        anchor: { path, offset: span.start },
        focus: { path, offset: span.end },
        pendingChangeId: change.id,
      } as SlateRange & { pendingChangeId: ID });
      return ranges;
    },
    [modifies],
  );

  const renderElement = useCallback((props: RenderElementProps) => <Element {...props} />, []);
  const renderLeaf = useCallback(
    (props: RenderLeafProps) => {
      const l = props.leaf as ConductinoSlateText;
      if (l.pendingChangeId) {
        const cid = l.pendingChangeId;
        return (
          <span
            {...props.attributes}
            onClick={(e) => {
              e.stopPropagation();
              setFocusId(cid);
            }}
            title="AI-proposed revision — click to review"
            className="cursor-pointer rounded-[3px] bg-hay-100 px-0.5 underline decoration-hay-300 decoration-2 underline-offset-4"
            data-change-id={cid}
          >
            {props.children}
          </span>
        );
      }
      return <Leaf {...props} />;
    },
    [],
  );

  const decide = (status: "accepted" | "rejected") => {
    if (!focused) return;
    dispatch({ type: "change.decide", id: focused.id, status });
    dispatch({ type: "toast", message: status === "accepted" ? "Change accepted" : "Change rejected" });
    setFocusId(null);
  };

  return (
    <div>
      {focused && (
        <div className="fade-in mb-2 rounded-lg border border-hay-200 bg-hay-50 px-3 py-2.5">
          <p className="mb-1 flex items-center gap-1.5 text-[11px] font-semibold uppercase tracking-wide text-hay-600">
            <Icon name="sparkles" size={12} /> Proposed revision
            <Badge tone="hay">pending</Badge>
            <button
              type="button"
              className="ml-auto font-normal normal-case tracking-normal text-mute hover:text-ink-700"
              onClick={() => setFocusId(null)}
            >
              dismiss
            </button>
          </p>
          <p className="font-serif text-[13.5px] leading-relaxed text-ink-700">{focused.newContent}</p>
          <div className="mt-2 flex gap-1.5">
            <button
              type="button"
              onClick={() => decide("accepted")}
              className="rounded-md bg-moss-600 px-2.5 py-1 text-[11.5px] font-medium text-white hover:bg-moss-700"
            >
              Accept
            </button>
            <button
              type="button"
              onClick={() => decide("rejected")}
              className="rounded-md border border-line bg-white px-2.5 py-1 text-[11.5px] font-medium text-ink-700 hover:border-rose-300"
            >
              Reject
            </button>
            <button
              type="button"
              onClick={() => {
                runRevise(focused.id);
                setFocusId(null);
              }}
              className="rounded-md border border-line bg-white px-2.5 py-1 text-[11.5px] font-medium text-ink-700 hover:border-iris-300"
            >
              Revise again
            </button>
          </div>
        </div>
      )}
      <Slate key={externalKey} editor={editor} initialValue={initial} onChange={onChange}>
        <Editable
          decorate={decorate}
          renderElement={renderElement}
          renderLeaf={renderLeaf}
          spellCheck={false}
          className="outline-none min-h-[12rem]"
          placeholder="Start writing the summary…"
          onKeyDown={(event) => {
            if (!event.metaKey && !event.ctrlKey) return;
            if (event.key === "b") {
              event.preventDefault();
              const marks = Editor.marks(editor) as ConductinoSlateText | null;
              if (marks?.strong) Editor.removeMark(editor, "strong");
              else Editor.addMark(editor, "strong", true);
            }
            if (event.key === "i") {
              event.preventDefault();
              const marks = Editor.marks(editor) as ConductinoSlateText | null;
              if (marks?.em) Editor.removeMark(editor, "em");
              else Editor.addMark(editor, "em", true);
            }
          }}
        />
      </Slate>
    </div>
  );
}

export default function SummaryDocumentView({ doc }: { doc: Document }) {
  const { state, dispatch } = useApp();
  const pending = pendingChangesFor(state, doc.id);
  const changeByBlock = new Map(pending.map((c) => [c.blockId, c]));

  const pendingInserts = pending.filter((c) => c.type === "insert");
  const pendingDeletes = pending.filter((c) => c.type === "delete");
  const pendingModifies = new Map(
    pending.filter((c) => c.type === "modify").map((c) => [c.blockId, c] as const),
  );

  const editableBlocks = doc.blocks.filter((b) => {
    const ch = changeByBlock.get(b.id);
    return !(ch && ch.type === "insert" && ch.status === "pending");
  });

  const saveDocx = async () => {
    const blocksJSON = JSON.stringify({ blocks: doc.blocks });
    try {
      const path = await writeSummaryDOCX(blocksJSON);
      if (path) {
        dispatch({ type: "toast", message: `Saved summary DOCX · ${path}` });
      } else {
        dispatch({ type: "toast", message: "DOCX save needs the desktop app (Wails)." });
      }
    } catch (e) {
      dispatch({
        type: "toast",
        message: `DOCX save failed: ${e instanceof Error ? e.message : String(e)}`,
      });
    }
  };

  // Gated autosave (issues 16+29): the user's accept IS the gate. Debounced
  // 3s after committed blocks settle, only accepted/user content is written
  // — pending-insert blocks (changeId still pending) are excluded, and
  // pending modify/delete never touch blocks until accept. Silent on
  // success; honest toast only on failure. No-op in browser/mock mode.
  const lastSaved = useRef<string>("");
  const [autoState, setAutoState] = useState<"idle" | "saving" | "saved">("idle");
  useEffect(() => {
    const committed = doc.blocks.filter((b) => {
      const ch = state.changes[b.changeId ?? ""];
      return !(ch && ch.type === "insert" && ch.status === "pending");
    });
    const snapshot = JSON.stringify({ blocks: committed });
    if (snapshot === lastSaved.current) return;
    const t = setTimeout(() => {
      lastSaved.current = snapshot;
      setAutoState("saving");
      writeSummaryDOCX(snapshot)
        .then((path) => setAutoState(path ? "saved" : "idle"))
        .catch((e: unknown) => {
          setAutoState("idle");
          dispatch({
            type: "toast",
            message: `Autosave failed: ${e instanceof Error ? e.message : String(e)}`,
          });
        });
    }, 3000);
    return () => clearTimeout(t);
  }, [doc.blocks, state.changes, dispatch]);

  return (
    <div className="mx-auto max-w-[700px] px-8 py-6">
      <div className="mb-4 flex items-center justify-between border-b border-line pb-2.5 text-[12px] text-mute">
        <span className="flex items-center gap-3">
          <span>Editable summary · Slate · {doc.sourceIds?.length ?? 0} sources</span>
          <button
            type="button"
            onClick={() => void saveDocx()}
            className="rounded-md border border-line bg-white px-2 py-0.5 text-[11px] font-medium text-ink-700 hover:border-moss-300 hover:text-moss-700"
            title="Write summary as .docx under the workspace (summaries/)"
          >
            Save DOCX
          </button>
        </span>
        <span className="flex items-center gap-1.5">
          <span className={cn("h-1.5 w-1.5 rounded-full", pending.length ? "bg-hay-300" : "bg-moss-200")} />
          {pending.length
            ? `${pending.length} pending change${pending.length === 1 ? "" : "s"}`
            : autoState === "saving"
              ? "saving…"
              : autoState === "saved"
                ? "all changes reviewed · saved"
                : "all changes reviewed"}
        </span>
      </div>

      {/* Pending modifies render inline as editor decorations (click the
          underlined span to review) — no cards here. Deletes and inserts
          below still use cards. */}

      {pendingDeletes.map((change) => (
        <div key={`del-${change.id}`} className="fade-in mb-2 rounded-lg border border-rose-200 bg-rose-50/60 px-4 py-3">
          <div className="mb-1 flex items-center gap-2 text-[11px] text-rose-600">
            <Icon name="sparkles" size={12} />
            <span className="font-semibold uppercase tracking-wide">Proposed deletion</span>
            <Badge tone="hay">pending</Badge>
          </div>
          <p className="font-serif text-[14px] leading-[1.7] text-ink-500 line-through decoration-rose-300">
            {change.oldContent || "(empty block)"}
          </p>
        </div>
      ))}

      {pendingInserts.map((change) => (
        <InsertCard
          key={change.id}
          change={change}
          sourceTitle={state.sources[change.sourceId ?? ""]?.title ?? "session source"}
        />
      ))}

      <SummarySlateEditor
        docId={doc.id}
        blocks={editableBlocks}
        modifies={pendingModifies}
        externalKey={`${doc.id}:${pending.map((c) => c.id + c.status).join(",")}:${editableBlocks.length}`}
      />

      {pending.length === 0 && (
        <p className="mt-8 flex items-center gap-2 border-t border-line-soft pt-4 text-[11.5px] text-mute">
          <Icon name="shieldCheck" size={13} className="text-moss-600" />
          AI edits appear inline as underlined proposals — click one to accept, reject, or revise. Cmd/Ctrl+B bold, Cmd/Ctrl+I italic.
        </p>
      )}
    </div>
  );
}
