# .dev-data — repo-local debug data root (pre-release)

This directory is the **dev stand-in for the production app-data root**
(`docs/release-prep/02-local-storage-layout.md`). It lets skills, mirrors,
SQLite, and cache be tested **without writing into the real user profile**.

## Layout

```
.dev-data/
  README.md   # this file (committed)
  skills/     # experimental SKILL.md files before promoting to backend/skills/
  work/       # stand-in for production work/ mirrors (summaries/*.md + *.json)
  cache/      # extract cache spillover
```

`conductino.db` may also live here when `CONDUCTINO_DATA` points at it
(SQLite creates it on first run).

## Detection order (backend/apppaths)

1. `CONDUCTINO_DATA` when set (tests set it to a temp dir).
2. `.dev-data` when this directory exists.
3. Legacy `backend/.work/` mirrors (until the resolver switch is complete).
4. Installed platform app-data (`%AppData%/Conductino`) or portable `./data`.

## Rules

- Everything under `.dev-data/` except this README is **gitignored**.
- Never commit API keys or real user research documents here.
- Promote stable skills to `backend/skills/`; promote mirror/cache writers
  to `backend/apppaths` (never a second hard-coded path).
