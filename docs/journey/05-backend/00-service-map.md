# 05-00 Backend service map (Backend aggregates 5)

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: see Backend as bridge. Where: `backend/main.go:19 Backend{fs,storage,docs,ai,work}`, `NewBackend`, `Init` restores last_library_root.
Exercise: draw call graph for ListLibraryTree, OpenFile, RunAI, SaveDocument.
Done when: you can add a 6th service.
