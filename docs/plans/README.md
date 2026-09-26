# Plans index (agent-friendly)

**Branch base:** `feature/academic-harness-reader-ai` @ `one messy push, seriously . . .`  
**This docs branch:** `feature/docs-release-prep-on-harness` (docs only; no merge to main yet)

## How agents should use this folder

1. Read **[00-current-priorities.md](00-current-priorities.md)** first — ordered work and what is already landed.
2. For **install / tags / local storage / choose D: drive** → **`docs/release-prep/`** (do this foundation next).
3. For AI harness behaviour already in flight → plans **10** and **11** (they match the current code more closely than older 01–08).
4. Older plans (01–08, 00-vision) remain historical design notes. Prefer 00 + 10 + 11 + release-prep when they conflict.

## Active set

| File | Status | Topic |
|------|--------|--------|
| [00-current-priorities.md](00-current-priorities.md) | **ACTIVE** | Ordered priorities vs landed code |
| [10-live-library-and-intermediates.md](10-live-library-and-intermediates.md) | **partially landed** | Mirrors, empty docs, page canvas |
| [11-skills-workflows-and-split.md](11-skills-workflows-and-split.md) | **partially landed** | Skills, workflows, summary edit loop, package split |
| [../release-prep/](../release-prep/) | **ACTIVE — next foundation** | Tags, installer drive choice, app-data layout, temp debug mirror |

## Historical / reference

01–09 and phase notes: keep for context; do not treat as the current todo list.

## Agent rules

- Paths: workspace jail via `tools` Resolve; summary AI surface via `mirror` (not raw DOCX bytes).
- User gate: proposals / in-file diffs before durable summary writes where the product requires it.
- Non-network packages (`extract`, `tools`, `skills`, `usage`, `mirror`, `models`): keep stdlib-first and rewritable later (see plan 11 §5).
- Reference plan file + section in commit messages.
