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
