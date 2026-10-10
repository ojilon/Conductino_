# 03-05 Slate adapter (blocks <-> editor, never source of truth)

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: keep blocks canonical. Where: `slateAdapter.ts toSlate/fromSlate`, `SummaryDocumentView.tsx`, tasks.md s3 steps 4-5.
Concept: block->Slate Element, segment->Text with marks (em/strong/highlightId transient).
Exercise: edit summary, save, verify blocks JSON round-trips.
Done when: you can swap editors in theory without migrating storage.
