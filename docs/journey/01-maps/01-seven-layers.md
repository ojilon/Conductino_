# 01-01 The seven layers and who owns what

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: enforce layer discipline. Non-goal: changing layers yet.
Layers: 1 UI (`features/**`, `components/**`) 2 app state (`state/appState.tsx` reducer) 3 domain types (`types/domain.ts`) 4 backend services (`services/backend.ts` + `backend/services/*.go`) 5 AI ops (`services/ai.ts` + `backend/services/ai/`) 6 rendering (`DocumentView.tsx`, `SummaryDocumentView.tsx`, `MockWebPage.tsx`) 7 persistence (`storage.go`, `sqlite_storage.go`).
Rule: layer 1 imports types + useApp only; service calls only at boundary files.
Exercise: pick `ReaderMode.tsx`, list its imports, label each with layer number.
Done when: you can label any import with its layer.
