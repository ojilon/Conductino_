/**
 * AI controller — the ONLY place AI requests are initiated.
 *
 * Flow (identical for mock and real providers):
 *   user action → runX() → activity recorded (aiActivities)
 *   → provider streams phases → state updates → UI re-renders
 *   → provider delivers result → companion / DocumentChange / toast.
 *
 * Because everything goes through the AIProvider interface, swapping
 * the mock for a real API changes nothing here. See docs/ai-integration.md.
 */

import { useCallback, useRef } from "react";
import { useApp, activeReaderDocument, primarySummaryDocument, activeWorkspace } from "./appState";
import { buildContextPackAsync } from "./contextPack";
import { getAIProvider } from "../services/ai";
import { saveChatThread, appendChatMessage, backend } from "../services/backend";
import { uid, parseBlocksWire } from "../utils/helpers";
import type { AIActivity, AIOperation, AIResult, ChatMessage, ChatThread, ChatTurn, Document, SummaryDiff } from "../types/domain";
import { resolveMentions } from "./mentions";

export interface SelectionRef {
  documentId: string;
  blockId: string;
  text: string;
  range?: { start: number; end: number };
}

/** Plain-text snapshot of a document for tool/context (Phase 5 summary tool). */function documentPlainText(doc: Document | undefined, maxChars = 8000): string {
  if (!doc?.blocks?.length) return "";
  const parts: string[] = [];
  for (const b of doc.blocks) {
    if (b.type === "list" && b.listItems?.length) {
      parts.push(b.listItems.map((item) => "• " + item.map((s) => s.text).join("")).join("\n"));
    } else {
      parts.push(b.segments.map((s) => s.text).join(""));
    }
  }
  let t = parts.join("\n\n").trim();
  if (t.length > maxChars) t = t.slice(0, maxChars - 20) + "\n…[truncated]";
  return t;
}

/** Fuzzy-match a quoted span to a summary block (proposal fallback). */
function findBlockForSpan(doc: Document, span: string): string | undefined {
  const needle = span.trim().slice(0, 160);
  if (!needle) return undefined;
  const probe = needle.length > 60 ? needle.slice(0, 60) : needle;
  return doc.blocks.find((b) =>
    b.segments.map((s) => s.text).join("").includes(probe),
  )?.id;
}

