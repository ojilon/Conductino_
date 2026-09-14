/**
 * AI Reading panel (right, resizable, collapsible).
 *
 * Two faces, driven by the active document:
 *  - source documents  → "AI Reading" companion: selected passage,
 *                        explanation, related sources, follow-up actions
 *  - summary document  → "Review AI changes": pending DocumentChange
 *                        proposals with accept / reject / inspect / revise
 *
 * The panel also surfaces live AI activity so the app can always answer
 * "what is the AI doing right now, and on which document?"
 */

import { useEffect, useState } from "react";
import { useApp, runningActivity, pendingChangesFor } from "../../state/appState";
import { useAIRunners, type SelectionRef } from "../../state/aiController";
import type { Document, DocumentChange } from "../../types/domain";
import { Icon, type IconName } from "../../components/icons";
import { Button, EmptyState, IconBtn, Spinner } from "../../components/ui";
import { cn } from "../../utils/cn";
import { timeLabel, truncate } from "../../utils/helpers";

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
            {runningHere ? running?.message : companion.explanation}
          </p>
        </section>

        <section>
          <h4 className="mb-1.5 flex items-center gap-1.5 text-[12px] font-semibold text-ink-900">
            <Icon name="bookOpen" size={13} className="text-iris-600" /> Related sources
          </h4>
          <div className="space-y-1.5">
            {[
              { t: "Oxidative Phosphorylation and ATP Synthesis", m: "Nelson et al. (2021) · Nature Reviews Molecular Cell Biology" },
              { t: "Structure and Mechanism of ATP Synthase", m: "Petersen & Junge (2019) · Biochimica et Biophysica Acta" },
            ].map((r) => (
              <div key={r.t} className="flex items-start gap-2.5 rounded-lg border border-line-soft bg-cream-50 px-3 py-2">
                <Icon name="fileText" size={13} className="mt-0.5 shrink-0 text-ink-400" />
                <div className="min-w-0">
                  <p className="truncate text-[12px] font-medium text-ink-900">{r.t}</p>
                  <p className="truncate text-[10.5px] text-mute">{r.m}</p>
                </div>
              </div>
            ))}
          </div>
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
/* Review face (summary document)                                      */
/* ------------------------------------------------------------------ */

const CHANGE_LABEL: Record<DocumentChange["type"], string> = {
  insert: "Inserted paragraph",
  modify: "Modified sentence",
  delete: "Deleted passage",
};

const CHANGE_ICON: Record<DocumentChange["type"], IconName> = {
  insert: "plus",
  modify: "note",
  delete: "x",
};

