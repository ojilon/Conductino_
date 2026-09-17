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

import { useCallback, useMemo, type ReactNode } from "react";
import {
  createEditor,
  type Descendant,
  type BaseEditor,
  Editor,
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

function ModifiedText({ full, fragment }: { full: string; fragment?: string }) {
  let body: ReactNode = full;
  if (fragment && full.includes(fragment)) {
    const idx = full.indexOf(fragment);
    body = (
      <>
        {full.slice(0, idx)}
        <span className="rounded-[3px] bg-hay-100 px-0.5 underline decoration-hay-300 decoration-2 underline-offset-4">
          {fragment}
        </span>
        {full.slice(idx + fragment.length)}
      </>
    );
  }
  return <p className="my-4 doc-body">{body}</p>;
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

function SummarySlateEditor({
  docId,
  blocks,
  externalKey,
}: {
  docId: ID;
  blocks: DocumentBlock[];
  externalKey: string;
}) {
  const { dispatch } = useApp();
  const editor = useMemo(() => withHistory(withReact(createEditor())), [externalKey]);
  const initial = useMemo(() => toSlate(blocks), [externalKey]); // eslint-disable-line react-hooks/exhaustive-deps

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

  const renderElement = useCallback((props: RenderElementProps) => <Element {...props} />, []);
  const renderLeaf = useCallback((props: RenderLeafProps) => <Leaf {...props} />, []);

  return (
    <Slate key={externalKey} editor={editor} initialValue={initial} onChange={onChange}>
      <Editable
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
  );
}

export default function SummaryDocumentView({ doc }: { doc: Document }) {
  const { state, dispatch } = useApp();
  const pending = pendingChangesFor(state, doc.id);
  const changeByBlock = new Map(pending.map((c) => [c.blockId, c]));

  const pendingInserts = pending.filter((c) => c.type === "insert");
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
            : "all changes reviewed"}
        </span>
      </div>

      {Array.from(pendingModifies.values()).map((change) => (
        <div key={`mod-${change.id}`} className="mb-2">
          <div className="mb-1 flex items-center gap-2 text-[11px] text-hay-600">
            <Icon name="sparkles" size={12} />
            <span className="font-semibold uppercase tracking-wide">Proposed revision</span>
            <Badge tone="hay">pending</Badge>
          </div>
          <ModifiedText full={change.newContent} fragment={change.highlightFragment} />
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
        externalKey={`${doc.id}:${pending.map((c) => c.id + c.status).join(",")}:${editableBlocks.length}`}
      />

      {pending.length === 0 && (
        <p className="mt-8 flex items-center gap-2 border-t border-line-soft pt-4 text-[11.5px] text-mute">
          <Icon name="shieldCheck" size={13} className="text-moss-600" />
          AI edits appear here as highlighted proposals until you accept or reject them. Cmd/Ctrl+B bold, Cmd/Ctrl+I italic.
        </p>
      )}
    </div>
  );
}
