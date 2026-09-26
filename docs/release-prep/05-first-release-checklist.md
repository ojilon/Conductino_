# 05 — First release checklist

**Part of:** `docs/release-prep/`

## Before tagging

### Paths & data

- [ ] Path resolver exists and is used (or clearly scheduled) for DB / skills / work mirrors.
- [ ] Dev still works with `.work` / `.dev-data` / `CONDUCTINO_DATA`.
- [ ] `CHANGELOG.md` ready; version has one source of truth.

### Build

- [ ] `pnpm build` + `wails build` succeed for Windows (primary).
- [ ] Installer offers **directory page** (D: allowed).
- [ ] Portable zip runs without admin.

### Smoke (clean profile)

- [ ] First run creates data root.
- [ ] Open research folder → library tree.
- [ ] Open txt/md/docx/pdf; failures typed, not mock.
- [ ] Chat turn with tool + thinking/tool trace visible.
- [ ] Summary mirror / edit / accept or in-file review path works as on harness tip.
- [ ] Restart persists folder / core state as designed.

### Tag

- [ ] Annotated `vX.Y.Z` on green commit; push tag; attach installer + zip.

## After tag

Return to [docs/plans/00-current-priorities.md](../plans/00-current-priorities.md) Phase B. Do not pull FUTURE items without updating priorities.
