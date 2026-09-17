# Implementation phases — ordered delivery

Each phase should leave the app shippable (no broken reader).

## Phase 0 — Docs & structure
- [x] Vision / gaps, harness architecture, backend structure, document model, phased plan

## Phase 1 — Backend AI package split
- [x] `backend/services/ai/{service,prompts,gemini}.go` + shim

## Phase 2 — Workspace identity & summary mapping
- [x] WorkspaceSession, primarySummaryDocument, folder bind, no first-summary-wins

## Phase 3 — Context pack for oneshot AI
**Goal:** explanations use more than the highlight.

- [x] Frontend `contextPack.ts`: selection + local window + outline (budgeted).
- [x] Backend `context.go` + prompt `withContext` (custom prompt + pack).
- [x] Extend `AIRequest` (TS + Go): workspaceId, customPrompt, contextPack, includeDocumentContext.
- [x] Selection toolbar: Ask AI opens optional instruction textarea; Explain / Go deeper pass prompt + pack.
- [x] Go unit tests for prompt assembly / truncate.

**Exit:** “Go deeper” quality improves; custom prompt reaches the model.

**Note:** Context is assembled on the frontend while document blocks live in
AppState; backend appends the pack to the prompt. Phase 6 may move assembly to Go.

## Phase 4 — Chat UI + thread persistence (still no tools)
- Message model + backend store
- Right panel: Chat | Review tabs
- Multi-turn `AI_CHAT` without tools first

## Phase 5 — Harness tools (folder-scoped)
- Tool registry + Resolve guards
- list_workspace, read_source, read_summary, propose_summary_edit

## Phase 6 — Storage: SQLite as system of record

## Phase 7 — Slate adapter for summary

## Phase 8 — DOCX summary file + richer sources
