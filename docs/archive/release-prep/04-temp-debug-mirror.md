# 04 — Temporary local-storage mirror (dev / pre-release)

**Status:** ACTIVE  
**Part of:** `docs/release-prep/`

## Goal

Provide a **repo-local** mirror of the app-data layout so you (and agents) can test skills, intermediates, SQLite, and cache **before** cutting a release — without installing the app and without writing into real user profile directories.

## Location

Suggested path (create when implementing):

```
.dev-data/                  # at repo root, gitignored except a README
  README.md                 # explains purpose; committed
  skills/                   # temporary skills for experiments
  workspaces/
    sample/
      intermediates/
  conductino.db             # optional local sqlite for tests
  cache/
```

- Add `.dev-data/` to `.gitignore` **except** `.dev-data/README.md` (and maybe empty `.gitkeep` files if useful).
- Never commit real user documents or API keys into this tree.

## How the app finds it

In development (`wails dev` or tests):

1. If `CONDUCTINO_DATA` is set → use it.
2. Else if a `.dev-data` directory exists next to the module / working directory → use it.
3. Else fall through to normal app-data resolution.

Document the exact detection in the same path-resolver package as production roots.

## Temporary skills folder

Use `.dev-data/skills/` (and/or a workspace `.conductino/skills/` under a sample folder) to trial SKILL.md files before promoting them to:

- bundled `backend/skills/` (for shipping), or
- the real user/global skills directory after install.

## Agent rules

- Prefer writing test artefacts under `.dev-data/` during development.
- Clean or document any large cache files so the repo stays small.
- Do not treat `.dev-data` as production storage; production always uses the resolved app-data root from [02-local-storage-layout.md](02-local-storage-layout.md).
