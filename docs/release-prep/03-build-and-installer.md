# 03 — Build and installer

**Part of:** `docs/release-prep/`

## Build

```bash
cd frontend && pnpm build && cd ..
wails build
```

- Use **pnpm** (`pnpm-lock.yaml`).
- Artefacts typically under `build/bin/` (confirm `wails.json`).

## Windows installer — choose install directory (e.g. D:)

- Wails/NSIS: enable a **directory page** so the user can install under `D:\Programs\Conductino` (or any path).
- Program location ≠ data location: data should still follow [02-local-storage-layout.md](02-local-storage-layout.md).
- Uninstall: remove program files; do **not** delete app-data by default (or ask clearly).

## Portable zip

- Same binary + adjacent data dir or `CONDUCTINO_DATA`.
- No admin required.

## Agent rules

- Do not commit binaries.
- After build, smoke: open folder → open pdf/docx → one tool-using chat turn.
