# 06 — CI and GitHub Releases (agent-friendly)

**Status:** ACTIVE — design to implement  
**Part of:** `docs/release-prep/`  
**Audience:** Humans and coding agents adding `.github/workflows/` and release metadata

## Goal

When a **git tag** is pushed (e.g. `v0.0.1`):

1. CI runs **tests** (fail → no release assets).
2. CI **builds** Windows desktop artefacts (exe / installer / portable zip as configured).
3. CI opens a **GitHub Release** and attaches those artefacts as **assets**.
4. For safety, the default is a **pre-release** (not the latest stable). Promoting to a full release is a deliberate second step.

Local day-to-day testing does **not** require CI or a Release (see [03-build-and-installer.md](03-build-and-installer.md) and [07-tests-local-and-ci.md](07-tests-local-and-ci.md)).

## What lives in the repo vs what CI produces

| Item | In git? | Who creates it |
|------|---------|----------------|
| Source, `wails.json`, `build/windows/*` templates | Yes | Developers |
| **Release notes inputs** under `docs/release-prep/release-notes/` (or agreed path) | Yes | Humans / agents before tagging |
| Workflow YAML under `.github/workflows/` | Yes | Agents implement once |
| Built `.exe`, installer, zip | **No** (gitignore) | **CI on tag** (or local `wails build`) |
| GitHub Release + downloadable assets | On GitHub only | CI (`softprops/action-gh-release` or equivalent) |

**Never commit** large binaries into the repo. CI uploads them only as Release assets.

## Folder CI should read (release metadata)

Keep a **fixed, small folder** so agents and CI always know where to look:

```
docs/release-prep/release-notes/
  README.md                 # how to write notes (committed)
  TEMPLATE.md               # copy for a new version
  v0.0.1.md                 # optional per-tag body (preferred when present)
  # … one file per published tag when you want custom notes
CHANGELOG.md                # repo root — Unreleased + version history (also used)
```

### Resolution order for Release body (agent rule)

1. If `docs/release-prep/release-notes/<tag>.md` exists (e.g. `v0.0.1.md` for tag `v0.0.1`) → use that file as the Release description.  
2. Else extract the matching section from root `CHANGELOG.md` if present.  
3. Else use a short generated stub: tag name + commit SHA + link to compare.

Do **not** invent long release notes in the workflow. Prefer files humans/agents wrote before `git tag`.

## Triggers (recommended two workflows or two jobs)

### A. Continuous test (every PR / push to active branches)

- **On:** `pull_request`, `push` to `main`, `feature/**` (or the branches you care about).
- **Does:** unit/build checks only (see [07-tests-local-and-ci.md](07-tests-local-and-ci.md)).
- **Does not:** create Releases or upload installers (saves minutes and avoids junk releases).

### B. Tag → build → pre-release

- **On:** `push` of tags matching `v*` (e.g. `v0.0.1`, `v0.1.0`).
- **Does:**
  1. Checkout the **tagged commit** (any branch is fine — tag points at the commit).
  2. Run the same test suite as (A); **abort on failure**.
  3. Install Go, Node/pnpm, Wails CLI (pin versions in the workflow).
  4. `pnpm --dir frontend install` + `pnpm --dir frontend build`.
  5. `wails build` (Windows runner for the primary Windows artefact).
  6. Collect outputs from `build/bin/` (exact names depend on Wails; document in the workflow comments once known).
  7. Create GitHub Release:
     - **name:** same as tag  
     - **prerelease:** `true` by default  
     - **body:** from release-notes folder (order above)  
     - **files:** built assets  
  8. Optional: also upload a portable `.zip` of the binary.

### Promoting pre-release → full release

- Manual on GitHub: edit Release → uncheck “Pre-release”, or  
- Optional later workflow `workflow_dispatch` with input `tag` + `make_latest=true` (only after human confirmation).

Agents must **not** auto-publish `latest` stable without an explicit doc change or manual step.

## Tags and branches

- Tag **any** commit (harness tip, release branch, `main`). Release follows the tag, not the branch name.
- Annotated tags preferred: `git tag -a v0.0.1 -m "v0.0.1"` then `git push origin v0.0.1`.
- Versioning stays **major.minor.patch** as in [01-versioning-and-tags.md](01-versioning-and-tags.md). Pre-1.0 tags are normal.

## Permissions

Workflow needs:

```yaml
permissions:
  contents: write   # create release + upload assets
```

Use `GITHUB_TOKEN` (default). No need to store the app binary in secrets.

## Safety defaults (non-negotiable for agents)

1. **Default prerelease: true** on tag-driven releases.  
2. **Tests must pass** before upload.  
3. **No release on every commit** — only on `v*` tags (plus optional `workflow_dispatch` for dry-runs).  
4. **Notes from repo files**, not from model hallucination in CI logs.  
5. Pin tool versions (Go, Node, Wails) in the YAML so builds are reproducible.

## Implementation sketch (for the agent that adds the workflow)

Create e.g.:

- `.github/workflows/ci.yml` — test on PR/push  
- `.github/workflows/release.yml` — on `tags: ['v*']` → test + build + softprops/action-gh-release with `prerelease: true`

Windows artefact job should use `runs-on: windows-latest` for the real `.exe` / NSIS path. Linux can still run Go tests if desired.

Until NSIS directory page is configured, still upload whatever `wails build` produces; improve packaging later without changing the tag trigger.

## Out of scope (for the first CI)

- Auto-updater inside the app that downloads from GitHub  
- Code signing / SmartScreen (document later)  
- Multi-OS matrix beyond Windows primary  
- Publishing to Winget / Microsoft Store  
