# 01 — Versioning and tags

**Status:** ACTIVE  
**Part of:** `docs/release-prep/`

## Version scheme

- Use **semantic versioning**: `vMAJOR.MINOR.PATCH` (example: `v0.1.0`).
- Pre-1.0 is expected; breaking changes are allowed in 0.x with a clear changelog entry.
- Single source of truth for the version string: keep it in one place that both Go and the frontend can read (e.g. a small `version` package or `wails.json` + build ldflags). Document the chosen location in the first PR that implements tagging.

## Tags

- Create **annotated** tags only on green `main` (or the release branch after merge):
  ```bash
  git tag -a v0.1.0 -m "v0.1.0 — first desktop release"
  git push origin v0.1.0
  ```
- Never move or delete a published tag.
- Tag message should mention the main user-visible change.

## Changelog

- Maintain `CHANGELOG.md` at repo root.
- Keep an `## Unreleased` section at the top; move items under the new version heading when tagging.
- Format: Keep a Changelog style (Added / Changed / Fixed / Security).

## Agent notes

- Do not invent version numbers in code without updating the single source of truth and the changelog.
- CI (when added) should run on tags and attach build artefacts.
