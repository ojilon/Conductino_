# Implementation phases — ordered delivery

Planning-only on this branch until phase work begins. Each phase should
leave the app shippable (no broken reader).

## Phase 0 — Docs & structure (this commit)
- [x] Vision / gaps
- [x] Harness architecture
- [x] Backend structure & storage direction
- [x] Document model / Slate notes
- [x] Phased plan

## Phase 1 — Backend AI package split (no behavior change)
**Goal:** `backend/services/ai.go` → `backend/services/ai/*`.

- [x] Move Gemini client, prompts, Run entry.
- [x] Keep identical events and operations (`services.NewAI` shim).
- [x] Update imports; `go build ./backend/...` + `go vet` OK.

**Exit:** explain / merge / revise still work end-to-end.

**Landed:** `backend/services/ai/{service,prompts,gemini}.go` + `ai_shim.go`; old `services/ai.go` removed.

## Phase 2 — Workspace identity & summary mapping
**Goal:** kill first-summary-wins.

- Add `workspaceId` (or sessionId) to Document / Source / Change / Activity.
- WorkspaceSession (or extend existing workspace service) holds
  `primarySummaryId`.
- Create-summary action: new summary doc + optional file under root.
- `runIncludeInSummary` resolves via mapping.

**Exit:** two folders / two summaries never cross-contaminate.

## Phase 3 — Context pack for oneshot AI
**Goal:** explanations use more than the highlight.

- `context.go`: selection + local window + outline (+ optional open docs).
- Extend `AIRequest` with flags / workspaceId.
- Custom prompt field on selection toolbar (simple textarea or modal).

**Exit:** “Go deeper” quality improves; custom prompt reaches the model.

## Phase 4 — Chat UI + thread persistence (still no tools)
**Goal:** chatbot-style panel.

- Message model + backend store (SQLite or workspace sidecar).
- Right panel: Chat | Review tabs.
- Multi-turn `AI_CHAT` without tools first (history in prompt).
- Reference pending change from chat (“revise this change”).

**Exit:** continued conversation on the same document/workspace.

## Phase 5 — Harness tools (folder-scoped)
**Goal:** AI can list/read under root and propose summary edits.

- Tool registry + Resolve guards.
- `list_workspace`, `read_source`, `read_summary`, `propose_summary_edit`.
- Chat loop in `chat.go`.
- @-mention resolution in composer.

**Exit:** user can ask “summarize the methods sections of these PDFs into
the summary” and get **pending** changes only.

## Phase 6 — Storage: SQLite as system of record
**Goal:** reduce in-memory authority.

- Implement `storage/sqlite.go`.
- Load workspace state on folder open; save documents/changes/chat.
- Frontend cache for open tabs only.
- Shrink reliance on `mock/data.ts` for desktop path.

**Exit:** restart app, reopen folder → summary, changes, chat survive.

## Phase 7 — Slate adapter for summary
**Goal:** editable summary without losing canonical blocks.

- `toSlate` / `fromSlate`.
- Replace contentEditable.
- Preserve change overlays and accept/reject.

**Exit:** typing in summary is reliable; AI inserts still reviewable.

## Phase 8 — DOCX summary file + richer sources (parallel tracks)
- Summary save-as DOCX under folder.
- PDF/DOCX extraction arms (from `tasks.md` §4) feeding same blocks.
- Range-precise highlights when renderers allow.

## Suggested PR sequence
1. Phase 1 alone (mechanical).
2. Phase 2 + domain type updates.
3. Phase 3 (product-visible quality).
4. Phase 4 UI + persistence of chat.
5. Phase 5 tools.
6. Phase 6 storage hardening.
7. Phase 7–8 editor/format.

## Testing notes
- Prefer pure Go tests for context budget and Resolve guards.
- Frontend: selection → request payload shape; accept/reject reducers.
- Manual: folder A vs folder B summary isolation; tool path traversal
  attempts must fail closed.

## Open decisions (resolve during Phase 2–4)
1. Global app DB vs `.conductino/` per folder.
2. One primary summary vs multiple summaries per workspace.
3. Chat panel only on the right vs dual left+right.
4. Auto-save interval for Slate → backend.
