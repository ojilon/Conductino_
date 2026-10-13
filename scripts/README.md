# scripts/ — what each bat does

Run everything from the repo root. Order for a normal day:

```
scripts\dev.bat              # code + test in the app
scripts\test.bat             # checks before release
scripts\build.bat            # compile exe + installer
scripts\release.bat v0.0.3   # local packaged release (test installer)
scripts\push_release.bat v0.0.3  # tag + GitHub pre-release
```

## dev.bat — everyday debugging (use this, not `wails dev` directly)

```
scripts\dev.bat
```

1. Creates `.dev-data\temporary\extract_dump\` if missing.
2. Sets `CONDUCTINO_DUMP_EXTRACT=1` for that window only — every file you
   open dumps its extraction JSON there so you can inspect it.
3. Runs `wails dev` (blocks until you close the app).
4. On close, deletes `.dev-data\temporary\extract_dump\*.json`.
   Cache, db, skills, `work/` are kept.

## test.bat — fast checks

```
scripts\test.bat
```

`go vet ./...` + `go test ./...` + frontend `tsc --noEmit` (skipped if
pnpm missing). Run before every release.

## build.bat — compile locally (no tag, no GitHub)

```
scripts\build.bat
```

Needs Go, Node+pnpm, Wails CLI. Builds frontend, `Conductino.exe`, and
`Conductino-Installer.exe` into `build\bin\` (gitignored).

## release.bat — local packaged release

```
scripts\release.bat v0.0.3
```

Bumps `backend/version/version.go` + `wails.json`, stubs
`docs\release-prep\release-notes\v0.0.3.md` from TEMPLATE if missing,
calls `build.bat`, packs
`installer\dist\Conductino-v0.0.3-windows-amd64\` (bin exes + README)
and zips it. Nothing touches git or GitHub — test the installer from
that folder first.

## push_release.bat — tag + GitHub pre-release

```
scripts\push_release.bat v0.0.3
```

Needs `gh` CLI + clean git. Creates annotated tag if missing, pushes it,
then `gh release create --prerelease` with the notes file and the zip
from `release.bat`. Promote to stable manually on GitHub when ready.

## Rule for new code

Never hardcode `.dev-data` or `Conductino_Data`. Resolve via
`backend/apppaths` (`DataRoot()`, `ExtractDumpDir()`, ...) and create
folders with `MkdirAll` on demand. Then `dev.bat` gives `.dev-data`
behavior and the installed exe gives `Conductino_Data` behavior — same
code, both work.
