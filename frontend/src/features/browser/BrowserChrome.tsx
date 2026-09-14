/**
 * Browser chrome: research-session header, page tabs and the address bar.
 * Navigation state lives in state (BrowserTab.nav); this component only
 * renders it and expresses intent via actions.
 */

import { useEffect, useState } from "react";
import { useApp, activeBrowserSession, activeBrowserTab } from "../../state/appState";
import { Icon, type IconName } from "../../components/icons";
import { IconBtn, Menu, MenuItem, Badge } from "../../components/ui";
import { cn } from "../../utils/cn";
import { hostOf, uid } from "../../utils/helpers";

function tabIcon(url: string): IconName {
  if (url.startsWith("lumen://search")) return "search";
  if (url === "lumen://newtab") return "globe";
  return "fileText";
}

export function SessionTabs() {
  const { state, dispatch } = useApp();
  const session = activeBrowserSession(state);

  return (
    <div className="mt-3 flex items-end gap-1 overflow-x-auto pb-px">
      {session.tabIds.map((tabId) => {
        const tab = state.browser.tabs[tabId];
        if (!tab) return null;
        const active = tabId === session.activeTabId;
        return (
          <div
            key={tabId}
            className={cn(
              "group flex shrink-0 cursor-pointer items-center gap-1.5 rounded-t-lg border border-b-0 px-3.5 py-2 text-[12.5px] transition-colors",
              active
                ? "border-line bg-white font-medium text-ink-900"
                : "border-transparent text-ink-500 hover:bg-cream-100/70",
            )}
            onClick={() => dispatch({ type: "browser.tab.select", tabId })}
          >
            <Icon name={tabIcon(tab.nav.currentUrl)} size={13} className={active ? "text-iris-600" : "opacity-50"} />
            <span className="max-w-[140px] truncate">{tab.title}</span>
            <button
              type="button"
              title="Close tab"
              className="rounded p-0.5 text-ink-400 opacity-0 transition-opacity hover:bg-cream-100 hover:text-ink-900 group-hover:opacity-100"
              onClick={(e) => {
                e.stopPropagation();
                dispatch({ type: "browser.tab.close", tabId });
              }}
            >
              <Icon name="x" size={11} />
            </button>
          </div>
        );
      })}
      <button
        type="button"
        title="New tab"
        className="mb-0.5 flex shrink-0 items-center gap-1 rounded-lg px-2.5 py-1.5 text-[12.5px] text-ink-500 transition-colors hover:bg-cream-100 hover:text-ink-900"
        onClick={() =>
          dispatch({ type: "browser.tab.add", tabId: uid("tab"), sessionId: session.id })
        }
      >
        <Icon name="plus" size={13} />
      </button>
    </div>
  );
}

export default function BrowserChrome() {
  const { state, dispatch } = useApp();
  const session = activeBrowserSession(state);
  const tab = activeBrowserTab(state);
  const aiOpen = state.browser.ui.aiPanelOpen;

  const [draft, setDraft] = useState(tab?.nav.currentUrl ?? "");
  useEffect(() => {
    setDraft(tab?.nav.currentUrl ?? "");
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [tab?.id, tab?.nav.currentUrl]);

  const navigate = (url: string) => tab && dispatch({ type: "browser.navigate", tabId: tab.id, url });

  return (
    <div>
      {/* session header */}
      <div className="flex items-center gap-2.5 rounded-lg border border-iris-200/70 bg-iris-50/70 px-4 py-2">
        <Icon name="flask" size={15} className="text-iris-600" />
        <span className="text-[13.5px] font-semibold text-ink-900">
          Research session: <span className="font-medium">{session.title}</span>
        </span>
        <span className="ml-auto text-[11.5px] text-mute">{session.tabIds.length} pages</span>
      </div>

      <SessionTabs />

      {/* address bar */}
      <div className="mt-1.5 flex items-center gap-1 rounded-b-lg rounded-tr-lg border border-line bg-white px-2 py-1.5 shadow-sm">
        <IconBtn
          name="arrowLeft"
          title="Back"
          disabled={!tab || tab.nav.index === 0}
          onClick={() => tab && dispatch({ type: "browser.nav", tabId: tab.id, dir: -1 })}
        />
        <IconBtn
          name="arrowRight"
          title="Forward"
          disabled={!tab || tab.nav.index >= tab.nav.history.length - 1}
          onClick={() => tab && dispatch({ type: "browser.nav", tabId: tab.id, dir: 1 })}
        />
        <IconBtn
          name="refresh"
          title="Reload"
          onClick={() => dispatch({ type: "toast", message: "Reloaded (mock page — real webview integration point)" })}
        />

        <div className="mx-1 flex h-8 min-w-0 flex-1 items-center gap-2 rounded-full border border-line bg-cream-50 px-3.5 focus-within:border-iris-300 focus-within:bg-white">
          <Icon name="lock" size={12} className="shrink-0 text-mute" />
          <input
            value={draft}
            onChange={(e) => setDraft(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter") {
                const url = draft.trim();
                if (url) navigate(/^https?:\/\//.test(url) || url.startsWith("lumen://") ? url : `https://${url}`);
              }
              if (e.key === "Escape") setDraft(tab?.nav.currentUrl ?? "");
            }}
            className="w-full bg-transparent font-mono text-[12px] text-ink-700 outline-none placeholder:text-mute"
            placeholder="Enter a URL or search"
            spellCheck={false}
          />
        </div>

        <IconBtn
          name="star"
          title="Bookmark this page"
          onClick={() => dispatch({ type: "toast", message: "Page bookmarked (mock — persisted via SQLite in the Go backend later)" })}
        />

        <Menu
          width={210}
          button={(open) => (
            <IconBtn name="dotsV" title="Page actions" active={open} onClick={() => undefined} />
          )}
        >
          {(close) => (
            <>
              <MenuItem
                icon="copy"
                label="Copy URL"
                onClick={() => {
                  navigator.clipboard?.writeText(tab?.nav.currentUrl ?? "");
                  dispatch({ type: "toast", message: "URL copied" });
                  close();
                }}
              />
              <MenuItem
                icon="plus"
                label="Open in new tab"
                onClick={() => {
                  const newTabId = uid("tab");
                  dispatch({ type: "browser.tab.add", tabId: newTabId, sessionId: session.id });
                  dispatch({ type: "browser.navigate", tabId: newTabId, url: tab?.nav.currentUrl ?? "lumen://newtab" });
                  close();
                }}
              />
              <MenuItem
                icon="refresh"
                label="Reload"
                onClick={() => {
                  dispatch({ type: "toast", message: "Reloaded (mock page)" });
                  close();
                }}
              />
            </>
          )}
        </Menu>

        {!aiOpen && (
          <button
            type="button"
            onClick={() => dispatch({ type: "browser.ui", patch: { aiPanelOpen: true } })}
            className="ml-1 flex h-8 shrink-0 items-center gap-1.5 rounded-lg bg-iris-100 px-3 text-[12.5px] font-medium text-iris-700 transition-colors hover:bg-iris-200/70"
          >
            <Icon name="sparkles" size={14} />
            AI Browse
            <Badge tone="iris" className="ml-0.5">{state.browse.sourceIds.length}</Badge>
          </button>
        )}
      </div>

      {/* context strip for the current page */}
      {tab && !tab.nav.currentUrl.startsWith("lumen://") && (
        <div className="mt-2 flex items-center gap-2 text-[11.5px] text-mute">
          <Icon name="info" size={12} />
          {hostOf(tab.nav.currentUrl)} · mock page rendering — real web content arrives with the browser engine
        </div>
      )}
    </div>
  );
}
