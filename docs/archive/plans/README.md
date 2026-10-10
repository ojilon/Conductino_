# Plans index (agent-friendly)

**Purpose.** This folder is the single source of truth for *what to build next* and *why*.  
Agents and humans should read this index first, then open only the numbered plan that matches the current task.

**Branch.** Work that changes these plans lands on `feature/ai-docs-and-release-prep` (or later feature branches). Do not commit plan changes directly to `main`.

## How to use these plans

1. Start with **00-current-priorities.md** — the ordered list of what is in scope *now*.
2. For release / install / local-storage / tags → go to **`docs/release-prep/`** (separate subfolder, first concrete work).
3. For AI harness, skills, intermediate documents, bidirectional reflection → **09–12** below.
4. Older numbered plans (01–08) remain as historical design notes. Prefer the new 00 + 09–12 + release-prep when they conflict.
5. Anything marked **FUTURE** is explicitly out of scope for the next release cycle.

## Current plan set (ordered)

| File | Status | Topic |
|------|--------|--------|
| [00-current-priorities.md](00-current-priorities.md) | **ACTIVE** | Ordered priorities + what is deferred |
| [09-intermediate-documents.md](09-intermediate-documents.md) | **ACTIVE** | Mirror files (md/txt + metadata), edit tracking, save path |
| [10-skills-and-workflows.md](10-skills-and-workflows.md) | **ACTIVE** | SKILL.md layer, dynamic guidance, self-documentation |
| [11-ai-surface-and-tools.md](11-ai-surface-and-tools.md) | **ACTIVE** | Tool host, safe folder jail, Python helpers, thinking events |
| [12-bidirectional-reflection.md](12-bidirectional-reflection.md) | **ACTIVE** | Live AI ↔ frontend document state, intermediate sync |
| [../release-prep/](../release-prep/) | **ACTIVE — DO FIRST** | Tags, first release, installer (choose drive), local storage layout, temp debug mirror |

## Historical / reference (do not treat as current todo)

| File | Notes |
|------|--------|
| 01-architecture-harness.md | Original harness mental model (still mostly valid) |
| 02–08 | Multi-provider, backend resplit, source reading, operations growth — many ideas absorbed into 09–12 + release-prep |
| ../architecture.md, ../future-work.md, ../state-model.md | Keep in sync when behaviour changes; prefer plans for *what to implement* |

## Agent rules when implementing

- Prefer the **ACTIVE** plans above.
- Never auto-apply AI edits to the user-visible document; always go through `DocumentChange` / proposal → user gate.
- Paths always go through `PathResolver.Resolve` (workspace jail).
- Intermediate files are the source of truth for AI context and live editing; DOCX/PDF are export formats.
- Local storage paths are decided at install / first-run; see `docs/release-prep/`.
- Keep commits small and reference the plan file + section in the commit message.
