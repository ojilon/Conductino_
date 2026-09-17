/**
 * Lumen — application shell.
 *
 * App = TopBar (logo · mode switcher · contextual actions) + active mode
 * + global overlays (settings, source preview, toast). Mode switching
 * only changes `AppState.mode`; both modes' state (and mounted views)
 * persist for the lifetime of the app.
 */

import { useEffect } from "react";
import { AppProvider, useApp, activeReaderDocument } from "./state/appState";
import { getAIProvider } from "./services/ai";
import { backend } from "./services/backend";
import { Icon } from "./components/icons";
import { IconBtn, Modal, Button, Badge } from "./components/ui";
import { cn } from "./utils/cn";
import BrowserMode from "./features/browser/BrowserMode";
import ReaderMode from "./features/reader/ReaderMode";
import { WindowMinimise, WindowToggleMaximise, Quit } from "../wailsjs/runtime/runtime";

/* ------------------------------------------------------------------ */
/* Top bar                                                             */
/* ------------------------------------------------------------------ */

function ModeSwitcher() {
  const { state, dispatch } = useApp();
  const mode = state.mode;
  return (
    <div className="flex rounded-full border border-line bg-cream-100/80 p-0.5">
      {(
        [
          { id: "reader", label: "Reader", icon: "bookOpen" },
          { id: "browser", label: "Browser", icon: "globe" },
        ] as const
      ).map((m) => (
        <button
          key={m.id}
          type="button"
          onClick={() => dispatch({ type: "mode.set", mode: m.id })}
          className={cn(
            "flex items-center gap-1.5 rounded-full px-4 py-1.5 text-[12.5px] font-medium transition-colors",
            mode === m.id ? "bg-iris-100 text-iris-700 shadow-sm" : "text-ink-500 hover:text-ink-900",
          )}
        >
          <Icon name={m.icon} size={14} />
          {m.label}
        </button>
      ))}
    </div>
  );
}

function WindowControls() {
  return (
    <div className="ml-1 flex items-center gap-0.5" style={{ "--wails-draggable": "no-drag" } as React.CSSProperties}>
      <IconBtn name="minus" title="Minimize" onClick={() => WindowMinimise()} />
      <IconBtn name="maximize" title="Maximize / Restore" onClick={() => WindowToggleMaximise()} />
      <IconBtn name="x" title="Close" onClick={() => Quit()} />
    </div>
  );
}

