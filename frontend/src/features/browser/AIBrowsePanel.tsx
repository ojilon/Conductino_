/**
 * AI Browse panel (right, resizable, collapsible).
 *
 * Flow: query → AI request (via provider interface) → streaming phases
 * → ranked sources → preview / open / save / send-to-reader.
 * The panel reads BrowseState + AIActivity from app state, so the same
 * UI works when the mock provider is swapped for a real one.
 */

import { useState } from "react";
import { useApp, activeBrowserTab } from "../../state/appState";
import { useAIRunners } from "../../state/aiController";
import { BROWSE_PHASES, type Relevance, type Source } from "../../types/domain";
import { Icon } from "../../components/icons";
import { Badge, Button, EmptyState, IconBtn, Menu, MenuItem, Spinner } from "../../components/ui";
import { cn } from "../../utils/cn";
import { truncate, uid } from "../../utils/helpers";
import { makeDocumentFromSource } from "../../mock/data";

const RELEVANCE: Record<Relevance, { label: string; tone: "moss" | "iris" | "blue" }> = {
  highest: { label: "Highest relevance", tone: "moss" },
  high: { label: "High relevance", tone: "iris" },
  good: { label: "Good relevance", tone: "blue" },
};

const RANK_TONE: Record<Relevance, string> = {
  highest: "bg-moss-100 text-moss-700",
  high: "bg-iris-100 text-iris-700",
  good: "bg-sky-100 text-sky-700",
};

/* ------------------------------------------------------------------ */
/* Progress steps (streaming AI activity)                              */
/* ------------------------------------------------------------------ */

function ProgressSteps({ phase, status }: { phase: number; status: string }) {
  const allDone = status === "done";
  return (
    <div className="flex items-start">
      {BROWSE_PHASES.map((label, i) => {
        const done = allDone || i < phase;
        const current = !allDone && status === "running" && i === phase;
        return (
          <div key={label} className={cn("flex items-start", i > 0 && "flex-1")}>
            {i > 0 && (
              <div
                className={cn("mt-[9px] h-px min-w-3 flex-1", done || current ? "bg-iris-400" : "bg-line")}
              />
            )}
            <div className="flex w-[68px] shrink-0 flex-col items-center gap-1.5">
              {done ? (
                <span className="flex h-[18px] w-[18px] items-center justify-center rounded-full bg-iris-600 text-white">
                  <Icon name="check" size={10} strokeWidth={3} />
                </span>
              ) : current ? (
                <span className="flex h-[18px] w-[18px] items-center justify-center rounded-full border-2 border-iris-500 bg-white">
                  <span className="h-1.5 w-1.5 animate-pulse rounded-full bg-iris-500" />
                </span>
              ) : (
                <span className="h-[18px] w-[18px] rounded-full border-2 border-line bg-white" />
              )}
              <span
                className={cn(
                  "text-center text-[10px] leading-[1.25]",
                  current ? "font-medium text-iris-700" : done ? "text-ink-500" : "text-mute",
                )}
              >
                {label}
              </span>
            </div>
          </div>
        );
      })}
    </div>
  );
}

/* ------------------------------------------------------------------ */
/* Source card                                                         */
/* ------------------------------------------------------------------ */

