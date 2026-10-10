# Release preparation (do this first)

**Status:** ACTIVE — foundation for almost everything else  
**Branch:** `feature/ai-docs-and-release-prep`  
**Audience:** Agents and humans preparing the first real desktop release

## Why this comes first

Skills, intermediate documents, SQLite, cache, logs, and the AI’s local workspace all need a **stable, user-chosen place on disk**.  
The installer must let the user pick the drive (e.g. D: instead of C:).  
Until that exists, later features have nowhere safe to write.

## Files in this folder (read in order)

| File | Topic |
|------|--------|
| [01-versioning-and-tags.md](01-versioning-and-tags.md) | Semantic version, tags, changelog |
| [02-local-storage-layout.md](02-local-storage-layout.md) | App-data roots, per-workspace dirs, env overrides |
| [03-build-and-installer.md](03-build-and-installer.md) | Wails build, NSIS directory choice, portable zip |
| [04-temp-debug-mirror.md](04-temp-debug-mirror.md) | Temporary local-storage mirror **inside the repo** for pre-release testing |
| [05-first-release-checklist.md](05-first-release-checklist.md) | End-to-end checklist before tagging v0.x.y |

## Agent rules

- Implement storage path resolution **before** writing skills, intermediates, or new SQLite tables that assume a fixed location.
- Prefer environment overrides (`CONDUCTINO_DATA`, etc.) so portable and installed modes both work.
- Keep each of these guidance files small; do not merge them into one large document.
- After the first release path works, return to `docs/plans/00-current-priorities.md` for Phase B.
