# 03-16 Selectors (derive, don't duplicate)

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: single source. Where: `state/selectors.ts`: activeBrowserSession/Tab, activeReaderTab/Document, pendingChangesFor, runningActivity, activeWorkspace, primarySummaryDocument.
Exercise: replace an inline `Object.values(...).find` with a selector (fixes tasks.md s2 hazard pattern).
Done when: no component scans flat maps directly.
