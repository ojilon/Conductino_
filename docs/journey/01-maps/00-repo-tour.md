# 01-00 Repo tour: what lives where

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: name every top-level entry. Non-goal: understanding each yet.
Where in code: `main.go` (thin router), `wails.json`, `frontend/app.go` + `frontend/main.go` (Wails shell), `backend/main.go` (pure-Go Backend), `backend/services/`, `backend/models/models.go`, `frontend/src/` (App.tsx, state/, services/, features/, components/), `docs/`, `tasks.md`, `build/`.
Concept (Go modules): `go.mod` module `Conductino`; import path `Conductino/backend/models`.
Exercise: open each top file, write one sentence per file in your log.
Done when: you can draw the tree from memory.
