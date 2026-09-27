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

## Phase 11 — Provider policy + audit + autosave + DOCX fidelity
- [x] Cost classes S/M/L + global semaphore (2 slots, "Waiting for AI capacity…" phase) + 5-min explain cache (oneshots only)
- [x] Usage telemetry (per-provider calls/errors/token-est/RPM ring) + tool audit ring (names + paths only) → Settings via `App.AIMeters`
- [x] Explicit per-turn tool budget (6 across rounds, was implicit)
- [x] Gated summary autosave: debounced 3s, accepted/user content only (pending inserts excluded), silent success / honest failure toast
- [x] DOCX reader: flat lists (`w:numPr` grouping) + tables as "a | b" rows, ordered token walk; writer: lists as "• " paragraphs
- [ ] Arbitrary shell-exec tool — evaluated, rejected: Windows lacks grep/rg, Go-native `search_in_workspace` covers "find in docs" without leaving the Resolve jail (see `01-architecture-harness.md`)

## Phase 12 — PDF extraction + canvas leaf (pnpm-managed)
- [x] `pdfjs-dist` via pnpm (never npm — `pnpm-lock.yaml` is canonical; `public/pdf.worker.min.mjs` ships in dist)
- [x] Go text-layer extraction (`ledongthuc/pdf`, pure Go): per-page blocks + `page`-type break anchors, stable IDs, 50 MiB / 500-page caps
- [x] `read_source` + `search_in_workspace` accept pdf/docx (both extract for real now)
- [x] `App.ReadRawFile` bridge (Resolve-jailed base64, 30 MiB cap) → `openRawFile` TS helper
- [x] `PdfView` canvas leaf: lazy per-page raster (IntersectionObserver, far pages release canvases) + pdf.js text layer for native selection
- [x] Page containers anchor to extraction page-break blocks (`data-block-id`) — chat/AI stack unchanged
- [ ] Scanned/image-only PDFs (no text layer — documented limitation, no OCR planned)

## Phase 13 — Native function tools (fix "Tool choice is none")
- [x] Declare the 5 tools as OpenAI function schemas (`tool_choice: auto`) — tool-trained models (gpt-oss) no longer emit undeclared calls
- [x] Translate native `tool_calls` back to internal `<tool>` tags (XML-escaped round-trip); text-embedded tags still parse as fallback
- [x] No-tools retry for endpoints without function calling; "tool choice" advances the model fallback chain

## Phase 14 — Backend resplit + paged source reading (planned: 06 + 07)
- [x] 06 steps 1–3: carve `extract/` (extract/text/docx/pdf/ids + normalize/page stubs) → `tools/` (tools/paths/search) → `usage/` (usage/policy); one-release aliases; `ai/` top-level, network-only
- [x] 06 step 4: split `models/` (document/ai/storage; records owned by models, aliased in services)
- [ ] 06 steps 5–6: bridge regroup + drop aliases
- [ ] 07 steps 1–3: `extract_cache` md/page-map columns, normalize, window fn
- [ ] 07 steps 4–6: `read_source` pages/offset/limit + envelope, `App.ReadSourcePage`, `path:page` search hits

## Phase 15 — Operations & growth (planned: 08)
- [ ] SKILL.md format + discovery (bundled/workspace/user) + starter set; prompt injection caps
- [ ] Workflows runner (gated routines reusing tool budget/audit)
- [ ] Parallel APIs (bounded fan-out, idempotent steps, thread/step usage attribution)
- [ ] Chat UI: per-turn tokens, `via provider/model`, tool-call trace, retry/switch action
- [ ] Logs DB (`ai_log`, `tool_audit`, `guidance_notes`) + retention + Settings viewer/export + self-guidance loop
- [ ] CI (vet/test/tsc/build) + vitest for pure TS + learning-tests convention
- [ ] Releases: tags + CHANGELOG + NSIS directory page (install-drive choice) + portable zip + smoke checklist

## Phase 16 — Unified chat + live library + empty docs (09 + 10 §1; landed)

