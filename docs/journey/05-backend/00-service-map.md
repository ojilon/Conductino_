# 05-00 Backend map (remade: small packages, thin Backend)

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: see Backend as wiring. Where: `backend/main.go: Backend{lib, storage, ai, mirrors, workspaceID}`, `NewBackend`, `Init` restores last_library_root.
Concept (Go): packages own logic + tests; Backend only orchestrates (cache key, workspace id, event sink). No interfaces until a second implementation exists.
Exercise: draw call graph for ListLibraryTree, OpenFile, RunAI. Note which methods have NO Backend layer (read/list/save called directly).
Done when: you can add a reader arm (new ext) touching only `backend/read/`.
