/**
 * AI Reading panel (right, resizable, collapsible).
 *
 * Faces:
 *  - Chat       — multi-turn workspace-scoped thread + composer
 *  - Reading    — source companion (selected passage / explanation)
 *
 * Review happens in-file: published summary edits decorate the document
 * itself (SummaryDocumentView diffs). There is deliberately no separate
 * proposal queue — one review path, no fallback.
 *
 * Tab switch is driven by reader.ui.aiPanelTab ("chat" | "reading").
 */

import { useEffect, useRef, useState } from "react";
import { useApp, runningActivity, activeWorkspace } from "../../state/appState";
import { useAIRunners, type SelectionRef } from "../../state/aiController";
import type { AiPanelTab, ChatMessage, Document } from "../../types/domain";
import { Icon } from "../../components/icons";
import { Button, EmptyState, IconBtn, Spinner } from "../../components/ui";
import { cn } from "../../utils/cn";
import { timeLabel } from "../../utils/helpers";

/* ------------------------------------------------------------------ */
/* Live activity strip (shared)                                        */
/* ------------------------------------------------------------------ */

function ActivityStrip() {
  const { state } = useApp();
  const running = runningActivity(state);
  if (!running) return null;
  const doc = running.documentId ? state.documents[running.documentId] : undefined;
  const src = running.sourceId ? state.sources[running.sourceId] : undefined;
  return (
    <div className="fade-in mx-4 mb-3 flex items-start gap-2.5 rounded-lg border border-iris-200 bg-iris-50 px-3 py-2.5">
      <Spinner size={13} className="mt-0.5 shrink-0 text-iris-600" />
      <div className="min-w-0">
        <p className="text-[12px] font-medium text-iris-700">{running.message}</p>
        <p className="mt-0.5 truncate text-[10.5px] text-mute">
          {running.operation.replace("AI_", "")}
          {doc ? ` · ${doc.metadata.title}` : ""}
          {src ? ` · ${src.title}` : ""}
        </p>
      </div>
    </div>
  );
}

function HistoryList({ documentId }: { documentId?: string }) {
  const { state } = useApp();
  const items = state.aiActivities.filter((a) => !documentId || a.documentId === documentId).slice(0, 4);
  if (items.length === 0) return null;
  return (
    <div className="mx-4 mt-auto border-t border-line-soft pt-3">
      <h4 className="mb-2 text-[10.5px] font-semibold uppercase tracking-wide text-mute">Recent AI activity</h4>
      <ul className="space-y-1.5">
        {items.map((a) => (
          <li key={a.id} className="flex items-center gap-2 text-[11px] text-ink-500">
            <Icon
              name={a.status === "error" ? "alert" : "sparkles"}
              size={11}
              className={cn("shrink-0", a.status === "error" ? "text-rose-500" : "text-iris-500")}
            />
            <span className="min-w-0 flex-1 truncate">{a.message}</span>
            <span className="shrink-0 text-mute">{timeLabel(a.startedAt)}</span>
          </li>
        ))}
      </ul>
    </div>
  );
}

/* ------------------------------------------------------------------ */
/* Chat face (Phase 4)                                                 */
/* ------------------------------------------------------------------ */

