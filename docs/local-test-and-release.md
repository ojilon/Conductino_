# Local test & release guide (reusable)

How to test Conductino on your machine, bump the version, and ship a
GitHub pre-release. Background design lives in `docs/release-prep/`
(01 versioning, 02 storage layout, 03 build/installer, 05 checklist,
06 CI/releases, 07 tests).

## 1. Prerequisites

| Tool | Version | Notes |
|------|---------|-------|
| Go | 1.25+ | `go version` |
| Node | 24 | `node --version` |
| pnpm | 9+ | `pnpm --version` (repo uses `pnpm-lock.yaml`; never npm/yarn) |
| Wails CLI | v2.15.0 | `wails version`; install: `go install github.com/wailsapp/wails/v2/cmd/wails@v2.15.0` |
| NSIS | 3.x | Only for local `-nsis` installer builds. Verified: `winget install -e --id NSIS.NSIS` (then open a fresh terminal so `makensis` is on PATH). CI installs it for you — skip if you only test the exe. |

PowerShell note: if `pnpm`/`npx` `.ps1` shims are blocked by the
execution policy, call them via `cmd /c "pnpm ..."`.

No tag, no GitHub Release, and no API key is needed for local testing.
Unit tests are hermetic (model HTTP stays behind fakes).

## 2. Test locally (quick loop — do this before every push)

From the repo root — frontend first, then Go:

```bash
cmd /c "pnpm --dir frontend install --frozen-lockfile"  # first time / lockfile change
cmd /c "pnpm --dir frontend exec tsc --noEmit"           # or: pnpm --dir frontend typecheck
cmd /c "pnpm --dir frontend build"                       # catches bundle errors
```

```bash
go vet ./...
go test ./...
```

Order matters on a fresh clone: the frontend build must run **before**
any Go command, because `frontend/main.go` embeds `frontend/dist`
(`//go:embed all:dist`) and `dist/` is gitignored — `go vet`/`go test`
fail with `pattern all:dist: no matching files found` until the first
`pnpm build` creates it. (Once `dist/` exists, any order works.)

These are the exact commands `.github/workflows/ci.yml` runs on every
PR/push — if they pass locally, CI will pass.

## 3. Run the app locally

### Option A — dev mode (fast iteration)

```bash
wails dev
```

Hot-reloads the frontend; Go backend rebuilds on save.

### Option B — production binary (what users get)

```bash
cmd /c "pnpm --dir frontend build"
wails build
.\build\bin\Conductino.exe
```

For the installer shape instead (needs NSIS installed):

```bash
wails build -nsis
# → build/bin/Conductino-<arch>-installer.exe
```

### Data directory while testing

The app resolves its data root via `backend/apppaths`
(`CONDUCTINO_DATA` → `.dev-data` → portable `./data` → `%AppData%\Conductino`).
To keep test data off your real profile:

```powershell
$env:CONDUCTINO_DATA = "D:\ConductinoData"   # any drive works
.\build\bin\Conductino.exe
```

Or drop a `.dev-data/` folder at the repo root (a committed `README.md`
keeps it detected; everything else inside is gitignored). First run
creates `config.json`, `conductino.db`, `skills/`, `cache/extract/`,
`work/summaries/`.

### Smoke checklist (clean profile)

1. First run creates the data root (above).
2. Choose folder → library tree appears.
3. Open txt/md/docx/pdf; failures are typed (`reason`), never mock content.
4. One chat turn with a tool call; thinking/tool trace visible.
5. Summary mirror / edit path works.
6. Restart persists folder + core state.

## 4. Ship a release

### 4.1 Bump the version (two places, same value)

1. `backend/version/version.go` — `Version = "0.0.2"` (single source of truth, no `v` prefix).
2. `wails.json` — `info.productVersion` (stamps the Windows binary metadata).

Both must match; tags add the `v` (`Version "0.0.2"` → tag `v0.0.2`).

### 4.2 Write the release notes

1. Copy `docs/release-prep/release-notes/TEMPLATE.md` →
   `docs/release-prep/release-notes/vX.Y.Z.md` (filename must equal the tag).
2. Fill Added / Changed / Fixed + tester notes (installer, data dir, known limits).
3. Move `CHANGELOG.md` `## Unreleased` entries under a `## vX.Y.Z` heading.

Commit notes + changelog **before or in the same commit you tag** — CI reads
the notes file from the tagged commit. Missing file? CI falls back to the
`CHANGELOG.md` section, then to a stub (tag + SHA + compare link).

### 4.3 Tag and push

```bash
git tag -a v0.0.2 -m "v0.0.2"
git push origin v0.0.2
```

Any branch is fine — the release follows the tag, not the branch.
Never move or delete a published tag.

### 4.4 What CI does (`.github/workflows/release.yml`)

1. Checks out the tagged commit; re-runs the full test gate (fail → no Release).
2. Builds frontend + `wails build -nsis` on `windows-latest`.
3. Zips the exe as a portable build (`Conductino-<tag>-windows-amd64-portable.zip`).
4. Creates a GitHub Release named `<tag>` as a **pre-release** with the exe,
   installer, and zip attached, body from §4.2.

Download the pre-release assets and re-smoke once on a clean
machine/profile before promoting.

### 4.5 Promote to stable (manual, deliberate)

GitHub → Releases → edit → uncheck “Pre-release” (optionally set as latest).
Agents/CI never do this automatically.

## 5. Cheat sheet

```bash
# Full local gate (same as CI — frontend first, then Go; see §2 for why)
cmd /c "pnpm --dir frontend exec tsc --noEmit"
cmd /c "pnpm --dir frontend build"
go vet ./... ; go test ./...

# Dev / production run
wails dev
wails build ; .\build\bin\Conductino.exe

# Release
# 1. bump backend/version/version.go + wails.json info.productVersion
# 2. write docs/release-prep/release-notes/vX.Y.Z.md + CHANGELOG.md
git commit -a -m "release: vX.Y.Z"
git tag -a vX.Y.Z -m "vX.Y.Z" ; git push origin vX.Y.Z
# 3. wait for CI pre-release → smoke → promote manually
```

## 6. Troubleshooting

| Symptom | Cause / fix |
|---------|-------------|
| `wails build -nsis` warns `makensis not found` | Install NSIS locally, or rely on CI (it installs via choco). The plain exe still builds. |
| App wrote to `%AppData%` instead of `D:` | Set `CONDUCTINO_DATA` before launch, or pick the directory page in the installer (program dir ≠ data dir by design). |
| CI release has stub notes | The `vX.Y.Z.md` file wasn't on the tagged commit — commit it first, then tag. |
| SmartScreen warns on install | Expected: no code signing yet (out of scope for the first releases). |
| `pnpm.ps1 cannot be loaded` | PowerShell execution policy — use `cmd /c "pnpm ..."`. |
