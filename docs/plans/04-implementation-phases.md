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
**Goal:** chatbot-style panel.

- [x] `ChatMessage` / `ChatThread` domain + `AppState.chat`
- [x] Actions: `chat.ensure` / `setActive` / `append` / `clear` + `reduceChat`
- [x] `AI_CHAT` operation (Go + TS) with `messageHistory` in prompt
- [x] Right panel tabs: **Chat** | Reading/Review
- [x] Composer + multi-turn thread (workspace-scoped)
- [ ] SQLite / workspace sidecar persistence (deferred to Phase 6)

**Exit:** continued conversation on the same document/workspace.

**Note:** Threads live in AppState for now. Phase 6 stores chat in SQLite.

## Phase 5 — Harness tools (folder-scoped)
**Goal:** AI_CHAT can list/read workspace files and propose summary edits under Resolve guards.

- [x] Tool registry + parse (`tools.go`) — list_workspace, read_source, read_summary, propose_summary_edit
- [x] Resolve + extension guards; no shell; size caps
- [x] Chat tool loop max 2 rounds (`chat.go`); phase labels for tools
- [x] Wire `NewAI(fs, docs)` from Backend
- [x] Frontend: summary snapshot on chat request; insertion → DocumentChange
- [x] Unit tests for parse + path rejection

**Exit:** chat can list tree / read .txt|.md under the open folder; summary proposals still require user accept.

## Phase 6 — Storage: SQLite as system of record

## Phase 7 — Slate adapter for summary

## Phase 8 — DOCX summary file + richer sources
