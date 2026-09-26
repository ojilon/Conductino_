# 07 — Tests: local and CI (agent-friendly)

**Status:** ACTIVE — design to implement  
**Part of:** `docs/release-prep/`  
**Related:** [06-ci-and-github-releases.md](06-ci-and-github-releases.md)

## Goal

- Same **test commands** run on a developer machine and in GitHub Actions.  
- CI **blocks** tag releases if tests fail.  
- Agents know **where** to put new tests and **how** to run a subset locally.

## Layers of tests

| Layer | Location (typical) | Local command | CI |
|-------|-------------------|---------------|-----|
| Go unit / package tests | `backend/**/*_test.go` | `go test ./...` from repo root (or `./backend/...` if modules require) | Required on PR + tag |
| Go vet | whole module | `go vet ./...` | Required |
| Frontend typecheck | `frontend/` | `pnpm --dir frontend exec tsc --noEmit` | Required |
| Frontend unit (when present) | `frontend/**/*.test.ts` | `pnpm --dir frontend test` | Required once Vitest (or similar) is added |
| Frontend production build | `frontend/` | `pnpm --dir frontend build` | Required on tag job (and optional on PR) |
| Wails build | root | `wails build` | Tag release job only (slow) |
| Manual smoke | — | Install/run exe; open folder; one tool chat | Human; checklist [05](05-first-release-checklist.md) |

Hermetic rule: **no live API keys** in CI unit tests. Model HTTP stays behind fakes/interfaces (existing pattern in `backend/ai` tests).

## Local quick loop (do this before push)

```bash
# From repo root — adjust if go.mod lives under backend/
cd frontend
pnpm install          # first time / lockfile change
pnpm exec tsc --noEmit
pnpm build            # catches bundle errors; ALSO creates frontend/dist,
                      # which frontend/main.go embeds (//go:embed all:dist).
                      # On a fresh clone, this build must run BEFORE any Go
                      # command — go vet/test fail with
                      # "pattern all:dist: no matching files found" until
                      # dist/ exists (it is gitignored).
# pnpm test           # when frontend tests exist
cd ..

go vet ./...
go test ./...
```

Optional full desktop binary (not required every commit):

```bash
wails build
# run the exe under build/bin/ — offline, no GitHub Release needed
```

## What CI should run on every PR / push

Minimum bar (implement in `.github/workflows/ci.yml`):

1. `go vet ./...`  
2. `go test ./...`  
3. `pnpm --dir frontend install --frozen-lockfile` (or project-equivalent)  
4. `pnpm --dir frontend exec tsc --noEmit`  
5. `pnpm --dir frontend build`  

Optional later: cache Go modules and pnpm store.

## What CI should run only on tag (release workflow)

Everything above, **plus**:

6. Install Wails CLI (pinned version)  
7. `wails build` on `windows-latest`  
8. Upload artefacts + create **pre-release** (see 06)

If step 1–5 fail, **do not** create a Release.

## Where agents add new tests

| New behaviour | Prefer test file |
|---------------|------------------|
| Tool jail / path resolve | `backend/tools/*_test.go` |
| Mirror / op log | `backend/mirror/*_test.go` |
| Skills parse / match | `backend/skills/*_test.go` |
| Workflows | `backend/workflows/*_test.go` |
| Extract empty / normalize | `backend/extract/*_test.go` |
| Provider failover (fake HTTP) | `backend/ai/*_test.go` |
| Pure TS helpers (mentions, adapters) | `frontend` Vitest once enabled |

**Agent rule:** every new non-trivial behaviour ships with at least one table-driven or example test that runs offline.

## Test cases humans should keep runnable locally

Document concrete cases in PRs; examples to always protect:

1. **Empty / zero-byte** supported file opens as blank, not parse_error.  
2. **Path jail** rejects `..` and absolute escapes in tools.  
3. **Did-you-mean** near-miss source path suggests a real tree path.  
4. **Skill frontmatter** invalid shape fails loudly (no silent skip).  
5. **Mirror** re-sync when user snapshot hash changes.  
6. **No network** in `extract` / `tools` / `skills` / `mirror` dependency checks if you already enforce them.

Manual smoke (not automated in first CI):

- Choose folder → open pdf/docx → one chat turn with a tool → thinking/trace visible.

## Providing “test cases” as project artefacts

Optional later folder (only if it helps agents):

```
docs/release-prep/test-cases/
  README.md           # index of scenarios
  smoke-desktop.md    # manual smoke steps
```

Do **not** duplicate Go tests as markdown. Markdown is for **manual** and **product** scenarios; code tests stay in `*_test.go`.

## Agent checklist when touching CI or tests

- [ ] Same commands documented here work locally.  
- [ ] Tag workflow keeps `prerelease: true` unless docs explicitly change.  
- [ ] No secrets required for unit tests.  
- [ ] New package → at least one `*_test.go` or an explicit deferral note in the PR.  
- [ ] Do not upload binaries from the PR workflow.  
