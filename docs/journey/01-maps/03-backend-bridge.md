# 01-03 Backend bridge: TS -> Wails -> Go and back

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: understand `window.go.frontend.App.*` boundary. Non-goal: writing new bindings yet.
Where in code: `frontend/app.go` (Bind — exactly what `backend.ts` calls, nothing more), `backend/main.go` OpenFile + ReadDumpFile, `frontend/src/services/backend.ts` WailsApp interface.
Concept (Go): exported method = bound; `(T, error)` -> Promise<T> rejecting on error; nil pointer -> null; classified failures return VALUES with Reason, never throws.
Flow: TS `app.OpenFile(path)` -> Go `Backend.OpenFile` -> Resolve jail -> cache? hit : extract -> dump file -> Doc{title, kind, dump} (inline JSON cleared) -> TS `app.ReadDumpFile(dump)` -> parse -> render.
Exercise: open a .txt in `wails dev`, watch the dump land in `.dev-data/temporary/extract_dump/`, then watch the UI read it back.
Done when: you can explain why classified failures return values not errors, and why content travels via dump files.
