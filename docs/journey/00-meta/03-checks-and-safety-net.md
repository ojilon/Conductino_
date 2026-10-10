# 03 Checks and safety net before you rewrite

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: always-green baseline. Non-goal: CI mastery yet (see 08-release).
Where in code: root `go vet ./...`, `go build ./...`; `frontend/` `pnpm exec tsc --noEmit`, `pnpm build`.
Concept (Go): `go vet` = static checks; `go build` = compile. (TS): `tsc --noEmit` = typecheck only.
Exercise: run all four now, paste output into your log. Fix nothing yet — just record baseline.
Done when: you can run the four checks from memory.
