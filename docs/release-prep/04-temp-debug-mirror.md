# 04 — Temp debug mirror (pre-release)

**Part of:** `docs/release-prep/`

## Goal

A **repo-local** data root for testing skills, mirrors, SQLite, and cache **before** a real install — without writing into the real user profile.

## Suggested layout

```
.dev-data/                 # repo root; mostly gitignored
  README.md                # committed — explains purpose
  skills/
  work/                    # stand-in for production work/ mirrors
  conductino.db            # optional
  cache/
```

Also keep using / formalize interaction with existing **`backend/.work/`** so current mirror tests keep working until the resolver is switched.

## Detection (dev)

1. `CONDUCTINO_DATA` if set  
2. Else `.dev-data` if present  
3. Else `backend/.work` behaviour as today for mirrors  
4. Else normal app-data resolution  

## Temporary skills

Put experimental SKILL.md files under `.dev-data/skills/` or workspace `.conductino/skills/` before promoting to `backend/skills/`.

## Agent rules

- Prefer writing test artefacts under `.dev-data/` or `.work/`, not random paths.
- Never commit API keys or real user research documents.
