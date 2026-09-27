# 00 — Current priorities (on academic-harness tip)

**Status:** ACTIVE  
**Code base:** `feature/academic-harness-reader-ai` latest (`one messy push, seriously . . .`)  
**Audience:** Humans and coding agents

## Reality check (already landed — do not re-implement from scratch)

From plans 10–11 and the messy-push tree:

- **Mirror** — `backend/mirror/` summary `.md` working copies under `backend/.work/` (op log, re-sync on user save, preferred by `read_summary` / propose path).
- **Skills** — loader + starter skill(s), prompt injection, `App.MatchSkills`.
- **Workflows** — `backend/workflows/` + `run_workflow` tool (e.g. summarize-folder, add-to-summary paths).
- **Tools split** — `backend/tools/`, `backend/extract/`, `backend/ai/` (network only), `backend/usage/`.
- **Thinking trace** — tool trace on chat `done` payload + SQLite persistence for tool names/status.
- **In-file / canvas summary editing** — substantial UI path beyond older proposal-card-only design (see plan 11).

Agents must **read the existing packages and plans 10–11** before writing parallel systems.

## Ordered work from here

### Phase A — Release & durable local storage (do next)

Almost everything user-facing still needs a **stable, choosable data root** (install on D:, portable mode, skills/cache/SQLite outside Program Files).

→ Entire folder **`docs/release-prep/`**

1. Versioning + tags (`major.minor.patch`)  
2. App-data / `CONDUCTINO_DATA` layout (and how it relates to current `backend/.work/`)  
3. Wails build + NSIS **directory page** (user picks install drive) + portable zip  
4. Repo **`.dev-data/`** (or formalize `.work`) as pre-release debug mirror  
5. First-release smoke checklist  
6. **CI:** tests on PR; on `v*` tag → build assets → **default pre-release** ([06](../release-prep/06-ci-and-github-releases.md), [07](../release-prep/07-tests-local-and-ci.md))  
7. **Release notes folder** `docs/release-prep/release-notes/vX.Y.Z.md` for CI body  

### Phase B — Close gaps on harness (after A or in parallel only if no new data roots needed)

- Finish remaining items called out as **planned** in [11-skills-workflows-and-split.md](11-skills-workflows-and-split.md) (extra starter skills, workspace/user skill layers, propose-capable workflow ledger if not done).
- Extend intermediate/metadata story for **sources** if needed beyond summary mirrors (see release-prep storage + plan 10).
- Bidirectional “always fresh intermediate + recent edit meta in context” — largely started via mirror; tighten any remaining user-edit → next-turn gaps.
- Optional temporary Python helpers only behind `ToolHost` (not a second orchestrator).

### Explicitly FUTURE (do not expand scope without updating this file)

- Real browser engine (replace mock pages)
- Full git-like history on intermediates (standby op log is enough for now)
- Unrestricted host shell / arbitrary Python on host
- Cloud multi-user backend
- Non-GC rewrite of offline packages (keep boundaries clean only)
- OCR / scanned PDF pipeline
- In-app auto-updater from GitHub Releases (CI can publish assets first)

## Agent checklist

- [ ] Base work on **this** harness tip, not stale `main`.
- [ ] Prefer extending `mirror` / `skills` / `workflows` / `tools` over new parallel trees.
- [ ] Any new on-disk state: define root via release-prep path rules first.
- [ ] Tag releases are **pre-release** by default until a human promotes them.
- [ ] Small commits; cite `docs/plans/…` or `docs/release-prep/…`.
