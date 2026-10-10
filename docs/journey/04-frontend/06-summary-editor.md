# 04-06 Summary editor (Slate, gated DOCX autosave)

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: Slate stays a view, blocks stay truth. Where: `SummaryDocumentView.tsx`, `slateAdapter.ts`, `writeSummaryDOCX`.
Exercise: change highlight rendering without touching block shape.
Done when: round-trip toSlate/fromSlate covered by a test.
