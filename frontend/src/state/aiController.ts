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
import { useApp, activeReaderDocument } from "./appState";
import { getAIProvider } from "../services/ai";
import { uid } from "../utils/helpers";
import type { AIActivity, AIOperation } from "../types/domain";

export interface SelectionRef {
  documentId: string;
  blockId: string;
  text: string;
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
    (operation: AIOperation, selection: SelectionRef) => {
      const doc = state.documents[selection.documentId];
      const meta = doc?.metadata;
      const sourceLabel = meta?.author ? `${meta.author}${meta.year ? ` (${meta.year})` : ""}${meta.venue ? `, ${meta.venue}` : ""}` : "Open document";
      const id = start(operation, {
        documentId: selection.documentId,
        selection: selection.text,
        message: "Reading selection…",
      });
      cancels.current[id] = getAIProvider().run(
        { operation, documentId: selection.documentId, selection: { blockId: selection.blockId, text: selection.text } },
        {
          onPhase: (_i, label) =>
            dispatch({ type: "activity.update", id, patch: { message: `${label}…` } }),
          onBrowseSources: () => undefined,
          onDone: (result) => {
            dispatch({
              type: "reader.ui",
              patch: {
                companion: {
                  documentId: selection.documentId,
                  passage: selection.text,
                  explanation: result.explanation ?? "",
                  sourceLabel,
                },
              },
            });
            finish(id, { status: "completed", message: `${operation.replace("AI_", "").toLowerCase()} completed` });
          },
          onError: (message) => finish(id, { status: "error", error: message }),
        },
      );
    },
    [state.documents, start, finish, dispatch],
  );

  /* ---------- Include selection in the summary (source → summary) ---------- */

  const runIncludeInSummary = useCallback(
    (selection: SelectionRef) => {
      const summary = Object.values(state.documents).find((d) => d.kind === "summary");
      if (!summary) {
        dispatch({ type: "toast", message: "No summary document in this session" });
        return;
      }
      const sourceDoc = state.documents[selection.documentId];
      const sourceId = sourceDoc?.sourceId ?? selection.documentId;
      const id = start("AI_MERGE", {
        documentId: summary.id,
        sourceId,
        selection: selection.text,
        message: "Extracting claim…",
      });
      cancels.current[id] = getAIProvider().run(
        { operation: "AI_MERGE", documentId: selection.documentId, sourceId, selection: { blockId: selection.blockId, text: selection.text } },
        {
          onPhase: (_i, label) =>
            dispatch({ type: "activity.update", id, patch: { message: `${label}…` } }),
          onBrowseSources: () => undefined,
          onDone: (result) => {
            const insertion = result.insertion;
            const changeId = uid("chg");
            const blockId = uid("s-ai");
            const text = insertion ? `${insertion.text} ${insertion.citation}` : `— “${selection.text.slice(0, 120)}”`;
            dispatch({
              type: "change.propose",
              change: {
                id: changeId,
                documentId: summary.id,
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
          onError: (message) => finish(id, { status: "error", error: message }),
        },
      );
    },
    [state.documents, start, finish, dispatch],
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
            if (result.revision) {
              dispatch({ type: "change.revise", id: changeId, newContent: result.revision });
            }
            finish(id, { status: "completed", message: "Revised the proposed change", changeIds: [changeId] });
            dispatch({ type: "toast", message: "AI revised the change — review again" });
          },
          onError: (message) => finish(id, { status: "error", error: message }),
        },
      );
    },
    [state.changes, start, finish, dispatch],
  );

  /* ---------- Companion panel: find related sources ---------- */

  const runRelatedSources = useCallback(() => {
    const doc = activeReaderDocument(state);
    const id = start("AI_SEARCH", {
      documentId: doc?.id,
      message: "Finding related sources…",
    });
    cancels.current[id] = getAIProvider().run(
      { operation: "AI_SEARCH", query: "related sources", documentId: doc?.id },
      {
        onPhase: (_i, label) => dispatch({ type: "activity.update", id, patch: { message: `${label}…` } }),
        onBrowseSources: () => undefined,
        onDone: () => {
          finish(id, { status: "completed", message: "Refreshed related sources" });
          dispatch({ type: "toast", message: "Related sources refreshed (mock provider)" });
        },
        onError: (message) => finish(id, { status: "error", error: message }),
      },
    );
  }, [state, start, finish, dispatch]);

  return { runBrowse, runExplain, runIncludeInSummary, runRevise, runRelatedSources };
}
