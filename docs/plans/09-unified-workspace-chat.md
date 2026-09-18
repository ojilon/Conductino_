# Unified workspace chat — one conversation per folder

> Status: **landed** (frontend). Born from a real session: every opened file
> spawned its own AI sidebar thread, so asking about an introduction and then
> saying "add it to the summary" lost context across documents.

## 1. Rule

**Thread identity = `workspaceId`.** The open document is a region anchor
and provenance hint — never thread identity. Switching files keeps the
conversation; a fresh start is always explicit via **New chat**.

## 2. What changed

| Before | After |
|---|---|
| `ensureThread(docId)` matched `workspaceId + documentId` → a thread per file | matches `workspaceId` only (`aiController.ts`) |
| ChatFace re-ensured per `doc.id` | ensures once; follows the active workspace thread |
| Only "Clear thread" (destructive wipe) | **New chat** button (history preserved in `byId`, active switches) + Clear kept |
| "Chat about this document" empty state | "Workspace chat — one conversation per folder" |

`chat.ensure` + `chat.setActive` needed no changes: `newThread()` composes
them (`aiController.ts`). `ChatThread.documentId` stays as "started from"
provenance; matching ignores it.

## 3. Still open (do not regress into per-doc threads)

- **History on folder switch:** `ensureThread` finds/activates the workspace
  thread, but older messages live in SQLite (`SaveThread`/`AppendMessage`
  bridges). On `workspace.setActive`, load that workspace's thread from the
  backend instead of starting visually empty. (Planned: Phase 16 remainder.)
- **Thread list UI:** `byId` accumulates New-chat threads; the panel shows
  only the active one. A minimal switcher (title + updatedAt) comes with the
  chat-observability work (08 §4).
- **Backend thread authority:** frontend `AppState.chat` is still the live
  thread; SQLite is the archive. Flip to backend-owned threads with the
  workflow runner (11), not before.