function TopBar() {
  const { state, dispatch } = useApp();
  const doc = activeReaderDocument(state);
  const isSummary = doc?.kind === "summary";
  const contribCount = doc?.sourceIds?.length ?? 0;

  return (
    <header
      className="flex h-12 shrink-0 items-center gap-3 border-b border-line bg-paper px-4"
      style={{ "--wails-draggable": "drag" } as React.CSSProperties}
    >
      <div className="flex w-[210px] items-center gap-2.5">
        <Icon name="logo" size={22} className="text-iris-600" />
        <span className="font-serif text-[19px] font-bold tracking-tight text-ink-900">Lumen</span>
      </div>

      <div
        className="flex flex-1 justify-center"
        style={{ "--wails-draggable": "no-drag" } as React.CSSProperties}
      >
        <ModeSwitcher />
      </div>

      <div
        className="flex w-[210px] items-center justify-end gap-1.5"
        style={{ "--wails-draggable": "no-drag" } as React.CSSProperties}
      >
        {state.mode === "reader" && isSummary && (
          <span className="mr-1 hidden items-center gap-1.5 text-[11.5px] text-ink-500 lg:flex">
            <span className="h-1.5 w-1.5 rounded-full bg-iris-500" />
            {contribCount} sources contribute to this summary
          </span>
        )}
        {state.mode === "reader" ? (
          <button
            type="button"
            title="Toggle AI Reading panel"
            onClick={() =>
              dispatch({ type: "reader.ui", patch: { aiPanelOpen: !state.reader.ui.aiPanelOpen } })
            }
            className={cn(
              "flex items-center gap-1.5 rounded-full border px-3 py-1.5 text-[12px] font-medium transition-colors",
              state.reader.ui.aiPanelOpen
                ? "border-moss-200 bg-moss-50 text-moss-700"
                : "border-line text-ink-500 hover:text-ink-900",
            )}
          >
            <Icon name="sparkles" size={13} />
            AI Reading
            <Icon name="chevronDown" size={11} className="opacity-60" />
          </button>
        ) : (
          <IconBtn
            name="search"
            title="Search (mock index)"
            onClick={() => {
              const session = state.browser.sessions.find((s) => s.id === state.browser.activeSessionId);
              const tabId = session?.activeTabId;
              if (tabId) {
                dispatch({
                  type: "browser.navigate",
                  tabId,
                  url: `lumen://search?q=${encodeURIComponent(state.browse.query || "chemiosmosis ATP synthesis")}`,
                });
              }
            }}
          />
        )}
        <IconBtn name="settings" title="Settings" onClick={() => dispatch({ type: "settings.set", open: true })} />
        <span
          title="Local profile"
          className="flex h-7 w-7 items-center justify-center rounded-full bg-cream-200 text-[10.5px] font-semibold text-ink-700"
        >
          JD
        </span>
        <WindowControls />
      </div>
    </header>
  );
}
/* ------------------------------------------------------------------ */
/* Settings dialog — honest status of every subsystem                  */
/* ------------------------------------------------------------------ */

function Row({ icon, name, status, note, tone }: { icon: Parameters<typeof Icon>[0]["name"]; name: string; status: string; note: string; tone: "ok" | "mock" }) {
  return (
    <div className="flex items-start gap-3 rounded-lg border border-line-soft bg-cream-50 px-3.5 py-3">
      <Icon name={icon} size={15} className="mt-0.5 shrink-0 text-ink-400" />
      <div className="min-w-0 flex-1">
        <div className="flex items-center gap-2">
          <p className="text-[12.5px] font-semibold text-ink-900">{name}</p>
          <Badge tone={tone === "ok" ? "moss" : "hay"}>{status}</Badge>
        </div>
        <p className="mt-0.5 text-[11.5px] leading-relaxed text-mute">{note}</p>
      </div>
    </div>
  );
}

function SettingsDialog() {
  const { state, dispatch } = useApp();
  if (!state.settingsOpen) return null;
  const provider = getAIProvider();
  return (
    <Modal title="Settings" subtitle="Subsystem status and integration points" onClose={() => dispatch({ type: "settings.set", open: false })} width={560}>
      <div className="space-y-2.5">
        <Row
          icon="sparkles"
          name="AI provider"
          status={provider.configured ? "Connected" : "Mock service"}
          tone={provider.configured ? "ok" : "mock"}
          note={`Active: ${provider.name}. The key lives in Go only (GEMINI_API_KEY env or git-ignored backend/.ai.env) — the frontend never sees it. docs/ai-integration.md`}
        />
        <Row
          icon="layers"
          name="Storage"
          status={`${backend.storage.engine === "sqlite" ? "SQLite" : "In-memory"} · boundary ready`}
          tone={backend.storage.engine === "sqlite" ? "ok" : "mock"}
          note="Persistence interface in src/services/backend.ts; the SQLite implementation belongs in backend/storage/store.go (Go side)."
        />
        <Row
          icon="folder"
          name="Filesystem"
          status={backend.mode === "wails" ? "Go backend" : "Mock"}
          tone={backend.mode === "wails" ? "ok" : "mock"}
          note="ShowContainingFolder, listRoot etc. are dispatched through the service boundary — the React UI never touches paths."
        />
        <Row
          icon="globe"
          name="Browser engine"
          status="Placeholder (mock pages)"
          tone="mock"
          note="BrowserView boundary in src/features/browser/MockWebPage.tsx. A real webview plugs in there only. docs/architecture.md §Browser"
        />
        <Row
          icon="fileText"
          name="Document renderer"
          status="Mock structured renderer"
          tone="mock"
          note="Documents render from the DocumentModel. pdf.js / docx-preview integrate per-format without touching app state. docs/document-rendering.md"
        />
      </div>
      <Button variant="outline" className="mt-4 w-full" onClick={() => dispatch({ type: "settings.set", open: false })}>
        Close
      </Button>
    </Modal>
  );
}

