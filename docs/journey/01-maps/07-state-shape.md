# 01-07 AppState shape: what persists vs ephemeral

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: separate domain vs UI state. Non-goal: SQLite schema yet.
Where in code: `types/domain.ts:473 AppState`, `docs/state-model.md`, `backend/services/storage.go`, `sqlite_storage.go`.
Domain (persistable): sources, documents, changes, chat threads, workspace byId. Ephemeral: panel widths in ui, selection, toast, previewSourceId.
Exercise: list 5 fields that must survive restart vs 5 that must not.
Done when: you can justify each placement.