/** Expandable per-turn tool loop log ("thinking"): what ran, ok/err, ms. */
function ThinkingTrace({ trace }: { trace: NonNullable<ChatMessage["toolTrace"]> }) {
  const [open, setOpen] = useState(false);
  const total = trace.reduce((n, t) => n + t.ms, 0);
  const errs = trace.filter((t) => !t.ok).length;
  return (
    <div className="mt-1.5 border-t border-line-soft pt-1.5">
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        className="flex items-center gap-1.5 text-[10.5px] text-mute hover:text-ink-700"
      >
        <Icon name={open ? "chevronDown" : "chevronRight"} size={11} />
        Thinking · {trace.length} call{trace.length === 1 ? "" : "s"}
        {errs > 0 ? ` · ${errs} failed` : ""} · {total}ms
      </button>
      {open && (
        <ul className="mt-1 space-y-0.5 font-mono text-[10.5px]">
          {trace.map((t, i) => (
            <li key={i} className={cn("flex items-center gap-1.5", t.ok ? "text-ink-500" : "text-rose-600")}>
              <Icon name={t.ok ? "check" : "alert"} size={10} className="shrink-0" />
              <span className="truncate">{t.tool}</span>
              <span className="shrink-0 text-mute">{t.ms}ms</span>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}

function ChatFace({ doc }: { doc: Document }) {
  const { state, dispatch } = useApp();
  const { runChat, ensureThread, newThread } = useAIRunners();
  const [draft, setDraft] = useState("");
  const bottomRef = useRef<HTMLDivElement>(null);
  const running = runningActivity(state);
  const runningChat = running?.operation === "AI_CHAT";

  const wsId = activeWorkspace(state)?.id ?? doc.workspaceId ?? "ws-demo";
  const thread =
    (state.chat.activeId && state.chat.byId[state.chat.activeId]) ||
    Object.values(state.chat.byId).find((t) => t.workspaceId === wsId) ||
    null;
  // All chats of this workspace, newest first (plan 09 §3 thread switcher).
  const wsThreads = Object.values(state.chat.byId)
    .filter((t) => t.workspaceId === wsId)
    .sort((a, b) => b.updatedAt - a.updatedAt);

  // @doc autocomplete: token after the last "@" before the cursor filters
  // workspace documents by title; picking one splices the full title in.
  // Send-time resolution (resolveMentions) is the ground truth — this list
  // is just a typing aid so tokens match real documents.
  const [cursor, setCursor] = useState(0);
  const atToken = (() => {
    const before = draft.slice(0, cursor);
    const m = /@([\p{L}\p{N}._-]*)$/u.exec(before);
    return m ? m[1] : null;
  })();
  const candidates = (() => {
    if (atToken === null) return [];
    const docs = Object.values(state.documents).filter(
      (d) => !d.workspaceId || d.workspaceId === wsId,
    );
    const needle = atToken.toLowerCase();
    return docs
      .filter((d) => !needle || d.metadata.title.toLowerCase().includes(needle))
      .slice(0, 6);
  })();
  const completeMention = (title: string) => {
    const before = draft.slice(0, cursor);
    const after = draft.slice(cursor);
    const replaced = before.replace(/@[\p{L}\p{N}._-]*$/u, `@${title} `);
    const next = replaced + after;
    setDraft(next);
    setCursor(replaced.length);
  };

  useEffect(() => {
    ensureThread();
  }, [ensureThread]);

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [thread?.messages.length, runningChat]);

  const send = () => {
    const t = draft.trim();
    if (!t || runningChat) return;
    setDraft("");
    runChat(t, { documentId: doc.id, includeContext: true });
  };

  const clear = () => {
    if (!thread) return;
    dispatch({ type: "chat.clear", threadId: thread.id });
  };

  const startNewChat = () => {
    if (runningChat) return;
    newThread();
  };

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <ActivityStrip />
      {wsThreads.length > 1 && (
        <div className="flex gap-1.5 overflow-x-auto px-4 pb-2">
          {wsThreads.map((t) => (
            <button
              key={t.id}
              type="button"
              onClick={() => dispatch({ type: "chat.setActive", id: t.id })}
              title={t.title ?? "Research chat"}
              className={cn(
                "shrink-0 rounded-full border px-2.5 py-1 text-[10.5px]",
                t.id === thread?.id
                  ? "border-iris-400 bg-iris-50 font-medium text-iris-700"
                  : "border-line-soft text-mute hover:text-ink-700",
              )}
            >
              {(t.title ?? "Chat").slice(0, 18)}
              {t.messages.length > 0 ? ` · ${t.messages.length}` : ""}
            </button>
          ))}
        </div>
      )}
      <div className="min-h-0 flex-1 space-y-3 overflow-y-auto px-4 py-3">
        {!thread || thread.messages.length === 0 ? (
          <EmptyState
            icon="sparkles"
            title="Workspace chat"
            hint="One conversation per folder — it follows you across files. Context from the open document is attached automatically. Type @ to target a document; select text to anchor a region."
          />
        ) : (
          thread.messages.map((m) => (
            <div
              key={m.id}
              className={cn(
                "rounded-lg px-3 py-2.5 text-[12.5px] leading-relaxed",
                m.role === "user"
                  ? "ml-6 bg-iris-50 text-ink-800"
                  : "mr-4 border border-line-soft bg-cream-50 text-ink-700",
              )}
            >
              <p className="mb-1 text-[10px] font-semibold uppercase tracking-wide text-mute">
                {m.role === "user" ? "You" : "Assistant"}
              </p>
              <p className="whitespace-pre-wrap">{m.content}</p>
              {m.role === "assistant" && m.toolTrace && m.toolTrace.length > 0 && (
                <ThinkingTrace trace={m.toolTrace} />
              )}
            </div>
          ))
        )}
        {runningChat && (
          <div className="mr-4 flex items-center gap-2 rounded-lg border border-iris-200 bg-iris-50 px-3 py-2 text-[12px] text-iris-700">
            <Spinner size={12} /> Thinking…
          </div>
        )}
        <div ref={bottomRef} />
      </div>

      <div className="border-t border-line-soft px-3 py-2.5">
        {candidates.length > 0 && (
          <div className="mb-1.5 overflow-hidden rounded-lg border border-line-soft bg-white shadow-sm">
            {candidates.map((d) => (
              <button
                key={d.id}
                type="button"
                onClick={() => completeMention(d.metadata.title)}
                className="flex w-full items-center gap-2 px-2.5 py-1.5 text-left text-[12px] text-ink-700 hover:bg-iris-50"
              >
                <span className="font-semibold text-iris-600">@</span>
                <span className="min-w-0 flex-1 truncate">{d.metadata.title}</span>
                <span className="shrink-0 text-[10.5px] text-mute">{d.kind}</span>
              </button>
            ))}
          </div>
        )}
        <div className="flex items-end gap-2">
          <textarea
            value={draft}
            onChange={(e) => {
              setDraft(e.target.value);
              setCursor(e.target.selectionStart ?? e.target.value.length);
            }}
            onSelect={(e) => setCursor((e.target as HTMLTextAreaElement).selectionStart ?? 0)}
            onKeyDown={(e) => {
              if (e.key === "Enter" && !e.shiftKey) {
                e.preventDefault();
                send();
              }
            }}
            rows={2}
            placeholder="Ask about the document… (@title to target a doc)"
            className="min-h-[44px] flex-1 resize-none rounded-lg border border-line bg-cream-50 px-3 py-2 text-[12.5px] text-ink-800 outline-none focus:border-iris-300"
            disabled={!!runningChat}
          />
          <Button size="sm" variant="soft" icon="sparkles" disabled={!draft.trim() || !!runningChat} onClick={send}>
            Send
          </Button>
        </div>
        <div className="mt-1.5 flex items-center justify-between">
          <p className="text-[10.5px] text-mute">Enter to send · Shift+Enter newline</p>
          <div className="flex items-center gap-3">
            <button
              type="button"
              className="text-[10.5px] text-mute hover:text-ink-700"
              onClick={startNewChat}
            >
              New chat
            </button>
            {thread && thread.messages.length > 0 && (
              <button
                type="button"
                className="text-[10.5px] text-mute hover:text-ink-700"
                onClick={clear}
              >
                Clear thread
              </button>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}

/* ------------------------------------------------------------------ */
/* Companion face (source documents)                                   */
/* ------------------------------------------------------------------ */

function CompanionFace({ doc }: { doc: Document }) {
  const { state } = useApp();
  const { runExplain, runIncludeInSummary, runRelatedSources } = useAIRunners();
  const companion =
    state.reader.ui.companion && state.reader.ui.companion.documentId === doc.id
      ? state.reader.ui.companion
      : null;
  const running = runningActivity(state);
  const runningHere = running && running.documentId === doc.id;

  const asRef = (text: string): SelectionRef => ({ documentId: doc.id, blockId: "", text });

  if (!companion) {
    return (
      <div className="flex min-h-0 flex-1 flex-col">
        <ActivityStrip />
        <EmptyState
          icon="sparkles"
          title="Ask about anything in the document"
          hint="Select a passage with your cursor — a toolbar appears with Ask AI, Explain, Go deeper, Include in summary and Save note."
        />
        <HistoryList documentId={doc.id} />
      </div>
    );
  }

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <div className="min-h-0 flex-1 space-y-4 overflow-y-auto px-4 py-4">
        <ActivityStrip />

        <section>
          <h4 className="mb-1.5 flex items-center gap-1.5 text-[12px] font-semibold text-ink-900">
            <Icon name="bookmark" size={13} className="text-moss-600" /> Selected passage
          </h4>
          <p className="rounded-lg bg-moss-50 px-3 py-2.5 font-serif text-[13px] leading-relaxed text-moss-700">
            {companion.passage}
          </p>
        </section>

        <section>
          <h4 className="mb-1.5 flex items-center gap-1.5 text-[12px] font-semibold text-ink-900">
            <Icon name="bulb" size={13} className="text-iris-600" />
            {runningHere ? "Working…" : "Concise explanation"}
          </h4>
          <p className="rounded-lg bg-iris-50 px-3 py-2.5 text-[12.5px] leading-relaxed text-ink-700">
            {runningHere ? running?.message : companion.explanation?.trim() ? companion.explanation : "AI unavailable now."}
          </p>
        </section>

        <section>
          <h4 className="mb-1.5 flex items-center gap-1.5 text-[12px] font-semibold text-ink-900">
            <Icon name="bookOpen" size={13} className="text-iris-600" /> Related sources
          </h4>
          {companion.relatedSources && companion.relatedSources.length > 0 ? (
            <div className="space-y-1.5">
              {companion.relatedSources.map((r) => (
                <div key={r.title} className="flex items-start gap-2.5 rounded-lg border border-line-soft bg-cream-50 px-3 py-2">
                  <Icon name="fileText" size={13} className="mt-0.5 shrink-0 text-ink-400" />
                  <div className="min-w-0">
                    <p className="truncate text-[12px] font-medium text-ink-900">{r.title}</p>
                    <p className="truncate text-[10.5px] text-mute">{r.meta}</p>
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <p className="rounded-lg border border-line-soft bg-cream-50 px-3 py-2.5 text-[12px] text-mute">
              No related sources — choose “Find related sources”.
            </p>
          )}
        </section>

        <div className="grid grid-cols-2 gap-2">
          <Button size="sm" variant="soft" icon="search" onClick={runRelatedSources} disabled={!!running}>
            Find related sources
          </Button>
          <Button
            size="sm"
            variant="green"
            icon="list"
            disabled={!!running}
            onClick={() => runIncludeInSummary(asRef(companion.passage))}
          >
            Include in summary
          </Button>
          <Button
            size="sm"
            variant="outline"
            icon="note"
            className="col-span-2"
            disabled={!!running}
            onClick={() => runExplain("AI_EXPLAIN", asRef(companion.passage))}
          >
            Revise explanation
          </Button>
        </div>
      </div>

      <div className="border-t border-line-soft px-4 py-3">
        <p className="flex items-center gap-1.5 text-[11px] text-mute">
          <Icon name="link" size={11} /> Source: {companion.sourceLabel}
          <Icon name="external" size={11} className="ml-auto" />
        </p>
      </div>
      <HistoryList documentId={doc.id} />
    </div>
  );
}

/* ------------------------------------------------------------------ */
/* Panel shell + tabs                                                  */
/* ------------------------------------------------------------------ */

export default function AIReadingPanel({ doc }: { doc: Document }) {
  const { state, dispatch } = useApp();
  const tab: AiPanelTab = state.reader.ui.aiPanelTab ?? "chat";

  const setTab = (next: AiPanelTab) =>
    dispatch({ type: "reader.ui", patch: { aiPanelTab: next } });

  return (
    <div className="flex h-full min-h-0 flex-col">
      <div className="flex items-center justify-between border-b border-line-soft px-3 py-2">
        <div className="flex items-center gap-1 rounded-lg bg-cream-100 p-0.5">
          <button
            type="button"
            onClick={() => setTab("chat")}
            className={cn(
              "rounded-md px-2.5 py-1 text-[12px] font-medium transition-colors",
              tab === "chat" ? "bg-white text-ink-900 shadow-sm" : "text-mute hover:text-ink-700",
            )}
          >
            Chat
          </button>
          <button
            type="button"
            onClick={() => setTab("reading")}
            className={cn(
              "rounded-md px-2.5 py-1 text-[12px] font-medium transition-colors",
              tab === "reading" ? "bg-white text-ink-900 shadow-sm" : "text-mute hover:text-ink-700",
            )}
          >
            Reading
          </button>
        </div>
        <IconBtn
          name="x"
          title="Hide panel"
          onClick={() => dispatch({ type: "reader.ui", patch: { aiPanelOpen: false } })}
        />
      </div>

      {tab === "chat" ? <ChatFace doc={doc} /> : <CompanionFace doc={doc} />}
    </div>
  );
}