/* ------------------------------------------------------------------ */
/* Source preview (from AI Browse cards)                               */
/* ------------------------------------------------------------------ */

function SourcePreviewDialog() {
  const { state, dispatch } = useApp();
  const src = state.previewSourceId ? state.sources[state.previewSourceId] : undefined;
  if (!src) return null;
  const close = () => dispatch({ type: "source.preview", id: null });
  return (
    <Modal title={src.title} subtitle={`${src.origin} · ${src.typeLabel}`} onClose={close} width={480}>
      {src.relevance && (
        <div className="mb-3 flex items-center gap-2">
          <Badge tone={src.relevance === "highest" ? "moss" : src.relevance === "high" ? "iris" : "blue"}>
            {src.relevance === "highest" ? "Highest relevance" : src.relevance === "high" ? "High relevance" : "Good relevance"}
          </Badge>
          {src.rank && <span className="text-[11.5px] text-mute">rank {src.rank}</span>}
        </div>
      )}
      <p className="text-[13px] leading-relaxed text-ink-700">{src.abstract}</p>
      <p className="mt-3 rounded-md bg-cream-100 px-3 py-2 text-[11.5px] leading-relaxed text-mute">
        Full text is extracted on demand by the source service (mock) — a real pipeline lives in
        backend/services/sources.go.
      </p>
      <div className="mt-4 flex gap-2">
        <Button
          variant="solid"
          icon="external"
          onClick={() => {
            if (src.url) {
              const session = state.browser.sessions.find((s) => s.id === state.browser.activeSessionId);
              if (session?.activeTabId) {
                dispatch({ type: "browser.navigate", tabId: session.activeTabId, url: src.url });
              }
            }
            close();
            dispatch({ type: "mode.set", mode: "browser" });
          }}
        >
          Open in browser
        </Button>
        <Button variant="outline" icon={src.saved ? "bookmark" : "plus"} onClick={() => dispatch({ type: "source.toggleSave", id: src.id })}>
          {src.saved ? "Unsave" : "Save"}
        </Button>
      </div>
    </Modal>
  );
}

/* ------------------------------------------------------------------ */
/* Toast                                                               */
/* ------------------------------------------------------------------ */

function Toast() {
  const { state, dispatch } = useApp();
  useEffect(() => {
    if (!state.toast) return;
    const t = setTimeout(() => dispatch({ type: "toast", message: null }), 3600);
    return () => clearTimeout(t);
  }, [state.toast, dispatch]);
  if (!state.toast) return null;
  return (
    <div className="fade-in fixed bottom-6 left-1/2 z-[80] -translate-x-1/2 rounded-full bg-ink-900 px-4 py-2 text-[12.5px] text-white shadow-xl">
      {state.toast.message}
    </div>
  );
}

/* ------------------------------------------------------------------ */

function Shell() {
  const { state } = useApp();
  return (
    <div className="flex h-full flex-col bg-cream-50">
      <TopBar />
      <main className="min-h-0 flex-1">
        {state.mode === "browser" ? <BrowserMode /> : <ReaderMode />}
      </main>
      <SettingsDialog />
      <SourcePreviewDialog />
      <Toast />
    </div>
  );
}

export default function App() {
  return (
    <AppProvider>
      <Shell />
    </AppProvider>
  );
}
