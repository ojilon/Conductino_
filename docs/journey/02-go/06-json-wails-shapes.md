# 02-06 JSON + Wails shapes (models.go <-> domain.ts)

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: keep Go/TS mirrors in sync. Non-goal: codegen.
Where: `backend/models/models.go` (FileTreeNode, OpenedDocument, AIRequest, AIEvent) <-> `frontend/src/types/domain.ts`, `services/backend.ts:56 OpenedFile`.
Exercise: add a field to OpenedDocument, mirror in TS, verify desktop open still works.
Done when: you can do a mirrored change without breaking bindings.
