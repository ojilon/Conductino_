# Skills, workflow runner, summary edit loop, frontend/backend split

> Status: **proposal + loader landed**. Born from a real session: the model
> showed tool calls instead of completing the edit, claimed success without
> a tool result, and had no verify/retry discipline. Shallow prompt strings
> in `tools/` cannot carry that discipline — versioned skill files can.

## 1. Skills (loader + starter landed, wiring planned)

- **Format:** one `.md` per skill, machine frontmatter (`skill, version,
  when, tools, budgets`) + prose steps. Parser is stdlib-only and total:
  unknown shapes fail loudly, never silently (`skills.go`).
- **Discovery layers:** bundled `backend/skills/*.md` < workspace
  `<root>/.conductino/skills/*.md` < user app-data skills. At most 2
  excerpts (~1500 chars) injected per prompt (`Match` + `Excerpt`).
- **Landed:** loader + `summarize-source` starter (contains the
  verify/retry/report rules the failed session was missing).
- **Planned:** prompt wiring at prompt build (`prompts.go`/`context.go`
  attach matched excerpts); starters `revise-proposal`, `find-in-workspace`,
  `quota-aware`; workspace/user layers with name-clash precedence.
- **Rule:** skills are data + loader. No network, no storage I/O, no
  third-party deps — same leaf discipline as `extract`/`usage`.

## 2. Tools vs skills vs workflows (placement decision)

`tools/` and `skills/` stay **sibling packages, not one module**: tools =
code verbs with sandbox guards; skills = instruction data + loader. The
runner that composes them is a third sibling, `backend/workflows/`:

```
backend/workflows/   # NEW (planned) — gated routines: ordered steps,
                     # checkpoints, idempotent (thread, step) retries.
                     # Reuses tools.ToolHost budget + audit; never a
                     # second tool system. Imports tools + skills only.
```

Workflows are skills with a runner (e.g. `summarize-folder`: list → read
windows → draft proposals → user gates each → merge pass). Gates are
mandatory: no workflow proposes past a user checkpoint.

## 3. Summary edit loop (planned — Phase 17, needs 10 §2 mirrors)

```
.docx summary ──extract──► .md mirror (backend/.work/)
      ▲                          │ read_summary / propose_summary_edit
      │ accept path              ▼
      └── blocks ──◄── tracked edits (pending DocumentChange, Review UI)
              ▲
 accept/revise (+ custom instruction + full summary + marked span) ──► AI
              │ produces new mirror content ──► render ──► review again
```

- Frontend shows exactly where the AI edited (pending modify/insert/delete
  treatments already exist); revise sends custom instruction + the span.
- Mirror is rewritten before each AI turn so the model always edits fresh
  text. Accept optionally notifies the AI (a `chat.append` system note —
  cheap, keeps thread coherent across turns).

## 4. Frontend simplification + backend long-running APIs (planned)

Goal: UI owns rendering + intent only; anything background/long-running
moves behind promise APIs (already the `backend.*` pattern).

| Candidate | Today | Target API |
|---|---|---|
| Context-pack assembly | frontend builds `ContextPack` string | `App.BuildContext(docIds, selection)` (Go owns budget) |
| Chat history load | in-memory only | `App.LoadThread` (exists) on workspace switch |
| DOCX save | via `writeSummaryDOCX` | keep; add mirror-aware save |
| Skill matching | n/a | `App.MatchSkills(op, intent)` (Go, embedded skills) |
| Usage aggregation | `AIMeters` string | keep; add structured meters later |

Nothing here changes the Wails contract shape (JSON in/out); the React
reducer stays the mutation funnel.

## 5. Non-GC readiness (standing rule, extended)

New constraint from this round: keep non-network submodules rewritable in
a non-GC language later. Rules, enforced at review:

- `extract`, `tools`, `skills`, `usage`, `models`: stdlib + a fixed allowlist
  (`modernc.org/sqlite`, `ledongthuc/pdf`) — no new third-party deps without
  updating this list.
- Boundaries are interfaces over JSON/blocks (`Extractor`, `ToolHost` narrow
  ifaces, `StorageService`) — no Go pointers or goroutine lifetimes leak
  across them. (Matches the 06 cgo rule: only C ABI + JSON/blocks cross.)
- Network (model HTTP) stays inside `backend/ai` only; offline packages must
  keep passing the `go list -deps` check (no `net/http`, no Wails).
