# 00 — Current priorities (agent entry point)

**Status:** ACTIVE  
**Last updated:** 2026-09-26  
**Audience:** Humans and coding agents working on Conductino_

## Principle

Do the **release-prep and local-storage foundation first**. Almost every later feature (skills, intermediate documents, AI tools, logs, cache) depends on a stable place to write files that survives restarts and can be chosen by the user (including D: drive).

## Ordered work (do in this sequence)

### Phase A — Foundation (must land before heavy AI work)

1. **Release preparation & local storage** → entire folder `docs/release-prep/`
   - Tags + versioning
   - First real build / installer with **directory choice** (user can pick D:)
   - App data directory layout (SQLite, skills, cache, logs, intermediate files)
   - Temporary local-storage mirror **inside the repo** for debug before a release
   - Portable zip + NSIS installer notes

2. **Intermediate document model** → [09-intermediate-documents.md](09-intermediate-documents.md)
   - Mirror representation (md/txt + metadata) for sources and for the editable summary
   - Edit history / AI-vs-user marks
   - Save path from intermediate → standard DOCX/PDF (export)
   - Standby change tracking (future git-like tracker is deferred)

### Phase B — AI surface (builds on A)

3. **Skills & workflows** → [10-skills-and-workflows.md](10-skills-and-workflows.md)
4. **Tool host, safety, Python helpers, thinking events** → [11-ai-surface-and-tools.md](11-ai-surface-and-tools.md)
5. **Bidirectional live reflection** → [12-bidirectional-reflection.md](12-bidirectional-reflection.md)

### Phase C — Polish & observability (can interleave after B starts)

- Persist tool audit + usage into SQLite (`ai_log`, `tool_audit`)
- Frontend “Thinking” panel fed by richer phase/tool events
- First-run wizard that creates the app data dir and optional temp skills area

## Explicitly deferred (FUTURE)

Do **not** implement these in the current cycle unless a plan file is updated to promote them:

- Real browser engine (replace MockWebPage)
- Full git-like history on intermediate files (standby tracker is enough for now)
- Unrestricted host shell or arbitrary Python code execution
- Cloud backend / multi-user server mode
- Non-GC rewrite (Odin/Zig) of non-network packages
- Character-range diffs across all document types
- OCR / scanned-PDF pipeline (explicitly out of scope for first releases)
- Parallel multi-provider map-reduce workflows beyond current failover
- Commercial PDF SDKs

## Agent checklist before any code change

- [ ] Read this file and the specific plan for the task.
- [ ] Confirm the change does not require a local-storage path that does not yet exist (if it does, finish Phase A first).
- [ ] Paths that touch the filesystem must go through the existing `PathResolver` / workspace jail.
- [ ] AI must never write the user-visible summary without a `DocumentChange` proposal + user accept.
- [ ] Prefer intermediate files for AI context; treat DOCX/PDF as import/export formats.
- [ ] Keep commits small; reference `docs/plans/NN-….md §section` in the message.

## Related docs

- `docs/release-prep/` — first concrete work
- `docs/architecture.md` — layers (update when intermediate model lands)
- `docs/future-work.md` — status table (keep in sync after each phase)
