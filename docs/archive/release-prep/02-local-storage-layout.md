# 02 — Local storage layout

**Status:** ACTIVE  
**Part of:** `docs/release-prep/`  
**Critical:** Skills, intermediates, SQLite, cache, and logs all depend on these paths.

## Goals

- User can install the app on **any drive** (including D:).
- Data is **not** trapped under Program Files; it lives in a dedicated app-data location.
- Portable mode works (zip next to a `data/` folder or env override).
- Per-workspace state stays with the research folder where possible (`.conductino/`), while global state lives under app-data.

## Roots (resolve in this order)

1. **Environment override** (highest priority)
   - `CONDUCTINO_DATA` — absolute path to the app data root.
   - Optional later: `CONDUCTINO_DB` for the SQLite file only.
2. **Installed mode** — platform app-data directory under a `Conductino` folder
   - Windows example: `%APPDATA%\Conductino` or a user-chosen path recorded at install/first-run.
   - The installer directory page (see 03) lets the user pick the *program* location; the *data* location should default to a writable app-data path and remain overridable.
3. **Portable mode** — directory next to the executable, e.g. `./data/` or `./ConductinoData/`.
4. **Dev / temp mirror** — see [04-temp-debug-mirror.md](04-temp-debug-mirror.md).

Document the exact resolution function in Go (single package, e.g. `backend/services/paths` or similar) so every subsystem uses the same roots.

## Recommended tree under the app-data root

```
<app-data>/
  config.json                 # optional user prefs (theme, last folder, …)
  conductino.db               # SQLite (sessions, logs, extract_cache, …)
  skills/                     # global / user skills (*.md)
  cache/
    extract/
  logs/                       # optional file logs if not only SQLite
  workspaces/
    <workspace-id>/
      intermediates/          # see docs/plans/09-intermediate-documents.md
        sources/
        summary/
      …
```

## Per-workspace folder (inside the research folder)

```
<open-research-folder>/
  .conductino/
    skills/                   # workspace-scoped skills (win on name clash)
    # optional small state that should travel with the folder
```

Do not put large caches or the main SQLite file inside the research folder by default (keeps the user’s folder clean).

## First-run behaviour

- If the resolved app-data root does not exist, create it.
- Optionally show a short first-run panel: “Data will be stored at … (Change)”.
- Persist the chosen data path if the user overrides the default.

## Agent rules

- Never hard-code `C:\` or `%USERPROFILE%` alone; always go through the path resolver.
- All new writers (skills, intermediates, SQLite migrations) must use the resolver.
- Tests may set `CONDUCTINO_DATA` to a temp directory.
