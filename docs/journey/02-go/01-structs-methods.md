# 02-01 Structs + methods (Backend, Filesystem, Workspace)

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: see aggregation. Non-goal: interfaces yet.
Where: `backend/main.go:19 Backend{fs,storage,docs,ai,work}`, `filesystem.go:40 Filesystem{root}`, `workspace.go`.
Concept: struct = data; `func (f *Filesystem)` = method borrowing the struct; `NewX` = constructor convention.
Exercise: draw Backend's 5 fields and which service each delegates to.
Done when: you can add a new service field without help.
