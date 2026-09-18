# Academic harness vision — gaps vs current reader/AI

> Branch: `feature/academic-harness-reader-ai`
> Scope: reader state + document reading ↔ AI connection only (blast radius).
> This file is the authoritative “what we want vs what exists” map.

## 1. Product goal (one sentence)

Turn the reader into an **academic harness**: folder-scoped sources (read-only),
one (or more) editable summary documents, and an AI that can read the
workspace under guardrails, hold conversation, and propose reviewed edits to
the summary — not just one-shot answers on a highlighted snippet.

## 2. Current baseline (verified against tree)

| Capability | Status | Location |
|---|---|---|
| Selection → fixed AI actions | Working (block-level) | `DocumentView` toolbar → `aiController` |
| Free-form / custom prompts | Missing | — |
| Context beyond selection | Missing (selection text only) | `AIRequest.selection`, `ai.go` prompts |
| Right panel as chat | Missing (companion + change review) | `AIReadingPanel` |
| Continued conversation | Missing | — |
| Folder-scoped AI tools | Missing | — |
| Create summary under open folder | Missing | summaries exist in state only |
| Dynamic summary (merge/improve/shorten) | Working (insert/modify/delete proposals, user-gated) | `propose_summary_edit` + `change.*` |
| Session/folder → summary mapping | Working | `WorkspaceSession.primarySummaryId` |
| @-mentions of files/sources | Working (resolve to `mentionIds` + autocomplete) | `state/mentions.ts` |
| Reference a specific change/diff in chat | Partial (changeId revise via panel; chat-targeted revise planned) | `runRevise` / Phase 9 |
| Sources read-only, summary editable | Correct direction | `kind: source \| summary` |
| Slate / rich editor | Not present | contentEditable plain text |
| DOCX round-trip for summary | Not present | planned in `document-rendering.md` |
| Long-term storage | In-memory only | `backend/services/storage.go` |
| AI complexity in one file | Growing | `backend/services/ai.go` (~500+ lines) |

## 3. Goals (from product brief) → requirements

### 3.1 Highlight + flexible AI actions
- Keep fixed actions (explain, go deeper, include in summary) as shortcuts.
- Add **custom prompt** entry on the selection toolbar and in chat.
- Every request may carry: selection, optional custom instruction, and
  **document context** (see 3.2).

### 3.2 Document / folder context for explanations
- Algorithms must assemble context for the model:
  - Primary: selected span + surrounding blocks (or page window).
  - Secondary: whole active document (truncated/summarized if large).
  - Optional: other open sources in the same workspace folder (ranked or
    user-@’d).
- Backend owns context assembly and token budgeting; frontend only passes
  ids + selection + optional user prompt.

### 3.3 Right panel evolves to chatbot-style (harness)
- Message history per workspace (or per summary session).
- User can continue the conversation after an explanation.
- AI can propose summary edits from chat, not only from selection toolbar.
- Ability to **reference a change/diff** (“expound on this”, “shorten this
  insertion”) by id or by selecting the pending/accepted region.

### 3.4 Harness tools + security guard
- AI may **read** files under the **current workspace root only**
  (reuse `Filesystem.Resolve` containment).
- Explicit allow-list of operations: list tree, read text of allowed
  extensions, read summary document content, propose summary edits.
- **Never** write source files; only propose edits to summary documents.
- Optional later: create a new summary `.docx` under the open folder.

### 3.5 Summary document lifecycle
- User can create a new summary under the opened folder (workspace file +
  in-app document).
- Mapping: workspace/session → primary summary document id (fixes
  first-summary-wins).
- Dynamic summary: AI can insert, modify structure, merge claims across
  sources, or shorten — always as **pending DocumentChange**(s) the user
  accepts or rejects.
- User judges each change; can request revise / narrow / expound via chat.

### 3.6 Left (or unified) chat + @ targeting
- Normal conversational chat.
- `@filename` / `@source` / `@summary` to scope what the model should
  consider.
- Response may include text answer **and** proposed summary edits.

### 3.7 Rendering & editing contract
- Sources: non-editable, custom render (PDF/HTML/DOCX extract → display).
- Summary: editable DOCX-backed document; editor is Slate (or equivalent)
  with a **canonical wire format** that is not Slate’s internal value
  (see `03-document-model-and-slate.md`).
- Highlights remain on sources; summary also supports highlight + forward
  to AI (“expound on this region / this change”).

## 4. Explicit non-goals (this branch / phase set)
- Real browser engine (still MockWebPage).
- OCR / scanned PDFs.
- Editing source PDFs/DOCX.
- Multi-user collaboration.
- Cloud sync of workspaces (local-first).

## 5. Success criteria (reader + AI)
1. Custom prompt on selection reaches Gemini with assembled document context.
2. Chat panel holds multi-turn history scoped to the current workspace.
3. AI can list/read files only under the open folder root.
4. “Include in summary” / chat-driven edits target the **mapped** summary
   for that workspace, not “first summary in memory”.
5. Pending changes remain user-reviewed; chat can revise a specific change.
6. In-memory storage path is replaced or sidelined for sessions/sources/
   documents/changes that must survive restarts (backend SQLite or file
   sidecar under workspace).

## 6. Related existing docs
- `docs/ai-integration.md` — current AI contract (extend, don’t break).
- `docs/state-model.md` — AppState shape (sessionId / workspaceId needed).
- `docs/document-rendering.md` — renderer + extraction boundaries.
- `docs/architecture.md` — layers; backend purity rules.
- `tasks.md` §1–4 — bugs and extraction plan still in force.