function SourceCard({ source }: { source: Source }) {
  const { state, dispatch } = useApp();
  const rel = source.relevance ? RELEVANCE[source.relevance] : undefined;

  const openInReader = (src: Source) => {
    const existing = src.documentId;
    if (existing) {
      dispatch({ type: "reader.doc.open", tabId: uid("rt"), documentId: existing, label: truncate(src.title, 20) });
      dispatch({ type: "toast", message: "Document opened in Reader" });
      return;
    }
    const docId = uid("doc");
    const doc = makeDocumentFromSource(src.id, src, docId);
    dispatch({ type: "doc.add", document: doc });
    dispatch({ type: "source.markReader", id: src.id, documentId: docId });
    dispatch({ type: "reader.doc.open", tabId: uid("rt"), documentId: docId, label: truncate(src.title, 20) });
    dispatch({ type: "toast", message: "Sent to Reader — open the Reader mode to view it" });
  };

  const open = () => {
    const tab = activeBrowserTab(state);
    if (source.kind === "pdf") {
      openInReader(source);
      return;
    }
    if (source.url && tab) dispatch({ type: "browser.navigate", tabId: tab.id, url: source.url });
    else dispatch({ type: "toast", message: "No URL for this source" });
  };

  const act = (label: string, icon: "external" | "eye" | "bookmark" | "bookOpen", onClick: () => void) => (
    <button
      key={label}
      type="button"
      onClick={onClick}
      className="flex items-center gap-1.5 rounded-md px-2 py-1 text-[11.5px] text-ink-500 transition-colors hover:bg-iris-50 hover:text-iris-700"
    >
      <Icon name={icon} size={12} /> {label}
    </button>
  );

  return (
    <div className="rounded-lg border border-line bg-paper p-3 transition-colors hover:border-iris-200">
      <div className="flex gap-3">
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-2">
            <span
              className={cn(
                "flex h-[22px] w-[22px] shrink-0 items-center justify-center rounded-md text-[11.5px] font-bold",
                RANK_TONE[source.relevance ?? "good"],
              )}
            >
              {source.rank}
            </span>
            {rel && <Badge tone={rel.tone}>{rel.label}</Badge>}
          </div>
          <div className="mt-2 flex items-center gap-1.5 text-[11px] text-mute">
            <Icon name="globe" size={11} />
            {source.origin}
            <span className="ml-auto truncate">{source.typeLabel}</span>
          </div>
          <button
            type="button"
            onClick={open}
            className="mt-1 block text-left text-[13px] font-semibold leading-snug text-[#1a56c4] hover:underline"
          >
            {source.title}
          </button>
          <p className="mt-1 line-clamp-3 text-[11.5px] leading-relaxed text-ink-500">{source.abstract}</p>
        </div>
        {/* thumbnail */}
        <div className="flex h-[62px] w-12 shrink-0 items-center justify-center rounded border border-line-soft bg-cream-50">
          {source.kind === "pdf" ? (
            <div className="flex flex-col items-center gap-0.5 text-[#c42b2b]">
              <Icon name="fileText" size={18} />
              <span className="text-[8px] font-bold">PDF</span>
            </div>
          ) : (
            <div className="w-8 space-y-1">
              {[7, 8, 6, 8, 5].map((w, i) => (
                <div key={i} className="h-1 rounded bg-line" style={{ width: `${w * 10}%` }} />
              ))}
            </div>
          )}
        </div>
      </div>

      <div className="mt-2 flex items-center gap-0.5 overflow-x-auto border-t border-line-soft pt-1.5">
        {act("Open", "external", open)}
        {act("Preview", "eye", () => dispatch({ type: "source.preview", id: source.id }))}
        {act(source.saved ? "Unsave" : "Save", "bookmark", () => {
          dispatch({ type: "source.toggleSave", id: source.id });
          dispatch({ type: "toast", message: source.saved ? "Removed from saved sources" : "Source saved to workspace" });
        })}
        {act("Send to Reader", "bookOpen", () => openInReader(source))}
        <Menu
          width={190}
          button={(open) => <IconBtn name="dotsH" title="More actions" active={open} size={14} />}
        >
          {(close) => (
            <>
              <MenuItem
                icon="plus"
                label="Open in new browser tab"
                onClick={() => {
                  if (!source.url) return;
                  const tabId = uid("tab");
                  const session = state.browser.sessions.find((s) => s.id === state.browser.activeSessionId);
                  dispatch({ type: "browser.tab.add", tabId, sessionId: session?.id ?? "sess-1" });
                  dispatch({ type: "browser.navigate", tabId, url: source.url });
                  close();
                }}
              />
              <MenuItem
                icon="copy"
                label="Copy link"
                onClick={() => {
                  navigator.clipboard?.writeText(source.url ?? source.origin);
                  dispatch({ type: "toast", message: "Link copied" });
                  close();
                }}
              />
            </>
          )}
        </Menu>
      </div>
    </div>
  );
}

/* ------------------------------------------------------------------ */
/* Panel                                                               */
/* ------------------------------------------------------------------ */

