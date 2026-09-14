/**
 * Summary document view — EDITABLE research summary.
 *
 * Status: BASIC WORKING IMPLEMENTATION.
 *  - Paragraphs and headings are real contentEditable blocks (plain text).
 *  - Pending AI changes render visually (inserted = amber card,
 *    modified = amber underline fragment) and become plain text on accept.
 *  - No rich formatting yet; a real editor (e.g. ProseMirror/TipTap or a
 *    Go-side editor) replaces EditableText + block rendering only.
 *
 * Deliberately NOT shared with SourceDocumentView: source documents are
 * read-only with range selection; summaries are block-editable. The
 * shared part is the DocumentModel itself.
 */

import { useRef, type ReactNode } from "react";
import { useApp, pendingChangesFor } from "../../state/appState";
import type { Document, DocumentChange } from "../../types/domain";
import { Icon } from "../../components/icons";
import { Badge } from "../../components/ui";
import { cn } from "../../utils/cn";

function escapeHtml(text: string): string {
  return text.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;");
}

/* Uncontrolled editable block. React never rewrites its text node while
   the user types; external updates (accept/reject) remount via key. */
function EditableText({
  initialText,
  onText,
  className,
}: {
  initialText: string;
  onText: (text: string) => void;
  className?: string;
}) {
  const init = useRef<string | null>(null);
  if (init.current === null) init.current = escapeHtml(initialText);
  return (
    <div
      contentEditable
      suppressContentEditableWarning
      spellCheck={false}
      onInput={(e) => onText((e.currentTarget as HTMLElement).innerText)}
      className={cn("cursor-text rounded-[4px] px-0.5 transition-shadow", className)}
      dangerouslySetInnerHTML={{ __html: init.current }}
    />
  );
}

/* "Modified" pending state: original sentence + new fragment underlined. */
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

export default function SummaryDocumentView({ doc }: { doc: Document }) {
  const { state, dispatch } = useApp();
  const pending = pendingChangesFor(state, doc.id);
  const changeByBlock = new Map(pending.map((c) => [c.blockId, c]));

  const setText = (blockId: string, text: string) =>
    dispatch({ type: "doc.block.text", documentId: doc.id, blockId, text });

  return (
    <div className="mx-auto max-w-[700px] px-8 py-6">
      <div className="mb-4 flex items-center justify-between border-b border-line pb-2.5 text-[12px] text-mute">
        <span>Editable summary · contributes {doc.sourceIds?.length ?? 0} sources</span>
        <span className="flex items-center gap-1.5">
          <span className={cn("h-1.5 w-1.5 rounded-full", pending.length ? "bg-hay-300" : "bg-moss-200")} />
          {pending.length ? `${pending.length} pending change${pending.length === 1 ? "" : "s"}` : "all changes reviewed"}
        </span>
      </div>

      {doc.blocks.map((block, i) => {
        const change = changeByBlock.get(block.id);

        /* title */
        if (i === 0 && block.type === "heading" && block.level === 1) {
          return (
            <h1
              key={block.id}
              className="mb-7 font-serif text-[26px] font-bold leading-tight text-ink-900"
            >
              <EditableText
                initialText={block.segments.map((s) => s.text).join("")}
                onText={(t) => dispatch({ type: "doc.title", documentId: doc.id, text: t })}
                className="block"
              />
            </h1>
          );
        }

        /* headings */
        if (block.type === "heading") {
          return (
            <h2 key={block.id} className="mt-8 font-serif text-[19px] font-bold text-ink-900">
              <EditableText
                initialText={block.segments.map((s) => s.text).join("")}
                onText={(t) => setText(block.id, t)}
                className="block"
              />
            </h2>
          );
        }

        /* paragraphs */
        const text = block.segments.map((s) => s.text).join("");
        if (change && change.type === "insert" && change.status === "pending") {
          return (
            <InsertCard
              key={`${block.id}:${change.status}`}
              change={change}
              sourceTitle={state.sources[change.sourceId]?.title ?? "session source"}
            />
          );
        }
        if (change && change.type === "modify" && change.status === "pending") {
          return (
            <div key={`${block.id}:${change.status}`}>
              <ModifiedText full={change.newContent} fragment={change.highlightFragment} />
            </div>
          );
        }
        return (
          <p key={`${block.id}:${change?.status ?? "none"}`} className="my-4 doc-body">
            <EditableText initialText={text} onText={(t) => setText(block.id, t)} className="block" />
          </p>
        );
      })}

      {/* accepted changes remain as plain text; show a quiet provenance line */}
      {pending.length === 0 && (
        <p className="mt-8 flex items-center gap-2 border-t border-line-soft pt-4 text-[11.5px] text-mute">
          <Icon name="shieldCheck" size={13} className="text-moss-600" />
          AI edits appear here as highlighted proposals until you accept or reject them.
        </p>
      )}
    </div>
  );
}
