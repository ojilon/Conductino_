# 10 — Skills and workflows

**Status:** ACTIVE  
**Depends on:** local storage roots (`docs/release-prep/`), intermediate documents (09)  
**Audience:** Agents adding instruction layers and multi-step AI behaviour

## Goal

Give the AI living, versioned, workspace-scoped instructions so it knows *how* to use tools, how to edit the summary, how to handle large documents, and how to recover from problems — without growing a single giant system prompt.

## Skill file format

One Markdown file per skill:

```md
---
skill: summarize-source
version: 1
when: ["user asks to summarize", "AI_CHAT + propose_summary_edit", "include-in-summary"]
tools: [list_workspace, read_source, read_summary, propose_summary_edit, search_in_workspace]
budgets: { maxToolCalls: 6, maxChars: 8000, maxRounds: 3 }
---
# Summarize a source into the living summary

1. Call read_summary first (harmonize, never append blindly).
2. Read the source via intermediate files / windows; never summarize unread content.
3. One propose_summary_edit per claim; modify and delete are allowed.
4. If a definition is superseded, redefine and cite both.
```

Frontmatter is machine-readable (routing + budgets).  
Body is natural-language steps the model must follow.

## Discovery locations (priority high → low)

| Layer | Path | Ownership |
|-------|------|-----------|
| Workspace | `<workspace-root>/.conductino/skills/*.md` | Per research folder; wins on name clash |
| User / global | `<app-data>/skills/*.md` | Cross-folder personal routines |
| Bundled | embedded `backend/skills/*.md` | Ships with the app |
| Temporary / debug | see `docs/release-prep/04-temp-debug-mirror.md` | Dev only, before release |

At prompt build the harness selects at most 1–2 matching skills (token-budgeted) and injects their bodies plus any dynamic notes.

## Dynamic and AI-written guidance

- User can edit any skill file; the next turn sees the new text.
- New tool (or extension of existing): `document_problem` / `update_guidance`
  - AI may write a short failure note + suggested fix into a `guidance_notes` store (SQLite or a markdown file under `.conductino/` or app-data).
  - Future turns can pull recent guidance under a “lessons learned” section.
- Never put secrets, absolute host paths, or full file contents into skills or guidance notes.

## Workflows (composition, not a second tool system)

A workflow is a skill that has an ordered, **gated** runner:

- Example: `summarize-folder` = list → read windows → draft proposals → **user gate each** → merge pass.
- Gates are mandatory; no workflow may write or propose past a user checkpoint without an explicit setting later.
- Implemented as composition of existing tools + the chat loop, not a separate agent runtime.

Full multi-step workflow runners are lower priority than the skill injection layer and the intermediate document model. Implement the loader + injection first.

## Starter skill set (implement these first)

1. `summarize-source`
2. `find-in-workspace`
3. `revise-proposal`
4. `quota-aware` (rate-limit / failover behaviour)
5. `use-intermediate` (always prefer intermediate files + recent edit meta)

## Agent rules

- Skills are instructions only; tools remain the only way to touch files.
- Budgets in frontmatter are soft hints; the Go tool loop still enforces hard caps.
- Keep each skill body short (one screen). Long skills get skipped by models.
- When adding a skill, add a matching entry to the test table or a small fixture under the temp debug mirror.