export default function AIBrowsePanel() {
  const { state, dispatch } = useApp();
  const { runBrowse } = useAIRunners();
  const [query, setQuery] = useState(state.browse.query);

  const { browse } = state;
  const running = state.aiActivities.find((a) => a.status === "running" && a.operation === "AI_SEARCH");
  const sources = browse.sourceIds.map((id) => state.sources[id]).filter(Boolean);

  const startBrowse = () => {
    const q = query.trim();
    dispatch({ type: "browser.ui", patch: { aiPanelOpen: true } });
    runBrowse(q);
  };

  return (
    <div className="flex h-full min-h-0 flex-col">
      {/* header */}
      <div className="flex items-center justify-between border-b border-line-soft px-4 py-3">
        <div className="flex items-center gap-2">
          <Icon name="sparkles" size={15} className="text-iris-600" />
          <h2 className="text-[14px] font-semibold text-ink-900">AI Browse</h2>
          {browse.status === "running" && <Spinner size={12} className="text-iris-600" />}
        </div>
        <div className="flex items-center gap-0.5">
          <IconBtn
            name="chevronDown"
            title="Collapse panel"
            onClick={() => dispatch({ type: "browser.ui", patch: { aiPanelOpen: false } })}
          />
          <IconBtn
            name="x"
            title="Close and reset"
            onClick={() => {
              dispatch({ type: "browse.reset" });
              dispatch({ type: "browser.ui", patch: { aiPanelOpen: false } });
            }}
          />
        </div>
      </div>

      <div className="min-h-0 flex-1 overflow-y-auto px-4 py-4">
        {browse.status === "error" ? (
          <div className="rounded-lg border border-rose-200 bg-rose-50 px-3 py-3 text-[12px] text-rose-700">
            <p className="flex items-center gap-2 font-medium">
              <Icon name="alert" size={14} /> AI Browse failed
            </p>
            <p className="mt-1 leading-relaxed">{browse.error}</p>
            <Button size="sm" variant="solid" className="mt-2" onClick={startBrowse}>
              Try again
            </Button>
          </div>
        ) : browse.status === "idle" ? (
          <EmptyState
            icon="sparkles"
            title="No AI Browse results yet"
            hint="Ask a question below and Lumen will search, find and rank the most relevant sources for your session."
          />
        ) : (
          <div className="space-y-4">
            <ProgressSteps phase={browse.phase} status={browse.status} />
            {running && (
              <p className="flex items-center gap-2 text-[11.5px] text-mute">
                <Spinner size={11} className="text-iris-500" /> {running.message}
              </p>
            )}

            <div>
              <div className="mb-2 flex items-center justify-between">
                <h3 className="text-[12.5px] font-semibold text-ink-900">AI-ranked sources</h3>
                {browse.status === "done" && (
                  <span className="text-[11.5px] text-mute">{sources.length} results</span>
                )}
              </div>
              {browse.status === "running" && sources.length === 0 ? (
                <div className="space-y-2">
                  {[0, 1, 2].map((i) => (
                    <div key={i} className="h-28 animate-pulse rounded-lg bg-cream-100/80" />
                  ))}
                </div>
              ) : (
                <div className="space-y-2.5">
                  {sources.map((s) => (
                    <SourceCard key={s.id} source={s} />
                  ))}
                </div>
              )}
            </div>
          </div>
        )}
      </div>

      {/* manual query */}
      <div className="border-t border-line-soft px-4 py-3">
        <div className="mb-1.5 flex items-center justify-between">
          <span className="text-[12px] font-semibold text-ink-900">Manual query</span>
          <span className="text-[10.5px] text-mute">ranked by the AI provider</span>
        </div>
        <div className="flex items-center gap-1.5">
          <div className="flex h-9 min-w-0 flex-1 items-center rounded-lg border border-line bg-cream-50 px-3 focus-within:border-iris-300 focus-within:bg-white">
            <input
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              onKeyDown={(e) => e.key === "Enter" && startBrowse()}
              placeholder="e.g. mechanism of ATP production during chemiosmosis"
              className="w-full bg-transparent text-[12.5px] text-ink-700 outline-none placeholder:text-mute"
            />
          </div>
          <IconBtn name="search" title="AI Browse" className="bg-iris-100 text-iris-700" onClick={startBrowse} />
        </div>
        <div className="mt-2 flex gap-2">
          <Button
            variant="outline"
            className="flex-1"
            onClick={() => {
              const tab = activeBrowserTab(state);
              if (tab) {
                dispatch({
                  type: "browser.navigate",
                  tabId: tab.id,
                  url: `lumen://search?q=${encodeURIComponent(query)}`,
                });
              }
            }}
          >
            Search normally
          </Button>
          <Button variant="solid" className="flex-1" icon="sparkles" onClick={startBrowse}>
            AI Browse
          </Button>
        </div>
      </div>
    </div>
  );
}
