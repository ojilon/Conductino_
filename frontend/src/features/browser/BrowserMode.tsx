/**
 * Browser mode layout:
 *   [workspace sidebar] [session + tabs + address bar + webview] [AI Browse panel]
 *
 * All open tabs stay mounted (hidden) so per-tab navigation state and
 * scroll position survive tab switches — the real webview integration
 * will rely on the same guarantee.
 */

import { useApp, activeBrowserSession } from "../../state/appState";
import { Icon } from "../../components/icons";
import { IconBtn, ResizablePanel, Button } from "../../components/ui";
import { uid } from "../../utils/helpers";
import WorkspaceSidebar from "./WorkspaceSidebar";
import BrowserChrome from "./BrowserChrome";
import MockWebPage from "./MockWebPage";
import AIBrowsePanel from "./AIBrowsePanel";

export default function BrowserMode() {
  const { state, dispatch } = useApp();
  const session = activeBrowserSession(state);
  const ui = state.browser.ui;

  return (
    <div className="flex h-full min-h-0">
      {/* left sidebar (collapsible) */}
      {ui.sidebarOpen ? (
        <WorkspaceSidebar />
      ) : (
        <div className="flex w-9 shrink-0 flex-col items-center border-r border-line bg-paper py-2">
          <IconBtn
            name="chevronsRight"
            title="Expand sidebar"
            onClick={() => dispatch({ type: "browser.ui", patch: { sidebarOpen: true } })}
          />
        </div>
      )}

      {/* center column */}
      <div className="min-w-0 flex-1 overflow-y-auto">
        <div className="mx-auto w-full max-w-[1180px] px-6 pb-10 pt-4">
          <BrowserChrome />

          <div className="overflow-hidden rounded-lg border border-line bg-white shadow-sm">
            {session.tabIds.length === 0 ? (
              <div className="flex flex-col items-center gap-3 px-6 py-16 text-center">
                <Icon name="globe" size={22} className="text-mute" />
                <p className="text-[13.5px] font-medium text-ink-700">No open pages in this session</p>
                <p className="text-[12px] text-mute">Open a page or run an AI Browse to fill the session.</p>
                <Button
                  variant="soft"
                  icon="plus"
                  onClick={() => dispatch({ type: "browser.tab.add", tabId: uid("tab"), sessionId: session.id })}
                >
                  New page
                </Button>
              </div>
            ) : (
              session.tabIds.map((tabId) => {
                const tab = state.browser.tabs[tabId];
                if (!tab) return null;
                const active = tabId === session.activeTabId;
                return (
                  <div key={tabId} className={active ? "block" : "hidden"}>
                    <MockWebPage
                      url={tab.nav.currentUrl}
                      onNavigate={(url) => dispatch({ type: "browser.navigate", tabId, url })}
                    />
                  </div>
                );
              })
            )}
          </div>

          <p className="mt-3 flex items-center gap-1.5 text-[11px] text-mute">
            <Icon name="info" size={11} />
            Browser remains usable while AI Browse works — sources can be opened, saved and sent to the Reader.
          </p>
        </div>
      </div>

      {/* AI Browse panel (collapsible + resizable) */}
      {ui.aiPanelOpen && (
        <ResizablePanel
          width={ui.aiPanelWidth}
          min={300}
          max={560}
          onWidth={(w) => dispatch({ type: "browser.ui", patch: { aiPanelWidth: w } })}
        >
          <AIBrowsePanel />
        </ResizablePanel>
      )}
    </div>
  );
}
