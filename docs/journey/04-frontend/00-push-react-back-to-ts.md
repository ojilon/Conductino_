# 04-00 Push React back to TS: philosophy

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: React for shell only, TS/HTML/CSS for UI. Non-goal: removing React entirely day 1.
Where: `App.tsx:306 AppProvider`, `features/browser/*`, `features/reader/*`, `components/ui.tsx`.
Concept: components = functions of state; move logic to `utils/`, `state/selectors.ts`, plain TS helpers.
Rewrite options: keep React shell + replace panels one by one with TS-rendered HTML; measure dist size + RAM.
Exercise: pick `MockWebPage.tsx`, extract pure `renderArticle(article)->string` TS helper (no JSX), keep thin wrapper.
Done when: one panel's logic lives in TS, React only mounts it.
