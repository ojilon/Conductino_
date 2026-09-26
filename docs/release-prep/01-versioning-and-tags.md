# 01 — Versioning and tags

**Part of:** `docs/release-prep/`  
**See also:** [06-ci-and-github-releases.md](06-ci-and-github-releases.md), [release-notes/](release-notes/)

## Scheme

- Semantic versions: **`MAJOR.MINOR.PATCH`** (tags look like `v0.0.1`, `v0.1.0`).
- Single source of truth for the version string (document it in the first implementation PR — e.g. build ldflags and/or `wails.json`).
- Pre-1.0 (`0.x.y`) may break freely; document milestones in changelog / release notes.

## Tags

- Annotated tags preferred:
  ```bash
  git tag -a v0.0.1 -m "v0.0.1"
  git push origin v0.0.1
  ```
- Tag may point at a commit on **any branch** (not only `main`).
- Do not move or delete published tags.
- Pushing a `v*` tag is what should trigger the **release workflow** (build assets + GitHub **pre-release** by default).

## Changelog and per-tag notes

- Root `CHANGELOG.md` with `## Unreleased`; move entries under the version heading when tagging.
- Preferred CI body: `docs/release-prep/release-notes/vX.Y.Z.md` (must exist on the **tagged** commit).
- Keep a Changelog style (Added / Changed / Fixed).

## Agent notes

- Do not invent version numbers in multiple places without updating the single source of truth.
- Local testing does **not** require a tag or Release; use `wails build` on disk first.
