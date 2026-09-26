# 05 — First release checklist

**Part of:** `docs/release-prep/`  
**See also:** [06-ci-and-github-releases.md](06-ci-and-github-releases.md), [07-tests-local-and-ci.md](07-tests-local-and-ci.md)

## Before tagging

### Paths & data

- [ ] Path resolver exists and is used (or clearly scheduled) for DB / skills / work mirrors.
- [ ] Dev still works with `.work` / `.dev-data` / `CONDUCTINO_DATA`.
- [ ] `CHANGELOG.md` ready; version has one source of truth.
- [ ] `docs/release-prep/release-notes/vX.Y.Z.md` written for this tag (or accept CI fallback).

### Tests (local — same commands CI will run)

- [ ] `go vet ./...` and `go test ./...` pass.
- [ ] `pnpm --dir frontend exec tsc --noEmit` and `pnpm --dir frontend build` pass.

### Build (local optional, CI on tag required)

- [ ] `wails build` succeeds for Windows (primary) when you care about installer shape.
- [ ] Installer offers **directory page** (D: allowed) once NSIS is configured.
- [ ] Portable zip path understood.

### Smoke (clean profile)

- [ ] First run creates data root.
- [ ] Open research folder → library tree.
- [ ] Open txt/md/docx/pdf; failures typed, not mock.
- [ ] Chat turn with tool + thinking/tool trace visible.
- [ ] Summary mirror / edit path works as on harness tip.
- [ ] Restart persists folder / core state as designed.

### Tag & CI

- [ ] Annotated `vX.Y.Z` on green commit; `git push origin vX.Y.Z`.
- [ ] CI creates a **pre-release** with built assets (not “latest stable” until you promote).
- [ ] Download assets from the pre-release and re-smoke once on a clean machine/profile.

## After tag

- Promote pre-release → full release only when intentional.  
- Return to [docs/plans/00-current-priorities.md](../plans/00-current-priorities.md) Phase B. Do not pull FUTURE items without updating priorities.
