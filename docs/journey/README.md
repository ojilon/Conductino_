# Journey README — your 2-month owner-led rewrite

Goal: you redo, rethink, and correct the AI-built codebase until you can
restructure it yourself. Non-goal: freezing current architecture.

## Who this is for

Beginner in Go + TypeScript/HTML/CSS, taking over a Wails (Go) + React-TS app.
You want: lighter frontend (TS/HTML/CSS + own canvas, less React), heavy work
pushed to backend services, serious installer + self-update, disciplined API use,
and later a Rust sidecar for non-network heavy work.

## 8-week calendar

- Weeks 1–2 — `00-meta`, `01-maps`, `02-go`, `03-ts`. Read maps, run both modes
  (`pnpm dev` vs `wails dev`), learn Go structs/interfaces + TS types/unions by
  editing real files. No big rewrites yet.
- Weeks 3–4 — `04-frontend`, `05-backend`. Lighten frontend: shrink React to TS
  helpers, own canvas for PDF, move heavy ops behind `backend.ts` services.
  Harden filesystem/workspace/storage/documents.
- Weeks 5–6 — `06-ai`, `07-mirrors`, `09-apis`. Tool jail, thinking events,
  reflection, mirror concept, API budgets/failover. Introduce goroutines for
  concurrent extraction (see `10-split/`).
- Weeks 7–8 — `08-release`, `10-split`, `11-rust`, `12-playbooks`. Serious
  installer (NSIS drive choice + portable zip), self-update loop, heavy-vs-network
  split with bridge from day 1, Rust plan for shell/parsing/tooling.

## Daily loop (45–90 min)

1. Read ONE journey file (15 min). 2. Do its Rewrite exercise in code (30–60 min).
3. Run checks: `go vet ./...` + `go build ./...` (root), `pnpm exec tsc --noEmit`
   + `pnpm build` (frontend/). 4. Write 5-line log in `.dev-data/work/`.

## Review gates

- G1 (end wk2): draw data flows from memory; explain `Backend` bridge + reducer.
- G2 (end wk4): demo lighter UI + one service moved fully to Go with typed errors.
- G3 (end wk6): demo tool jail + mirror + API meters with budgets enforced.
- G4 (end wk8): installer + updater dry-run; split/bridge doc + Rust pilot (one parser).

## How files are organized

Each file: Goal / Non-goal / Where in code (file:line) / Concept (Go/TS mini-lesson)
 / How data flows / Rewrite options (keep/reshape/replace) / Exercise / Done when.
Groups of files = one combined issue (e.g. `04-frontend/02-*.md` = selection toolbar).
