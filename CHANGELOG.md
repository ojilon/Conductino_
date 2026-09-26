# Changelog

All notable changes to this project are documented here.
Format follows [Keep a Changelog](https://keepachangelog.com/en/1.0.0/);
versioning follows [Semver](https://semver.org/) (`vMAJOR.MINOR.PATCH` tags).

Single source of truth for the version string: `backend/version/version.go`
(`Version`), mirrored in `wails.json` `info.productVersion` for the Windows
binary metadata. CI builds on `v*` tags and opens a **pre-release** by default.

## Unreleased

(nothing yet — add new entries here after tagging v0.0.2)

## v0.0.2

First tagged pre-release (test release, Windows only).
Full notes: `docs/release-prep/release-notes/v0.0.2.md`.

### Added

- `backend/apppaths`: single path resolver (`CONDUCTINO_DATA` → `.dev-data` →
  portable `./data` → platform app-data) with `DBPath`, `WorkDir`,
  `SummariesDir`, `SkillsDir`, `CacheDir`, `EnsureDataRoot`.
- `backend/version`: single source of truth (`Version = "0.0.2"`).
- Mirror, SQLite, and skill loading resolve through `apppaths`; dev still
  works with legacy `backend/.work/` until `.dev-data` exists.
- `.dev-data/` repo-local debug root (committed README, rest gitignored).
- CI (`.github/workflows/ci.yml`) and tag-release
  (`.github/workflows/release.yml`) workflows; pre-release by default.
- Windows installer (NSIS) with install-directory page (D: allowed);
  portable-zip job alongside the installer.
- Release notes input `docs/release-prep/release-notes/v0.0.2.md` (CI uses it
  as the GitHub Release body).
