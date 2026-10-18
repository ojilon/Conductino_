# Changelog

All notable changes to this project are documented here.
Format follows [Keep a Changelog](https://keepachangelog.com/en/1.0.0/);
versioning follows [Semver](https://semver.org/) (`vMAJOR.MINOR.PATCH` tags).

Single source of truth for the version string: `backend/version/version.go`
(`Version`), mirrored in `wails.json` `info.productVersion` for the Windows
binary metadata. CI builds on `v*` tags and opens a **pre-release** by default.

## Unreleased

### Added — the mirror pipeline (document rendering/editing/save rework)

- `backend/docpipe`: every opened file is mirrored into `content.json`
  (format-free blocks) + `meta.json` (the guide: kind, page size, fonts,
  margins, page map, per-block hashes, source stamp) under
  `<app-data>/cache/extract/mirrors/<key>/`. Guide: `docs/mirror-pipeline.md`.
- Save bridge: `App.SaveDocumentMirror` → `docpipe.WriteBack` verifies only
  the changed blocks (hashes) + refuses stale saves, then rewrites the
  source (.docx via OOXML, .txt/.md plain). Live mirror push while typing:
  `App.UpdateMirrorContent` (the AI's `read_mirror` sees edits immediately).
- Library file ops: `CreateFile` (empty .docx/.txt/.md), `CreateFolder`,
  `RenameFile` + right-click context menu in the library tree.
- Workspace summary TARGET: right-click a .docx → "Make this the target
  summary document" (`SetSummaryTarget`/`GetSummaryTarget`, marker in
  app-data). AI finds it with the new `get_summary_target` tool, and
  `publish_summary` siphons proposals into it even when it is closed.
- `frontend/src/features/reader/MirrorCanvas.tsx`: custom paginated canvas
  driven by meta.json (page sizes, fonts, margins) with direct editing for
  docx/txt/md, colored highlights (yellow/green/blue/pink), right-click
  highlight menu (save permanently / recolor / remove / ask AI), and
  Ctrl+S save-to-file.
- `SelectionToolbar.tsx`: shared floating toolbar (Ask AI / Explain / Go
  deeper / To summary / highlight color dots) for canvas + PDF views.
- Skills folder UI: "Skills" button in the AI panel → `SkillsPanel.tsx`
  (browse/read/edit `skills/`, create custom folders like
  `skills/my_instructions/`, plus the sandboxed shell console).
- AI tools: `read_mirror`, `get_summary_target`, `read_skill`, `shell`
  (basic sandbox: cwd = open folder, 10s timeout, capped output, deny-list)
  — XML protocol + OpenAI-compatible function schemas; `skills/app-guide.md`
  is the AI's internal user guide (injected on chat turns).
- PDF view: highlights ledger (colored, removable) under the header.

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
