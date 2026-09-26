# Release preparation (foundation)

**Status:** ACTIVE — next foundation work  
**Code base:** branch off `feature/academic-harness-reader-ai`  
**Docs branch:** `feature/docs-release-prep-on-harness`

## Why first

Skills, mirrors, SQLite, cache, and logs need a **stable place on disk** the user can put on **D:** (or any drive). The current `backend/.work/` mirror dir is fine for dev; production needs install-time / first-run app-data roots and an installer directory page.

## Read in order

| File | Topic |
|------|--------|
| [01-versioning-and-tags.md](01-versioning-and-tags.md) | Semver, annotated tags, changelog |
| [02-local-storage-layout.md](02-local-storage-layout.md) | App-data, `CONDUCTINO_DATA`, relation to `.work` / `.conductino` |
| [03-build-and-installer.md](03-build-and-installer.md) | Wails build, NSIS directory choice, portable zip |
| [04-temp-debug-mirror.md](04-temp-debug-mirror.md) | Repo-local debug data root before release |
| [05-first-release-checklist.md](05-first-release-checklist.md) | Smoke + tag checklist |

## Agent rules

- Implement path resolution before assuming fixed `C:\` or only `backend/.work/`.
- Do not commit built binaries or user documents.
- After path roots exist, promote mirror/skills/cache writers to use them.