function ReviewFace({ doc }: { doc: Document }) {
  const { state, dispatch } = useApp();
  const { runRevise } = useAIRunners();
  const pending = pendingChangesFor(state, doc.id);
  const [focusedId, setFocusedId] = useState<string | null>(pending[0]?.id ?? null);
  const [inspecting, setInspecting] = useState(false);

  useEffect(() => {
    if (focusedId && !pending.some((c) => c.id === focusedId)) {
      setFocusedId(pending[0]?.id ?? null);
      setInspecting(false);
    }
  }, [pending, focusedId]);

  const focused = pending.find((c) => c.id === focusedId) ?? null;
  const accepted = Object.values(state.changes).filter((c) => c.documentId === doc.id && c.status === "accepted").length;
  const rejected = Object.values(state.changes).filter((c) => c.documentId === doc.id && c.status === "rejected").length;

  const decide = (status: "accepted" | "rejected") => {
    if (!focused) return;
    dispatch({ type: "change.decide", id: focused.id, status });
    dispatch({ type: "toast", message: status === "accepted" ? "Change accepted" : "Change rejected" });
  };

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <div className="min-h-0 flex-1 space-y-4 overflow-y-auto px-4 py-4">
        <ActivityStrip />

        {/* pending list */}
        <section>
          <h4 className="mb-2 flex items-center justify-between text-[12px] font-semibold text-ink-900">
            Pending changes
            <span className="text-[11px] font-normal text-mute">{pending.length}</span>
          </h4>
          {pending.length === 0 ? (
            <div className="rounded-lg border border-moss-200 bg-moss-50 px-3 py-3 text-[12px] text-moss-700">
              <p className="flex items-center gap-2 font-medium">
                <Icon name="circleCheck" size={14} /> All changes reviewed
              </p>
              <p className="mt-1 text-[11.5px]">
                {accepted} accepted · {rejected} rejected. Select text in a source and choose “Include in
                summary” to add new material.
              </p>
            </div>
          ) : (
            <ul className="space-y-1.5">
              {pending.map((c) => (
                <li key={c.id}>
                  <button
                    type="button"
                    onClick={() => setFocusedId(c.id)}
                    className={cn(
                      "w-full rounded-lg border px-3 py-2 text-left transition-colors",
                      focusedId === c.id
                        ? "border-iris-300 bg-iris-50"
                        : "border-line-soft bg-cream-50 hover:border-line",
                    )}
                  >
                    <p className="flex items-center gap-1.5 text-[11.5px] font-semibold text-ink-900">
                      <Icon name={CHANGE_ICON[c.type]} size={12} className="text-hay-600" />
                      {CHANGE_LABEL[c.type]}
                      <span className="ml-auto font-normal text-mute">{timeLabel(c.createdAt)}</span>
                    </p>
                    <p className="mt-1 line-clamp-2 text-[11px] leading-snug text-ink-500">{truncate(c.newContent, 110)}</p>
                  </button>
                </li>
              ))}
            </ul>
          )}
        </section>

        {/* actions for focused change */}
        {focused && (
          <div className="space-y-2">
            <Button variant="solid" icon="check" className="w-full" onClick={() => decide("accepted")}>
              Accept
            </Button>
            <div className="grid grid-cols-3 gap-2">
              <Button variant="outline" icon="x" onClick={() => decide("rejected")}>
                Reject
              </Button>
              <Button
                variant={inspecting ? "soft" : "outline"}
                icon="search"
                onClick={() => setInspecting((v) => !v)}
              >
                Inspect
              </Button>
              <Button variant="outline" icon="refresh" onClick={() => runRevise(focused.id)}>
                Revise
              </Button>
            </div>
            {inspecting && (
              <div className="fade-in rounded-lg border border-line bg-cream-50 px-3 py-2.5 text-[11.5px] leading-relaxed">
                <p className="text-mute">
                  Before:{" "}
                  {focused.oldContent ? (
                    <span className="text-ink-500 line-through decoration-rose-300">{truncate(focused.oldContent, 160)}</span>
                  ) : (
                    <span className="italic">nothing (insert)</span>
                  )}
                </p>
                <p className="mt-1.5 text-mute">
                  After:{" "}
                  <span className="rounded-[3px] bg-hay-100 px-1 text-ink-700">{truncate(focused.newContent, 160)}</span>
                </p>
                <p className="mt-1.5 text-mute">
                  From: {state.sources[focused.sourceId]?.title ?? "session source"}
                </p>
              </div>
            )}
          </div>
        )}

        {/* legend */}
        <div className="space-y-1.5 rounded-lg border border-line-soft bg-cream-50 px-3 py-2.5">
          <p className="flex items-center gap-2 text-[11.5px] text-ink-700">
            <span className="h-3 w-5 rounded bg-hay-100 ring-1 ring-hay-200" /> Inserted
          </p>
          <p className="flex items-center gap-2 text-[11.5px] text-ink-700">
            <span className="h-0 w-5 border-b-2 border-hay-300" /> Modified
          </p>
        </div>
      </div>

      <div className="border-t border-line-soft px-4 py-3">
        <p className="flex items-center gap-2 text-[11.5px] font-medium text-iris-700">
          <Icon name="shieldCheck" size={14} /> You review every change
        </p>
      </div>
      <HistoryList documentId={doc.id} />
    </div>
  );
}

/* ------------------------------------------------------------------ */
/* Panel shell                                                         */
/* ------------------------------------------------------------------ */

export default function AIReadingPanel({ doc }: { doc: Document }) {
  const { dispatch } = useApp();
  const title = doc.kind === "summary" ? "Review AI changes" : "AI Reading";

  return (
    <div className="flex h-full min-h-0 flex-col">
      <div className="flex items-center justify-between border-b border-line-soft px-4 py-3">
        <div className="flex items-center gap-2">
          <Icon name="sparkles" size={15} className="text-iris-600" />
          <h2 className="font-serif text-[15px] font-semibold text-ink-900">{title}</h2>
        </div>
        <IconBtn
          name="x"
          title="Hide panel"
          onClick={() => dispatch({ type: "reader.ui", patch: { aiPanelOpen: false } })}
        />
      </div>

      {doc.kind === "summary" ? <ReviewFace doc={doc} /> : <CompanionFace doc={doc} />}
    </div>
  );
}
