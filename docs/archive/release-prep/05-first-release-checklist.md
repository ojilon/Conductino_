# 05 — First release checklist

**Status:** ACTIVE  
**Part of:** `docs/release-prep/`

Use this list before tagging the first real desktop release (`v0.1.0` or similar).

## Code & docs

- [ ] Path resolver implemented and used by storage, skills, intermediates, logs.
- [ ] `.dev-data/` mirror works for local debugging (04).
- [ ] `CHANGELOG.md` has an `Unreleased` section ready to become the release notes.
- [ ] Version string has a single source of truth.
- [ ] Plans index (`docs/plans/README.md`) and `00-current-priorities.md` still match reality.

## Build

- [ ] `pnpm build` in `frontend/` succeeds.
- [ ] `wails build` produces a Windows artefact (primary target).
- [ ] Installer offers a **directory page** (user can choose D: or other).
- [ ] Portable zip runs without installation and respects `CONDUCTINO_DATA` or side-by-side data dir.

## Smoke test (clean machine or clean profile)

- [ ] Install (or unzip) → first run creates app-data root.
- [ ] Choose / open a research folder → library tree appears.
- [ ] Open `.txt` / `.md` / `.docx` / `.pdf` → content shows; failures are typed, not mock.
- [ ] One AI chat turn that uses a tool (e.g. list or read) completes; phase labels visible.
- [ ] Accept or reject a summary proposal; intermediate / save path behaves as documented.
- [ ] Restart app → chosen folder and basic state persist as designed.

## Tag & publish

- [ ] All checks green on the release commit.
- [ ] Update `CHANGELOG.md` (move Unreleased → version heading).
- [ ] Annotated tag `vX.Y.Z` and push tag.
- [ ] Attach installer + portable zip to the GitHub Release (or internal distribution point).

## After the tag

- Return to `docs/plans/00-current-priorities.md` Phase B (skills, intermediates, AI surface).
- Do not expand scope into FUTURE items without updating the plans.
