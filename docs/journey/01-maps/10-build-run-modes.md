# 01-10 Build & run modes: pnpm dev vs wails dev

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: never confuse mock vs desktop. Non-goal: installer yet.
Where in code: `frontend/vite.config.ts`, `wails.json`, `frontend/main.go //go:embed all:dist`, `services/backend.ts:242 createBackend()`.
`pnpm dev` (frontend/): full UI, ALL mocked, no dialog. `wails dev` (root): real Go (library+filesystem+AI+SQLite).
Checks: `go vet/build ./...`, `tsc --noEmit`, `pnpm build`.
Exercise: run both, screenshot Settings dialog differences.
Done when: you can predict which mode each service uses.
