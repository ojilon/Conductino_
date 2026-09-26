# 01 — Versioning and tags

**Part of:** `docs/release-prep/`

## Scheme

- Semantic versions: `vMAJOR.MINOR.PATCH` (e.g. `v0.1.0`).
- Single source of truth for the version string (document it in the first implementation PR — e.g. build ldflags and/or `wails.json`).

## Tags

- Annotated tags only on a green release commit:
  ```bash
  git tag -a v0.1.0 -m "v0.1.0 — first desktop release"
  git push origin v0.1.0
  ```
- Do not move or delete published tags.

## Changelog

- Root `CHANGELOG.md` with `## Unreleased`; move entries under the version heading when tagging.
- Keep a Changelog style (Added / Changed / Fixed).

## Agent notes

- Do not invent version numbers in multiple places without updating the single source of truth.
