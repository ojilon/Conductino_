# Operations & growth — skills, workflows, chat observability, tests, releases

> Status: **proposal** (08) — future stuff. Nothing here is implemented;
> items graduate into `04-implementation-phases.md` as phases when started.
> Predecessors: 05 (multi-provider), 06 (resplit), 07 (source reading).

## 1. SKILL.md files (instruction layer above tools)

Tools are code; **skills are prose**. A skill is a versioned Markdown file
that teaches the harness (and future workflows, §2) *how* to use tools,
perform tasks, and run providers in parallel — robust instructions the
prompt assembler injects only when relevant, instead of one ever-growing
system prompt.

### 1.1 Format (one file per skill)

```md
---
skill: summarize-source
version: 1
when: ["user asks to summarize", "AI_MERGE", "AI_CHAT + propose_summary_edit"]
tools: [list_workspace, read_source, read_summary, propose_summary_edit]
budgets: { maxToolCalls: 6, maxChars: 6000 }
---
# Summarize a source into the living summary
1. read_summary first (harmonize, never append blindly).
2. Read the source in windows (07); never summarize an unread window.
3. One propose_summary_edit per claim; modify/delete allowed, not just insert.
4. If a definition on page N is superseded on page M, redefine (cite both).
```

Frontmatter = machine routing (`when` matches operation/intent,
`tools`/`budgets` constrain the loop); body = steps the model follows.
Keep each skill to one screen — long skills get skipped by every model.

### 1.2 Discovery + layering

| Layer | Location | Ownership |
|---|---|---|
| Bundled skills | `backend/skills/*.md` (embedded) | ships with the app, versioned in git |
| Workspace skills | `<root>/.conductino/skills/*.md` | per-folder craft (yours), wins on name clash |
| User skills | app-data `Conductino/skills/*.md` | cross-folder personal routines |

At prompt build, the harness attaches **at most 2 skill excerpts**
(matched by `when`, capped ~1500 chars total). Skills never contain keys,
paths, or file content — instructions only.

### 1.3 Starter set (proposal)

* `summarize-source` (above), `revise-proposal` (focusedChange etiquette),
  `find-in-workspace` (search → read windows → cite `path:page`),
  `parallel-map` (§3: split sources across providers, merge on primary),
  `quota-aware` (429 → shrink windows, switch provider, say so honestly).

## 2. Workflows (named multi-step routines, future)

A workflow is a skill with a **runner**: an ordered, gated routine the chat
loop (or a button) can invoke, e.g. `summarize-folder`: list → read each
source in windows → draft per-source proposals → user gates each →
merge pass. Lives in `tools/` post-06 (runner + checkpoints), defined in
YAML-or-Markdown beside skills. Gates are mandatory: no workflow writes or
proposes past a user checkpoint. Workflows reuse the tool budget/audit —
they are composition, not a second tool system.

## 3. Parallel APIs (grows out of 05)

Skills declare parallelizable steps (`parallel-map`); the router gains:

* per-task provider assignment (S-class fan-out to secondary, merge on
  primary), bounded fan-out (default 3, semaphore-shared with §policy),
* idempotency per (thread, step) so retries never double-propose,
* result merging rule: extracts concatenate with `path:page` cites;
  conflicts surface as two proposals, never silent overwrite.
* `UsageEvent` gains `threadId` + `step` so meters attribute map-reduce.

No change to the `AIProvider`/event contract: parallelism is a backend
scheduling detail the UI sees only as phase labels.

## 4. Chat interface upgrades (observability per turn)

* **Token usage**: per-turn `in/out` chips (exact counts when the provider
  returns usage, else the chars/4 estimate, labeled `~`), thread totals in
  the thread header. Source: persisted `UsageEvent`s (§5), not ad-hoc math.
* **Model names**: every assistant message records `via <provider>/<model>`
  (already in failover paths conceptually — persist on the message so
  "who answered this?" survives restarts and provider switches mid-thread).
* **Tool-call log**: collapsible per-turn trace (tool, redacted arg, ok/err,
  ms) from the persisted audit — the user-visible twin of today's
  in-memory ring. "Why did it read that file?" becomes answerable.
* **Retry / switch-provider action** on error bubbles (re-run turn on
  secondary explicitly, not just auto-failover).

## 5. Logs database + AI self-guidance

New tables (additive, same SQLite DB): `ai_log` (turns: thread, op, class,
provider, model, tokens, latency, status, error class), `tool_audit`
(persist today's ring: tool, redacted arg, ok, ms), `guidance_notes`
(deduped failure signatures → advice text).

* Retention: 90-day rolling delete (nightly on startup, one DELETE).
* Viewer: Settings → "Logs" tab (filter by thread/status, export JSONL).
* Self-guidance loop: when an error class repeats (e.g. groq 429 ×3 in
  10 min), the harness prepends a guidance block to the next prompt
  ("Groq is rate-limited — prefer smaller windows or ask the user to
  switch primary; do not retry the same call verbatim"). Guidance notes
  are data (editable, deletable), never hardcoded nag text.
* Privacy: prompts/contents never logged — ids, names, counts, error
  classes only (same rule as today's audit ring).

## 6. Tests — CI/CD + local learning tests

* **CI** (new `.github/workflows/ci.yml`): `go vet ./...`, `go test
  ./backend/...`, `pnpm --dir frontend exec tsc --noEmit`, `pnpm
  --dir frontend build`. No secrets needed (all model tests are hermetic;
  network paths stay behind interfaces with fakes).
* **Frontend tests** (new): add `vitest` + `pnpm test` for pure logic
  (`mentions.ts`, `contextPack.ts`, `slateAdapter.ts`) — the three files
  most likely to regress under UI churn.
* **Learning tests** (for you): keep a `*_test.go` + `*.test.ts` style of
  small table tests with plain-English names (`TestReadSourceTypoSuggests`,
  `resolveMentions-accent folding`, …). Reading order if new to Go tests:
  `tools/tools_test.go` (table + fakes) → `extract/ids_test.go`
  (property-style: stability) → `usage/policy_test.go` (cache/semaphore).
  Every new behavior ships with one such test — that *is* the CI contract.

## 7. Releases — tags, packaging, install location

* **Tags**: `v0.x.y` annotated tags + short changelog (keep a
  `CHANGELOG.md`, "Unreleased" section at top). Tag only green `main`.
* **Packaging**: `wails build` per OS (windows/amd64 first). Artifacts:
  installer + portable `.zip` of the same binary (zip = escape hatch when
  installers misbehave on locked-down machines).
* **Install location** (your D: vs C: ask): Wails uses NSIS on Windows —
  add an `nsis:` block in `wails.json` with a directory page
  (`MUI_PAGE_DIRECTORY`) so setup asks where to install, defaulting to
  `%ProgramFiles%`. Verify the uninstaller removes the choice cleanly;
  keep the portable zip path-agnostic (DB under `%APPDATA%`, overridable
  via `CONDUCTINO_DB`, so drive choice never strands user data).
* **Pre-release checklist**: bump version, changelog entry, `pnpm build`
  artifacts fresh in `dist/`, smoke test (open folder → open pdf/docx →
  one chat turn with a tool call) on a clean machine profile.
