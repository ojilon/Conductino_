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
- **Landed:** prompt wiring — `buildPromptFor` injects ≤2 matched excerpts
  (~1500 chars) for every operation; no skills loaded = byte-identical
  prompts (tested). `App.MatchSkills` exposes matching as JSON for UI/debug.
- **Landed:** intent-phrase routing — `Match` fires when a when-phrase
  appears inside the user text ("add to summary"), not just on operation
  names; skill v2 teaches the autonomous fast path (one `run_workflow`
  call, then report — no narration, no asking first).
- **Planned:** starters `revise-proposal`, `find-in-workspace`,
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

**Landed v1:** `backend/workflows/` (`Runner`, `summarize-folder` digest
routine, budget caps, tests) + `run_workflow` tool (catalog + native
function schema + compat test). v1 routines are read-only (inherently
idempotent); the (thread, step) ledger lands with propose-capable routines.

**Landed v2:** `add-to-summary` — the deterministic read→propose loop in
one tool call (verify summary → read source with fuzzy resolution → up to
3 cited inserts). Multi-proposal surfacing: `ToolHost` accumulates EVERY
`propose_summary_edit` per turn and the chat `done` payload carries all of
them, so multi-claim turns never drop earlier edits (only the last used to
survive). The Review gate stays the confirmation — a same-turn retry shows
as visible duplicates, never silent double-writes.

## 3. Summary edit loop (planned — Phase 17, needs 10 §2 mirrors)

```
.docx summary ──extract──► .md mirror (backend/.work/)
      ▲                          │ read_summary / propose_summary_edit
      │ publish_summary          ▼ (tool: mirror → blocks → .docx, jailed to mapped path)
      └── blocks ──◄── tracked edits ──► in-file diffs (reload + decorate)
              ▲                          │ accept dismisses · reject restores
 accept/revise (+ custom instruction + full summary + marked span) ──► AI
```

**Landed:** revise-with-context — `runRevise` sends `focusedChange` +
summary snapshot; Go `OpRewrite` renders the focused span via the shared
`focusedProposalBlock` helper (also used by chat). Diff-targeted revise
with a custom instruction lives in the diff popover (span-anchored chat).

**Landed:** `publish_summary` — summaries are openly editable by default
(sources stay read-only). Writes the mirror through to the mapped `.docx`
only (frontend-supplied path, Resolve-jailed); records unpublished mirror
ops as diffs; the chat payload tells the UI to reload + decorate in-file.
`extract.BlocksFromMarkdown` shapes mirror text back to wire blocks with
stable IDs, so anchors survive publish round-trips. A turn that proposes
but never publishes writes through automatically (single-path guarantee,
traced + audited). Proposed-but-unwritten turns carry `publishError`
(no summary / no path / publish failure text) so the UI toasts the exact
fix instead of silence.

**Landed:** in-file diff review replaces proposal cards (removed: Review
face, `runRevise`, `change.revise`, Slate editor + adapter). `DocxCanvas`
edits the `.docx` directly — paginated, `plaintext-only` blocks (rich paste
lands as plain text), blur-commit, Enter-split, per-item lists, blank-line
runs normalized into real blocks on commit. Diffs highlight by color
(insert green, modify amber, deletes as cards); right-click a diff for
Accept / Reject / instruction + Send to AI (native webview menu suppressed
on pages). Save always writes the open file itself — never a `summaries/`
copy. Reject restores the pre-image (autosave writes the `.docx`, mirror
re-syncs). Toolbar "Include in summary" lands accepted at birth — no cards
anywhere.

- Frontend shows exactly where the AI edited (pending modify/insert/delete
  treatments already exist); revise sends custom instruction + the span.
- Mirror is rewritten before each AI turn so the model always edits fresh
  text. Accept optionally notifies the AI (a `chat.append` system note —
  cheap, keeps thread coherent across turns).

## 4. Frontend simplification + backend long-running APIs (planned)

**Landed — chat thinking trace (the "what did it do" view):** every tool
dispatch is timed in the chat loop; the `done` payload carries
`toolTrace: [{tool, ok, ms}]`; assistant messages render it as an
expandable "Thinking · N calls · Xms" block. Trace persists in SQLite
(`chat_messages.tool_trace`, additive migration for old DBs) so reloads
keep the log. Names + status only — never content (same rule as audit).

Goal: UI owns rendering + intent only; anything background/long-running
moves behind promise APIs (already the `backend.*` pattern).

| Candidate | Today | Target API |
|---|---|---|
| Context-pack assembly | ~~frontend builds `ContextPack` string~~ → `buildContextPackAsync` (bridge-first, local fallback) | `App.BuildContextPack` (**landed** + Go parity test) |
| Chat history load | ~~in-memory only~~ → persisted + loaded on folder switch | `App.LoadThread/ListThreads/ListMessages` (**landed**; were bound but uncalled) |
| DOCX save | via `writeSummaryDOCX` | keep; mirror re-syncs on mtime (**landed**) |
| Skill matching | n/a | `App.MatchSkills` (**landed**; UI pending) |
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