export function useAIRunners() {
  const { state, dispatch } = useApp();
  const cancels = useRef<Record<string, () => void>>({});

  const start = useCallback(
    (operation: AIOperation, extra: Partial<AIActivity> = {}) => {
      const activity: AIActivity = {
        id: uid("act"),
        operation,
        status: "running",
        message: "Starting",
        changeIds: [],
        startedAt: Date.now(),
        ...extra,
      };
      dispatch({ type: "activity.start", activity });
      return activity.id;
    },
    [dispatch],
  );

  const finish = useCallback(
    (id: string, patch: Partial<AIActivity>) => {
      dispatch({ type: "activity.update", id, patch: { ...patch, finishedAt: patch.status ? Date.now() : undefined } });
    },
    [dispatch],
  );

  /* ---------- AI Browse (web research) ---------- */

  const runBrowse = useCallback(
    (query: string) => {
      const id = start("AI_SEARCH", { message: "Searching…" });
      dispatch({ type: "browse.start", query });
      cancels.current[id] = getAIProvider().run(
        { operation: "AI_SEARCH", query },
        {
          onPhase: (i, label) => {
            dispatch({ type: "browse.phase", phase: i });
            dispatch({ type: "activity.update", id, patch: { message: `${label}…` } });
          },
          onBrowseSources: (sourceIds) => dispatch({ type: "browse.done", sourceIds }),
          onDone: () =>
            finish(id, { status: "completed", message: `Ranked sources for “${query || "query"}”` }),
          onError: (message) => {
            dispatch({ type: "browse.error", message });
            finish(id, { status: "error", error: message });
          },
        },
      );
    },
    [start, finish, dispatch],
  );

  /* ---------- Explanation family (selection → companion panel) ---------- */

  const runExplain = useCallback(
    async (operation: AIOperation, selection: SelectionRef, customPrompt?: string) => {
      const doc = state.documents[selection.documentId];
      const meta = doc?.metadata;
      const sourceLabel = meta?.author
        ? `${meta.author}${meta.year ? ` (${meta.year})` : ""}${meta.venue ? `, ${meta.venue}` : ""}`
        : "Open document";
      const wsId = activeWorkspace(state)?.id ?? doc?.workspaceId;
      // Server-side assembly when bridged (plan 11 §4); local fallback in browser.
      const contextPack = await buildContextPackAsync(doc, {
        blockId: selection.blockId,
        text: selection.text,
        range: selection.range,
      });
      const id = start(operation, {
        documentId: selection.documentId,
        workspaceId: wsId,
        selection: selection.text,
        message: "Reading selection…",
      });
      cancels.current[id] = getAIProvider().run(
        {
          operation,
          documentId: selection.documentId,
          selection: { blockId: selection.blockId, text: selection.text },
          workspaceId: wsId,
          customPrompt: customPrompt?.trim() || undefined,
          includeDocumentContext: true,
          contextPack: contextPack || undefined,
        },
        {
          onPhase: (_i, label) =>
            dispatch({ type: "activity.update", id, patch: { message: `${label}…` } }),
          onBrowseSources: () => undefined,
          onDone: (result) => {
            const explanation = result.explanation?.trim() ? result.explanation : "AI unavailable now.";
            dispatch({
              type: "reader.ui",
              patch: {
                companion: {
                  documentId: selection.documentId,
                  passage: selection.text,
                  explanation,
                  sourceLabel,
                  relatedSources: result.relatedSources ?? [],
                },
                aiPanelTab: "reading",
                aiPanelOpen: true,
              },
            });
            finish(id, { status: "completed", message: `${operation.replace("AI_", "").toLowerCase()} completed` });
          },
          onError: (message) => {
            dispatch({
              type: "reader.ui",
              patch: {
                companion: {
                  documentId: selection.documentId,
                  passage: selection.text,
                  explanation: "AI unavailable now.",
                  sourceLabel,
                  relatedSources: [],
                },
                aiPanelTab: "reading",
                aiPanelOpen: true,
              },
            });
            finish(id, { status: "error", error: message });
            dispatch({ type: "toast", message: "AI unavailable now." });
          },
        },
      );
    },
    [state, start, finish, dispatch],
  );

  /* ---------- Include selection in the summary (source → summary) ---------- */

  const runIncludeInSummary = useCallback(
    async (selection: SelectionRef) => {
      const summary = primarySummaryDocument(state);
      if (!summary) {
        dispatch({
          type: "toast",
          message: "No summary mapped to this workspace — create a research summary first.",
        });
        return;
      }
      const sourceDoc = state.documents[selection.documentId];
      const sourceId = sourceDoc?.sourceId ?? selection.documentId;
      const wsId = activeWorkspace(state)?.id ?? sourceDoc?.workspaceId ?? summary.workspaceId;
      const id = start("AI_MERGE", {
        documentId: summary.id,
        sourceId,
        workspaceId: wsId,
        selection: selection.text,
        message: "Extracting claim…",
      });
      const contextPack = await buildContextPackAsync(sourceDoc, {
        blockId: selection.blockId,
        text: selection.text,
        range: selection.range,
      });
      cancels.current[id] = getAIProvider().run(
        {
          operation: "AI_MERGE",
          documentId: selection.documentId,
          sourceId,
          selection: { blockId: selection.blockId, text: selection.text },
          workspaceId: wsId,
          includeDocumentContext: true,
          contextPack: contextPack || undefined,
        },
        {
          onPhase: (_i, label) =>
            dispatch({ type: "activity.update", id, patch: { message: `${label}…` } }),
          onBrowseSources: () => undefined,
          onDone: (result) => {
            const insertion = result.insertion?.text?.trim() ? result.insertion : undefined;
            const changeId = uid("chg");
            const blockId = uid("s-ai");
            const text = insertion
              ? `${insertion.text} ${insertion.citation}`
              : `— “${selection.text.slice(0, 120)}” (AI unavailable — pasted without drafting)`;
            dispatch({
              type: "change.propose",
              change: {
                id: changeId,
                documentId: summary.id,
                workspaceId: wsId,
                type: "insert",
                blockId,
                oldContent: "",
                newContent: text,
                sourceId,
                activityId: id,
                // User-explicit (toolbar click): the insert lands directly in
                // the editor — no card review. Single-path: accepted at birth.
                status: "accepted",
                createdAt: Date.now(),
              },
              block: { id: blockId, type: "paragraph", changeId: changeId, segments: [{ text }] },
            });
            dispatch({ type: "summary.addSource", documentId: summary.id, sourceId });
            finish(id, {
              status: "completed",
              message: `Proposed insertion in ${summary.metadata.title}`,
              changeIds: [changeId],
            });
            dispatch({ type: "selection.set", selection: null });
            dispatch({ type: "toast", message: "Added to the summary — edit inline or ask the AI to revise" });
          },
          onError: (message) => {
            finish(id, { status: "error", error: message });
            dispatch({ type: "toast", message: "AI unavailable now — nothing was added to the summary." });
          },
        },
      );
    },
    [state, start, finish, dispatch],
  );

  /* ---------- Companion panel: find related sources ---------- */

  const runRelatedSources = useCallback(() => {
    const doc = activeReaderDocument(state);
    if (!doc) {
      dispatch({ type: "toast", message: "Open a document first to show AI response." });
      return;
    }
    const id = start("AI_SEARCH", {
      documentId: doc.id,
      message: "Finding related sources…",
    });
    cancels.current[id] = getAIProvider().run(
      { operation: "AI_SEARCH", query: "related sources", documentId: doc.id },
      {
        onPhase: (_i, label) => dispatch({ type: "activity.update", id, patch: { message: `${label}…` } }),
        onBrowseSources: () => undefined,
        onDone: () => {
          finish(id, { status: "completed", message: "Refreshed related sources" });
          dispatch({ type: "toast", message: "Related-source lookup is not available yet." });
        },
        onError: (message) => {
          finish(id, { status: "error", error: message });
          dispatch({ type: "toast", message: "AI unavailable now." });
        },
      },
    );
  }, [state, start, finish, dispatch]);

  /* ---------- Phase 4/5: multi-turn chat (+ tools) ---------- */

  /** Ensure the workspace chat exists and is active; returns its id. */
  const ensureThread = useCallback(
    (documentId?: string): string => {
      const ws = activeWorkspace(state);
      const wsId = ws?.id ?? "ws-demo";
      // ONE chat per workspace (plan 09): match workspace only. The open
      // document rides on runChat opts as the region anchor — never as
      // thread identity — so the conversation survives file switches.
      // documentId is kept on creation as provenance ("started from").
      const existing = Object.values(state.chat.byId).find(
        (t) => t.workspaceId === wsId,
      );
      if (existing) {
        if (state.chat.activeId !== existing.id) {
          dispatch({ type: "chat.setActive", id: existing.id });
        }
        return existing.id;
      }
      const now = Date.now();
      const thread: ChatThread = {
        id: uid("thr"),
        workspaceId: wsId,
        documentId,
        title: "Research chat",
        messages: [],
        createdAt: now,
        updatedAt: now,
      };
      dispatch({ type: "chat.ensure", thread });
      // Persist best-effort (desktop SQLite; silent no-op in browser).
      void saveChatThread(thread);
      return thread.id;
    },
    [state, dispatch],
  );

  /** Start a fresh workspace chat (explicit "New chat" button). History stays in byId. */
  const newThread = useCallback((): string => {
    const ws = activeWorkspace(state);
    const wsId = ws?.id ?? "ws-demo";
    const now = Date.now();
    const thread: ChatThread = {
      id: uid("thr"),
      workspaceId: wsId,
      title: "Research chat",
      messages: [],
      createdAt: now,
      updatedAt: now,
    };
    dispatch({ type: "chat.ensure", thread });
    dispatch({ type: "chat.setActive", id: thread.id });
    void saveChatThread(thread);
    return thread.id;
  }, [state, dispatch]);

  // Published summaries: the .docx changed on disk mid-turn — reload blocks
  // in place (editor-refresh semantics) and decorate write-through diffs.
  const reloadPublishedSummaries = useCallback(
    async (published: NonNullable<AIResult["publishedSummaries"]>) => {
      for (const pub of published) {
        const doc =
          state.documents[pub.summaryId] ??
          Object.values(state.documents).find((d) => d.metadata.path === pub.path);
        if (!doc?.metadata.path) continue;
        try {
          const opened = await backend.filesystem.openFile(doc.metadata.path);
          if (!opened || opened.reason || !opened.blocksJSON) continue;
          const blocks = parseBlocksWire(opened.blocksJSON);
          if (!blocks) continue;
          dispatch({ type: "doc.blocks.replace", documentId: doc.id, blocks });
          const diffs: SummaryDiff[] = pub.diffs.map((d) => ({
            id: uid("df"),
            op: d.op,
            target: d.target,
            text: d.text ?? "",
            oldText: d.oldText,
            note: d.note,
          }));
          dispatch({ type: "doc.diffs.set", documentId: doc.id, diffs });
          dispatch({
            type: "toast",
            message: `Summary updated — ${diffs.length} diff${diffs.length === 1 ? "" : "s"} to review in-file`,
          });
        } catch {
          dispatch({ type: "toast", message: "Summary published — reopen it to see the changes." });
        }
      }
    },
    [state.documents, dispatch],
  );

  const runChat = useCallback(
    async (text: string, opts?: { documentId?: string; includeContext?: boolean; selection?: SelectionRef; changeId?: string }) => {
      const trimmed = text.trim();
      if (!trimmed) return;

      const doc = opts?.documentId
        ? state.documents[opts.documentId]
        : activeReaderDocument(state);
      const documentId = doc?.id;
      const wsId = activeWorkspace(state)?.id ?? doc?.workspaceId;
      const threadId = ensureThread(documentId);

      // @doc resolution: tokens in the composer become explicit document ids
      // so the harness never guesses. A mentioned summary becomes the
      // proposal target, overriding the workspace primary.
      const wsDocs = Object.values(state.documents).filter(
        (d) => !wsId || !d.workspaceId || d.workspaceId === wsId,
      );
      const mentionIds = resolveMentions(trimmed, wsDocs, (d) => d.metadata.title);
      const mentionedSummary = mentionIds
        .map((id) => state.documents[id])
        .find((d) => d?.kind === "summary");
      const summary = mentionedSummary ?? primarySummaryDocument(state);

      // Region anchor rides along: explicit opt wins, else the live text
      // selection (same document) — "@doc + selected span" needs no guessing.
      // Range offsets are preserved so the context pack can scope the local
      // window to the exact span (in-file diff revise path).
      const liveSel: SelectionRef | undefined =
        state.selection && state.selection.documentId === documentId
          ? {
              documentId: state.selection.documentId,
              blockId: state.selection.blockId,
              text: state.selection.text,
              range: state.selection.range ? { ...state.selection.range } : undefined,
            }
          : undefined;
      const sel = opts?.selection ?? liveSel;

      // Focused pending proposal for chat-targeted revise ("shorten this
      // proposal"): explicit changeId wins; else the pending change on the
      // selected block; else revise-intent wording falls back to the target
      // summary's most recent pending change. Heuristic by design — the
      // model is told to touch it only when the user asks.
      const pendingInSummary = summary
        ? Object.values(state.changes)
            .filter((c) => c.documentId === summary.id && c.status === "pending")
            .sort((a, b) => b.createdAt - a.createdAt)
        : [];
      const focused = opts?.changeId
        ? state.changes[opts.changeId]
        : sel?.blockId
          ? pendingInSummary.find((c) => c.blockId === sel.blockId)
          : undefined;
      const reviseIntent = /\b(revise|shorten|expand|extend|rewrite|reword|rephrase|narrow|broaden|improve|change|edit|fix|update|lengthen)\b[^.?!]{0,40}\b(this|that|it|proposal|proposed|edit|change|draft|insertion|deletion|revision)\b|\b(this|that)\s+(proposal|proposed|edit|change|draft|insertion|deletion|revision)\b/i.test(trimmed);
      const focusedChange = focused ?? (reviseIntent ? pendingInSummary[0] : undefined);

      const userMsg: ChatMessage = {
        id: uid("msg"),
        role: "user",
        content: trimmed,
        createdAt: Date.now(),
        documentId,
      };
      dispatch({ type: "chat.append", threadId, message: userMsg });
      void appendChatMessage(threadId, userMsg);
      dispatch({ type: "reader.ui", patch: { aiPanelTab: "chat", aiPanelOpen: true } });

      const thread = state.chat.byId[threadId];
      const prior: ChatTurn[] = (thread?.messages ?? []).map((m) => ({
        role: m.role === "system" ? "system" : m.role === "assistant" ? "assistant" : "user",
        content: m.content,
      }));

      // The anchored selection scopes the pack's local window (±2 blocks
      // via localWindow) while the pack still carries title + outline +
      // summary snapshot — targeted edit, full-document context.
      const contextPack =
        opts?.includeContext !== false && doc
          ? await buildContextPackAsync(
              doc,
              sel?.text?.trim()
                ? {
                    blockId: sel.blockId,
                    text: sel.text,
                    range: sel.range ? { ...sel.range } : undefined,
                  }
                : undefined,
            )
          : undefined;

      const id = start("AI_CHAT", {
        documentId,
        workspaceId: wsId,
        message: "Thinking…",
      });

      cancels.current[id] = getAIProvider().run(
        {
          operation: "AI_CHAT",
          query: trimmed,
          documentId,
          workspaceId: wsId,
          selection: sel ? { blockId: sel.blockId, text: sel.text } : undefined,
          includeDocumentContext: !!contextPack,
          contextPack: contextPack || undefined,
          messageHistory: prior,
          mode: "chat",
          summaryContent: documentPlainText(summary) || undefined,
          primarySummaryId: summary?.id,
          summaryPath: summary?.metadata.path,
          mentionIds: mentionIds.length ? mentionIds : undefined,
          focusedChange: focusedChange
            ? {
                id: focusedChange.id,
                op: focusedChange.type,
                blockId: focusedChange.blockId,
                oldContent: focusedChange.oldContent || undefined,
                newContent: focusedChange.newContent || undefined,
              }
            : undefined,
        },
        {
          onPhase: (_i, label) =>
            dispatch({ type: "activity.update", id, patch: { message: `${label}…` } }),
          onBrowseSources: () => undefined,
          onDone: (result) => {
            const reply =
              result.explanation?.trim() ||
              "AI returned an empty reply. Try rephrasing or check the API key.";
            const assistantMsg: ChatMessage = {
              id: uid("msg"),
              role: "assistant",
              content: reply,
              createdAt: Date.now(),
              documentId,
              toolTrace: result.toolTrace?.length ? [...result.toolTrace] : undefined,
            };
            dispatch({ type: "chat.append", threadId, message: assistantMsg });
            void appendChatMessage(threadId, assistantMsg);

            // Published summaries (publish_summary): the .docx changed on
            // disk — reload blocks in place (like an editor refresh) and
            // decorate the write-through diffs in-file for review.
            if (result.publishedSummaries?.length) {
              void reloadPublishedSummaries(result.publishedSummaries);
            }

            // Phase 5: proposals → DocumentChanges (user still must accept).
            // Full edit power, not append-only: insert adds, modify rewrites
            // a block, delete strikes it. Legacy single `insertion` maps to
            // one insert for backward compat with older backends.
            const proposals = result.proposals?.length
              ? result.proposals
              : result.insertion?.text?.trim()
                ? [{ op: "insert" as const, newContent: `${result.insertion.text} ${result.insertion.citation ?? "(AI draft)"}` }]
                : [];
            let changeIds: string[] = [];
            // Single-path review (plan 11 §3): when the turn published, the
            // .docx already carries the edits and in-file diffs are the
            // review — materializing parallel card proposals would fork
            // reality. Only the unpublishable path keeps cards (and only
            // with an explicit no-file-path toast below).
            const publishedIds = new Set((result.publishedSummaries ?? []).map((p) => p.summaryId));
            const reviewInFile = summary && publishedIds.has(summary.id);
            // No summary mapped: proposals have nowhere to land. Say so
            // loudly (with the fix) instead of dropping them silently.
            let droppedForNoSummary = 0;
            for (const p of proposals) {
              if (!summary || !p.newContent?.trim()) {
                if (!summary && p.newContent?.trim()) droppedForNoSummary++;
                continue;
              }
              if (reviewInFile) continue;
              if (p.op === "modify" || p.op === "delete") {
                // Anchor to the named block; fall back to fuzzy-matching the
                // old span when the model returned text but no block id.
                const targetId = p.targetBlockId && summary.blocks.some((b) => b.id === p.targetBlockId)
                  ? p.targetBlockId
                  : findBlockForSpan(summary, p.oldContent ?? p.highlightFragment ?? "");
                if (!targetId) continue;
                const target = summary.blocks.find((b) => b.id === targetId);
                const oldText = p.oldContent?.trim()
                  || target?.segments.map((s) => s.text).join("") || "";
                const cid = uid("chg");
                dispatch({
                  type: "change.propose",
                  change: {
                    id: cid,
                    documentId: summary.id,
                    workspaceId: wsId,
                    type: p.op,
                    blockId: targetId,
                    oldContent: oldText,
                    newContent: p.op === "delete" ? "" : p.newContent,
                    highlightFragment: p.highlightFragment,
                    sourceId: doc?.sourceId ?? summary.sourceId,
                    activityId: id,
                    status: "pending",
                    createdAt: Date.now(),
                  },
                });
                changeIds = [...changeIds, cid];
                // Supersede: an older pending modify/delete on the same block
                // is now stale (the pending map renders one per block). Reject
                // flips status only — committed blocks are untouched — so the
                // user still sees the full audit trail.
                for (const stale of pendingInSummary) {
                  if (stale.id !== cid && stale.blockId === targetId && stale.type !== "insert") {
                    dispatch({ type: "change.decide", id: stale.id, status: "rejected" });
                  }
                }
              } else {
                const changeId = uid("chg");
                const blockId = p.targetBlockId && !summary.blocks.some((b) => b.id === p.targetBlockId)
                  ? p.targetBlockId : uid("s-ai");
                dispatch({
                  type: "change.propose",
                  change: {
                    id: changeId,
                    documentId: summary.id,
                    workspaceId: wsId,
                    type: "insert",
                    blockId,
                    oldContent: "",
                    newContent: p.newContent,
                    sourceId: doc?.sourceId ?? summary.sourceId,
                    activityId: id,
                    status: "pending",
                    createdAt: Date.now(),
                  },
                  block: {
                    id: blockId,
                    type: "paragraph",
                    changeId,
                    segments: [{ text: p.newContent }],
                  },
                });
                changeIds = [...changeIds, changeId];
              }
            }
            if (reviewInFile) {
              // Diffs are the review; the reload toast already fired. Silence.
            } else if (result.publishError) {
              // Backend diagnosed the unwritten turn — surface it verbatim.
              dispatch({ type: "toast", message: result.publishError });
            } else if (droppedForNoSummary > 0) {
              dispatch({
                type: "toast",
                message: "No workspace summary — open the target document and choose “Make summary”, then ask again.",
              });
            } else if (proposals.length > 0 && summary && !summary.metadata.path) {
              dispatch({
                type: "toast",
                message: "Summary has no file path — save it as a .docx under the workspace (Save DOCX), then ask again.",
              });
            } else if (changeIds.length) {
              dispatch({
                type: "toast",
                message: "Chat proposed summary edits — open the summary document to review them",
              });
            }

            finish(id, {
              status: "completed",
              message: changeIds.length ? "Chat reply + summary proposal" : "Chat reply",
              changeIds,
            });
          },
          onError: (message) => {
            const errMsg: ChatMessage = {
              id: uid("msg"),
              role: "assistant",
              content: `AI unavailable: ${message}`,
              createdAt: Date.now(),
              documentId,
            };
            dispatch({ type: "chat.append", threadId, message: errMsg });
            void appendChatMessage(threadId, errMsg);
            finish(id, { status: "error", error: message });
          },
        },
      );
    },
    [state, start, finish, dispatch, ensureThread, reloadPublishedSummaries],
  );

  return { runBrowse, runExplain, runIncludeInSummary, runRelatedSources, runChat, ensureThread, newThread };
}
