# docs/ — Master Index (rewritten for owner-led rewrite journey)

> You (the owner) are taking over from AI-generated code. This folder is now a
> **2-month, do-it-yourself rewrite guide**, not a description of frozen architecture.
> Every track teaches the Go / TypeScript concept first, then maps it to this repo's
> real files, then gives you a rewrite exercise where YOU change the code.

## Where to start

1. `journey/README.md` — the 8-week calendar, daily loop, and review gates.
2. `journey/00-meta/` — how to read these docs, how to learn Go+TS while rewriting.
3. `journey/01-maps/` — codebase maps: data flows, section transfers, file ownership.

## Tracks (each = folder with 10–20 small files, one issue per file)

| Track | Folder | Goal |
|---|---|---|
| 0. Meta + learning how to learn | `journey/00-meta/` | Doc conventions, 2-month plan, Go/TS study loop |
| 1. Codebase maps | `journey/01-maps/` | Repo tour, 7 layers, every data flow end-to-end |
| 2. Go foundations | `journey/02-go/` | Go concepts mapped to `backend/` files |
| 3. TS foundations | `journey/03-ts/` | TS/HTML/CSS mapped to `frontend/src/` files |
| 4. Frontend lightweight rewrite | `journey/04-frontend/` | Push React back to TS, own UI+canvas, push heavy work to Go |
| 5. Backend services | `journey/05-backend/` | filesystem, workspace, storage, documents, block IDs, cache |
| 6. AI tooling | `journey/06-ai/` | Providers, prompts, tools jail, thinking events, reflection |
| 7. Mirrors + intermediates | `journey/07-mirrors/` | Mirror concept, edit tracking, export, `.dev-data/` |
| 8. Releases + installer + updater | `journey/08-release/` | Versioning, CI, NSIS, portable zip, self-update loop |
| 9. API economy | `journey/09-apis/` | Using Gemini/Groq/OpenRouter well, budgets, failover, meters |
| 10. Heavy/non-network split + bridge | `journey/10-split/` | Separate heavy parsing/shell/tooling from network Go from day 1; goroutines in middle stage |
| 11. Rust (non-GC) | `journey/11-rust/` | Why Rust, safety, Go↔Rust bridge, what moves where |
| 12. Rewrite playbooks | `journey/12-playbooks/` | Checklists, exercises, review gates per track |

## Old docs (archived as history, superseded by journey)

- `archive/architecture.md`, `archive/state-model.md`, `archive/ai-integration.md`,
  `archive/document-rendering.md`, `archive/future-work.md` — snapshot of AI-built system.
  Read only after `journey/01-maps/`.
- `archive/plans/` — 00–12 plan files that drove the AI implementation. Historical.
- `archive/release-prep/` — 01–05 versioning/storage/installer notes. Absorbed into `journey/08-release/`.
- `tasks.md` (repo root) — bug inventory that motivated typed failures. Still referenced by file:line.

## Conventions used in every journey file

- Header: `Goal / Non-goal / Where in code / Go-or-TS concept / Rewrite exercise / Done when`.
- `file:line` links are verified against the tree at time of writing; if drifted, search by symbol name.
- No file tells you to "stay rigid to current format" — each ends with `Rewrite options` (keep / reshape / replace).
- Language primers never assume prior Go or TS: every new keyword is shown in a 5-line snippet from THIS repo.
