# 01-03 Backend bridge: TS -> Wails -> Go and back

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: understand `window.go.frontend.App.*` boundary. Non-goal: writing new bindings yet.
Where in code: `frontend/app.go` (Bind), `backend/main.go:52 ListTree`, `:73 OpenFile`, `:189 RunAI`, `frontend/src/services/backend.ts:27 WailsApp`, `:242 createBackend()`.
Concept (Go): exported method = bound; `(T, error)` -> Promise<T> rejecting on error; nil pointer -> null.
Flow: TS `app.OpenFile(path)` -> Go `Backend.OpenFile` -> Resolve jail -> docs extract -> OpenedDocument value (reason on failure, never throw for classified errors).
Exercise: open a .txt in `wails dev`, watch Network/devtools + Go logs.
Done when: you can explain why classified failures return values not errors.