- [x] One chat per workspace (`ensureThread` matches workspace only) + New chat button
- [x] Thread persistence (save/append best-effort) + history load on folder switch + switcher UI
- [x] Thinking trace (timed dispatches → payload → SQLite column → expandable chat UI)
- [x] Make-summary designation (kind flip + primary persist) + no-summary guidance toast
- [x] Tool budgets raised (4 rounds, 8 calls/turn, 4/round)
- [x] Empty files open as blank pages (`extract.emptyDocument`, tested)
- [x] Library refresh on window focus + open focuses existing tab (path+root match)
- [x] Skills loader + `summarize-source` starter (`backend/skills/`)

## Phase 17 — Intermediates + canvas + edit loop (10 §§2–3 + 11 §3; landed)

- [x] `backend/.work/` temp mirrors + mirror-aware read_summary/propose + mtime re-sync accept path
- [x] Real `extract.NormalizeBlocksJSON` (+ page map) backing mirrors
- [x] Page canvas renderer for docx/md/txt (`paginateBlocks`; pdf stays pdf.js); empty doc = one empty page
- [x] Revise-with-context loop (`focusedChange` + summary snapshot both ends; shared `focusedProposalBlock`)
- [ ] Custom-instruction revise input in Review UI
- [ ] Accept notification to the thread (system note)

## Phase 18 — Skills wiring + workflow runner (11 §§1–2; landed)

- [x] Prompt wiring (≤2 skill excerpts at prompt build; byte-identical with no skills) + `App.MatchSkills`
- [x] `backend/workflows/` runner + `run_workflow` tool (read-only v1) + schema parity
- [x] `add-to-summary` workflow (deterministic read→propose loop) + multi-proposal surfacing per turn
- [x] Intent-phrase skill routing + skill v2 autonomous fast path
- [ ] Starters (revise-proposal, find-in-workspace, quota-aware) + workspace/user layers
- [ ] Propose-capable workflows + (thread, step) idempotency ledger

## Phase 19 — Frontend simplification + backend async APIs (11 §§4–5; partially landed)

- [x] Context-pack assembly behind `App.BuildContextPack` (bridge-first, local fallback; Go parity test)
- [x] Chat history behind `App.ListThreads/ListMessages` (were bound, never called)
- [ ] Skill matching UI (API landed) + structured usage meters
- [ ] Keep non-network packages stdlib-only (allowlist: sqlite, ledongthuc/pdf); dep check in CI spirit

## Phase 20 — Live publish + in-file diff review (landed)

- [x] `publish_summary` tool (mirror → blocks → mapped `.docx`, Resolve-jailed; honest errors)
- [x] `BlocksFromMarkdown` (stable IDs; page markers skipped) + mirror op pre-images + unpublished watermark
- [x] Auto-reload on publish payload (`doc.blocks.replace` + `doc.diffs.set`, editor-refresh semantics)
- [x] Auto-publish safety net (proposed-but-unpublished turns write through; traced + audited)
- [x] In-file diff decorations + Accept/Reject/instruction→AI (was Slate popover; now canvas menu)
- [x] Old approval path removed (Review face, `runRevise`, `change.revise`, InsertCard, pending decorations)
- [x] Slate removed (editor + adapter deleted; bundle −200 kB): `DocxCanvas` direct editing
- [x] Make-summary designation (was dead actions) + no-summary guidance + `summaryPath` wire
- [x] Skill v3 publish discipline ("write through, then report")
- [x] Save-to-same-file (open path passed; no more `summaries/` copies) + Wails menu suppression on pages
- [x] Plaintext canvas editing (rich paste as plain paragraphs) + commit paragraph-splitting
- [x] publishError payload → exact-fix toasts for unwritten turns
- [x] Basename path resolution into subfolders (ambiguous → suggest)
- [ ] Backspace-merge across blocks in canvas
- [ ] Backend-owned threads flip — EXPLICITLY DEFERRED (thread archive works; edit loop owns priority)
