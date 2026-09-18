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
import { buildContextPack } from "./contextPack";
import { getAIProvider } from "../services/ai";
import { uid } from "../utils/helpers";
import type { AIActivity, AIOperation, ChatMessage, ChatThread, ChatTurn, Document } from "../types/domain";
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
    (operation: AIOperation, selection: SelectionRef, customPrompt?: string) => {
      const doc = state.documents[selection.documentId];
      const meta = doc?.metadata;
      const sourceLabel = meta?.author
        ? `${meta.author}${meta.year ? ` (${meta.year})` : ""}${meta.venue ? `, ${meta.venue}` : ""}`
        : "Open document";
      const wsId = activeWorkspace(state)?.id ?? doc?.workspaceId;
      const contextPack = buildContextPack(doc, {
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
    (selection: SelectionRef) => {
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
      const contextPack = buildContextPack(sourceDoc, {
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
                status: "pending",
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
            dispatch({ type: "toast", message: "Change proposed in Research Summary — review it in the document" });
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

  /* ---------- Revise a pending change ---------- */

  const runRevise = useCallback(
    (changeId: string) => {
      const change = state.changes[changeId];
      if (!change) return;
      const id = start("AI_REWRITE", {
        documentId: change.documentId,
        message: "Revising draft…",
      });
      cancels.current[id] = getAIProvider().run(
        { operation: "AI_REWRITE", query: change.newContent, changeId },
        {
          onPhase: (_i, label) =>
            dispatch({ type: "activity.update", id, patch: { message: `${label}…` } }),
          onBrowseSources: () => undefined,
          onDone: (result) => {
            if (result.revision?.trim()) {
              dispatch({ type: "change.revise", id: changeId, newContent: result.revision });
              finish(id, { status: "completed", message: "Revised the proposed change", changeIds: [changeId] });
              dispatch({ type: "toast", message: "AI revised the change — review again" });
            } else {
              finish(id, { status: "completed", message: "Revise finished with no suggestion", changeIds: [changeId] });
              dispatch({ type: "toast", message: "AI unavailable now — change left as-is." });
            }
          },
          onError: (message) => {
            finish(id, { status: "error", error: message });
            dispatch({ type: "toast", message: "AI unavailable now — change left as-is." });
          },
        },
      );
    },
    [state.changes, start, finish, dispatch],
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

  /** Ensure a workspace-scoped thread exists and is active; returns its id. */
  const ensureThread = useCallback(
    (documentId?: string): string => {
      const ws = activeWorkspace(state);
      const wsId = ws?.id ?? "ws-demo";
      const existing = Object.values(state.chat.byId).find(
        (t) => t.workspaceId === wsId && (documentId ? t.documentId === documentId : true),
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
      return thread.id;
    },
    [state, dispatch],
  );

  const runChat = useCallback(
    (text: string, opts?: { documentId?: string; includeContext?: boolean; selection?: SelectionRef; changeId?: string }) => {
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
      const liveSel = state.selection && state.selection.documentId === documentId
        ? { documentId: state.selection.documentId, blockId: state.selection.blockId, text: state.selection.text }
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
      dispatch({ type: "reader.ui", patch: { aiPanelTab: "chat", aiPanelOpen: true } });

      const thread = state.chat.byId[threadId];
      const prior: ChatTurn[] = (thread?.messages ?? []).map((m) => ({
        role: m.role === "system" ? "system" : m.role === "assistant" ? "assistant" : "user",
        content: m.content,
      }));

      const contextPack =
        opts?.includeContext !== false && doc
          ? buildContextPack(doc, undefined)
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
            };
            dispatch({ type: "chat.append", threadId, message: assistantMsg });

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
            for (const p of proposals) {
              if (!summary || !p.newContent?.trim()) continue;
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
            if (changeIds.length) {
              dispatch({
                type: "toast",
                message: "Chat proposed summary edits — review them on the summary tab",
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
            finish(id, { status: "error", error: message });
          },
        },
      );
    },
    [state, start, finish, dispatch, ensureThread],
  );

  return { runBrowse, runExplain, runIncludeInSummary, runRevise, runRelatedSources, runChat, ensureThread };
}
