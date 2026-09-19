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

## 3. Landed since (do not regress into per-doc threads)

- **Make workspace summary:** disk-opened files used to stay kind "source"
  forever (`workspace.createSummary`/`setPrimarySummary` had no UI), so
  every merge/propose path silently dropped for lack of a target. Now the
  source header has a **Make summary** button: flips the doc editable +
  sets the workspace primary (persisted via `App.SetPrimarySummary`).
  `workspace.setPrimarySummary` owns the kind flip in the reducer.
- **No-summary guidance:** when proposals arrive but no summary is mapped,
  the UI toasts the Make-summary fix instead of dropping silently.
- **History on folder switch:** `pickFolder` loads the workspace's threads
  from SQLite (latest first) via `loadWorkspaceThreads`; in-memory wins on
  id collision; browser loads nothing.
- **Thread list UI:** workspace threads (newest first) as pills above the
  composer; click activates. History preserved by New chat.
- **Backend persistence:** every thread create + message append persists
  best-effort (`saveChatThread`/`appendChatMessage`; silent no-op in browser).
- **Still open:** backend-owned threads (SQLite as authority) flip with the
  workflow runner (11), not before.
