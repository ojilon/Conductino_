# 04-17 Browser stack (Mode/Chrome/Sidebar/MockPage/AIBrowse)

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: know the placeholder boundary. Where: `BrowserMode.tsx`, `BrowserChrome.tsx`, `WorkspaceSidebar.tsx`, `MockWebPage.tsx` (2 articles + search + new-tab + fallback), `AIBrowsePanel.tsx`.
To integrate real webview: replace `<MockWebPage>` with webview mounting per `tab.nav.currentUrl`, report back via `browser.navigate`. Nothing else changes.
Exercise: add a third mock article + history entry.
Done when: navigation state fully in `tab.nav`.
