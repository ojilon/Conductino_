# Implementation phases — ordered delivery

Each phase should leave the app shippable (no broken reader).

## Phase 0 — Docs & structure
- [x] Vision / gaps, harness architecture, backend structure, document model, phased plan

## Phase 1 — Backend AI package split
- [x] `backend/services/ai/{service,prompts,gemini}.go` + shim

## Phase 2 — Workspace identity & summary mapping
- [x] WorkspaceSession, primarySummaryDocument, folder bind, no first-summary-wins

## Phase 3 — Context pack for oneshot AI
- [x] contextPack, custom prompt, AIRequest fields, selection toolbar

## Phase 4 — Chat UI + thread persistence (still no tools)
- [x] ChatMessage / ChatThread domain + AppState.chat
- [x] Actions + reduceChat
- [x] AI_CHAT with messageHistory
- [x] Right panel tabs: Chat | Reading/Review
- [x] Composer + multi-turn thread (workspace-scoped)
- [x] SQLite chat tables (Phase 6)

## Phase 5 — Harness tools (folder-scoped)
- [x] Tool registry + Resolve guards
- [x] list_workspace, read_source, read_summary, propose_summary_edit
- [x] Chat tool loop; insertion → DocumentChange

## Phase 6 — Storage: SQLite as system of record
- [x] Expand StorageService (workspaces, documents, changes, chat, settings)
- [x] SQLiteStorage via modernc.org/sqlite (pure-Go) + memory fallback
- [x] App-data DB path; CONDUCTINO_DB override
- [x] Shared storage; restore last library root on Init
- [x] Bridge + Wails: Save/Load document, change, chat; SetPrimarySummary
- [x] RunAI fills SummaryContent from SQLite primary summary when omitted
- [x] Unit tests (memory + sqlite)

## Phase 7 — Slate adapter for summary
- [x] `toSlate` / `fromSlate` in `frontend/src/features/reader/slateAdapter.ts`
- [x] Summary view uses Slate (slate-react); block/segment remains canonical
- [x] `doc.blocks.replace` action for full sync on editor change
- [x] Pending AI insert/modify still card-based; accept re-hydrates editor
- [x] Marks: strong / em / highlightId round-trip through adapter
- [x] Never persist Slate JSON — only DocumentBlock[]

## Phase 8 — DOCX summary file + richer sources
- [x] Stdlib DOCX reader (ZIP+OOXML) → blocks with bold/italic + heading styles
- [x] Stdlib DOCX writer: blocks → minimal OOXML package under workspace
- [x] OpenFile supports `.docx` (alongside txt/md)
- [x] MD headings (`#`/`##`/`###`) promoted to heading blocks
- [x] `WriteSummaryDOCX` on Backend + Wails App; path containment under library root
- [x] Frontend `writeSummaryDOCX` + Save DOCX control on summary view
- [x] Unit tests: round-trip, path escape rejection, default name

## Phase 9 — @-resolution + full proposal edits (living summary)
- [x] `mentionIds` wire (TS + Go) — `@doc` tokens resolve to explicit ids
- [x] Composer autocomplete dropdown (typing aid; send-time resolution is truth)
- [x] Selection rides along in `AI_CHAT` as the region anchor
- [x] `propose_summary_edit` ops: insert / modify / delete (never append-only)
- [x] Prompt harmonization: redefine / restructure / remove allowed, user gatekeeps
- [x] Pending delete rendering in summary view (strike-through card)
- [x] `search_in_workspace` tool (keyword, root-only, capped snippets)
- [x] `changeId`-targeted chat revise (`focusedChange`: explicit / selection-overlap / revise-intent; same-block supersede)

## Phase 10 — Stable ids + extract cache + ranges (issues 7/8/19/23/24/26/27/28)
- [x] Content-addressed block IDs (`b-<hash>`, `services/blockids.go`) — reopening an unchanged file yields identical IDs; highlights/changes/anchors survive
- [x] Bounded reads: Stat-gated size refusal + `LimitReader` (oversized files never load into memory)
- [x] `extract_cache` table (SQLite + memory): root-anchored path + mtime + size key, 200-entry LRU eviction; open path consults it
- [x] Range selection/highlights (`TextSelection.range`, `Highlight.range`, UTF-16 block offsets; multi-block stays block-anchored)
- [x] Range-aware highlight rendering + range in context packs
- [x] Pending modifies as inline Slate decorations (click to accept/reject/revise); deletes/inserts stay card-based
- [ ] PDF text-layer coordinates (needs a `pdfjs-dist` leaf — block offsets are the interim system)
- [ ] Frontend windowed doc cache (open tabs still fully resident; backend cache covers re-extract cost)
