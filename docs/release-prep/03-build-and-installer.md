# 03 — Build and installer

**Status:** ACTIVE  
**Part of:** `docs/release-prep/`

## Goal

Produce a real desktop app the user can install **on a drive of their choice** (including D:) and a portable zip that needs no installer.

## Build commands (current stack)

From repo root (Wails project):

```bash
# Frontend production assets
cd frontend && pnpm build && cd ..

# Desktop binary / installer
wails build
```

- Prefer `pnpm` (lockfile is `pnpm-lock.yaml`).
- After build, artefacts typically appear under `build/bin/` (confirm against current `wails.json`).

## NSIS / Windows installer — directory choice

Wails uses NSIS on Windows. Configure so setup shows a **directory page**:

- User can change the install location (e.g. `D:\Programs\Conductino`).
- Default may remain under Program Files; the important part is that the choice is offered.
- Record the chosen program path if needed; **data** still prefers the app-data layout in [02-local-storage-layout.md](02-local-storage-layout.md) so user documents and DB are not locked under Program Files.

Concrete steps for agents:

1. Inspect `wails.json` for existing `nsis` / info settings.
2. Enable or add the directory page (MUI_PAGE_DIRECTORY or Wails equivalent).
3. Verify uninstall removes the program files cleanly and does **not** delete the user’s app-data by default (or offers a clear choice).

## Portable zip

- Ship a zip of the same binary + any required adjacent files.
- Portable mode should use a data directory next to the executable (or `CONDUCTINO_DATA`).
- No admin rights required.

## Signing / SmartScreen (future note)

Not required for the first internal release. Document later if public distribution needs code signing.

## Agent rules

- Do not commit built binaries into git.
- Document any new `wails.json` keys in this file when changed.
- Smoke-test after build: open folder → open a pdf/docx → one chat turn with a tool call.
