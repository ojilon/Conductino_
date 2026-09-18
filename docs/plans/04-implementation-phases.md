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
- [ ] 06 steps 1–3: carve `extract/` → `tools/` → `usage/` (aliases, then delete)
- [ ] 06 steps 4–6: split `models/`, bridge regroup, drop aliases
- [ ] 07 steps 1–3: `extract_cache` md/page-map columns, normalize, window fn
- [ ] 07 steps 4–6: `read_source` pages/offset/limit + envelope, `App.ReadSourcePage`, `path:page` search hits
