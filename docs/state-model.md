# Lumen — State model

Everything is defined in `src/types/domain.ts` and mutated only in `src/state/appState.tsx`.

## Root shape

```ts
AppState = {
  mode: "browser" | "reader",

  browser: {                 // application state (persistable)
    sessions: BrowserSession[]   // research sessions, each owns tabIds + activeTabId
    activeSessionId
    tabs: Record<id, BrowserTab> // each tab owns NavigationState {currentUrl, history, index}
    ui: BrowserUIState           // EPHEMERAL: sidebar/panel open, panel width
  },

  reader: {                  // application state (persistable)
    session: { id, tabIds, activeTabId }   // reader subtabs
    tabs: Record<id, ReaderTab>            // ReaderTab → documentId + label
    ui: ReaderUIState            // EPHEMERAL: rail/sidebar/panel, companion payload
  },

  sources: Record<id, Source>,        // the research corpus (web + local + summary)
  documents: Record<id, Document>,    // structured DocumentModel instances
  changes: Record<id, DocumentChange>,// AI proposals (pending/accepted/rejected)
  aiActivities: AIActivity[],         // newest first — "what is AI doing right now?"
  browse: BrowseState,                // AI Browse pipeline status + ranked sourceIds

  // ephemeral
  selection: TextSelection | null,    // live text selection + toolbar position
  toast, settingsOpen, previewSourceId
}
```

## What is persistent vs ephemeral

| Persistent (SQLite candidates) | Ephemeral (UI, never stored) |
|---|---|
| `browser.sessions` / `tabs` (as history lists) | `*.ui` (panel widths, open/closed) |
| `sources` (incl. `saved`, `rank`, `relevance`) | `selection` |
| `documents` (blocks, highlights) | `toast`, `previewSourceId`, `settingsOpen` |
| `documents[].sourceIds` (summary ← sources) | `browse.phase` (re-runnable pipeline state) |
| `changes` (audit trail of accepted/rejected) | `aiActivities[].status === "running"` |
| `aiActivities` (history) | `reader.ui.companion` (recomputable) |

## Key relationships

- **Browser:** `BrowserSession → tabIds[] → BrowserTab`. A tab *is* a navigation stack; opening/closing tabs never destroys history of other tabs.
- **Reader:** `ReaderSession → tabIds[] → ReaderTab → Document`. Multiple subtabs open simultaneously; `activeTabId` is the current document. Views stay mounted, so scroll position + edits survive switches.
- **Source ↔ Document:** `Document.sourceId → Source`; `Source.documentId` points back once a source is opened in the Reader. A source may be opened, closed and reopened — the `reader.doc.open` action reuses the existing tab if one exists. Files opened from disk (`.txt` via `App.OpenFile`) get a fabricated `Source` (`kind: "text"`, `origin` = the root-relative path token) — real blocks, mock provenance.
- **File ↔ Document location:** `metadata.path` holds the root-relative `FileTreeNode.path` token for disk-opened files (Send-to-Reader docs carry no path). "Show in library" navigates the Library rail to that token. Tokens resolve against the CURRENT workspace root — see `tasks.md` §1.1 for the stale-tab bug this creates on folder switch.
- **Multi-session scaling (known limitation):** `sources` / `documents` / `changes` are flat global maps with no session ownership. `runIncludeInSummary` picks its target with first-summary-wins (`aiController.ts:122`), which breaks the moment two session/summary pairs coexist. Do not build features assuming "the" summary or "the" folder — see `tasks.md` §2 for the `sessionId` design constraint.
- **Summary ↔ Sources:** `Document.kind === "summary"` carries `sourceIds[]` — the explicit "these papers fed this summary" relationship. The top bar's "N sources contribute to this summary" reads exactly this. Adding a source happens in one reducer case (`summary.addSource`), triggered when an AI merge proposes an insertion from that source.
- **Change ↔ Activity ↔ Source:** `DocumentChange.activityId → AIActivity` and `change.sourceId → Source` answer "where did this edit come from and who proposed it" (shown in the Inspect view).
- **Chat targeting:** composer `@tokens` resolve to `AIRequest.mentionIds` (`state/mentions.ts`); a mentioned summary overrides the workspace primary as proposal target. The live `selection` rides along as the region anchor.
- **Living summary:** proposals are insert/modify/delete (never append-only) — the model may redefine or restructure against the summary snapshot; every change stays pending until the user accepts or revises.

## DocumentModel (the renderer contract)

```ts
Document = {
  id, kind: "source" | "summary", sourceId,
  metadata: { title, author?, venue?, year?, format, pageCount?, path? },
  blocks: DocumentBlock[],      // ordered; renderer maps over these
  highlights: Highlight[],      // block-anchored annotations (note / saved selection)
  sourceIds?: ID[],             // summary only
  currentPage?: number          // mock pagination
}
DocumentBlock = { id, type: "heading"|"paragraph"|"list"|"page", level?,
                  segments: Segment[], listItems?: Segment[][], changeId? }
Segment = { text, em?, strong?, highlightId? }
```

Source documents are **read-only** with native text selection; summary documents are **block-editable** (plain-text contentEditable). Both render from the same model — the behavioral difference lives in the two view components, not in duplicated data structures.

## Selection → AI workflow (state path)

```
native selection (mouseup)
  → "selection.set" { documentId, blockId, text, x, y }
  → SelectionToolbar renders (fixed, at x/y)
  → user picks action
  → useAIRunners.runX() records an AIActivity ("running")
  → provider streams phases → "activity.update"
  → onDone → companion payload / DocumentChange / highlight
  → "selection.set" null (where appropriate)
```

`TextSelection` stores viewport coordinates for the toolbar plus optional
`range { start, end }` offsets in the block's text (single-block selections;
multi-block stays block-anchored); scrolling the document clears it (cheap
and predictable). Block IDs are content-addressed and stable across
re-extracts, so selections, highlights, and pending changes survive reopening.

## Where things live — quick index

- "Where does document state live?" → `state.documents`, mutated by `doc.*` and `change.*` actions.
- "Where are AI activities tracked?" → `state.aiActivities` (history) + `state.browse` (pipeline).
- "Where is the summary stored?" → `state.documents["…"].kind === "summary"`, with `sourceIds` for provenance.
- "Where is the pending-change queue?" → `state.changes` filtered by `status: "pending"` per document (`pendingChangesFor`).
- "Where is the library tree?" → NOT in `AppState`. `ReaderMode.tsx` holds `tree`/`treeError`/`locatePath` in `useState` as re-fetchable server-mirror cache from `backend.library.list()` (null in browser — Choose-folder empty state; `walkDir` tree in desktop). Open/file-expand state lives in `FileTree`, keyed by path token.
- "Where is the chosen folder remembered?" → nowhere persistent. Go `Filesystem.root` (memory only) + a toast. Restart forgets it; `Workspace.Save` does not persist the root (see `tasks.md` §4-adjacent gap list).
