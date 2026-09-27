# 02 — Local storage layout

**Part of:** `docs/release-prep/`  
**Critical for:** SQLite, skills, cache, logs, and **promoting** today’s `backend/.work/` mirrors to a user-owned data root.

## Goals

- Install program files on any drive (including **D:**).
- Keep **data** (DB, skills, mirrors, cache) in a writable app-data (or portable) root — not locked under Program Files.
- One path resolver used by every subsystem.

## Resolution order

1. `CONDUCTINO_DATA` (absolute app-data root) if set  
2. Installed mode — platform app-data under `Conductino` (or path recorded at first-run)  
3. Portable mode — directory next to the executable (e.g. `./data/`)  
4. Dev — see [04-temp-debug-mirror.md](04-temp-debug-mirror.md); today also `backend/.work/` for summary mirrors

Optional later: `CONDUCTINO_DB` for SQLite file only.

## Suggested tree under app-data root

```
<app-data>/
  config.json
  conductino.db
  skills/                 # global user skills
  cache/extract/
  workspaces/<id>/intermediates/   # if multi-workspace state is stored centrally
  work/                   # production home for what is now backend/.work/ (mirrors, op logs)
```

## Per research folder

```
<open-folder>/
  .conductino/
    skills/               # workspace skills (win on name clash — plan 11)
```

## Relation to current code

- `backend/mirror` today uses `backend/.work/summaries` (git-ignored).  
  **Target:** same package, root taken from the path resolver so installed builds write under app-data / portable data.
- Do not break dev workflow: resolver must still prefer explicit env or `.dev-data` / `.work` when present.

## First-run

- Create missing app-data root.
- Optional UI: show path + “Change”.
- Persist override if the user picks a custom data directory.

## Agent rules

- No hard-coded `C:\Users\…` alone; always go through the resolver.
- Tests set `CONDUCTINO_DATA` to a temp dir.
