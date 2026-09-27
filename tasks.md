# tasks.md — Lumen folder-open feature: bugs, design constraints, extraction plan

Reader: a future model session with zero prior context.
Go module is `Conductino` (`go.mod:1`). Wails shell lives in `frontend/app.go`
(`package frontend`, bound as `window.go.frontend.App.*`); pure-Go bridge in
`backend/main.go` (`package backend`); services in `backend/services/`;
React UI in `frontend/src/`. All `file:line` references below were verified
against the tree at time of writing.

---

## Section 1: Known bugs (from current folder-open feature)

### 1.1 Stale tab / stale path resolution on folder switch

- **Problem:** `Filesystem.root` (`backend/services/filesystem.go:36`) and the
  `tree` state (`frontend/src/features/reader/ReaderMode.tsx:32`) both replace
  cleanly on re-pick, but already-open tabs retain `metadata.path` tokens
  resolved against the OLD root. `Filesystem.Resolve`
  (`backend/services/filesystem.go:76`) joins every relative token against the
  *current* root, so re-revealing, re-locating, or re-opening a stale tab
  silently targets `new-root/old-relative-path` — a wrong file or a confusing
  not-found that collapses into the mock fallback (see 1.2).
- **Why it matters:** The user's intended UX is opening DIFFERENT folders
  across sessions (per topic/project), not one fixed folder. This triggers in
  normal use on every folder change, not as an edge case. Silent misresolution
  (wrong file's contents under the old tab's title) is worse than an error.
- **Current code:**
  - Root overwrite, no tab handling: `SetRoot`
    (`backend/services/filesystem.go:45`), `SelectFolder`
    (`frontend/app.go:139`), `pickFolder`
    (`frontend/src/features/reader/ReaderMode.tsx:61`).
  - Stale carriers: `metadata.path` (`frontend/src/types/domain.ts:203`), set
    at open time (`frontend/src/features/reader/ReaderMode.tsx:139`) and from
    Send-to-Reader (`makeDocumentFromSource` in `AIBrowsePanel.tsx`); consumed
    by Show-in-library and Show-containing-folder without root validation.
- **Suggested fix — two candidates:**
  - (a) *Invalidate on switch:* on `SelectFolder` success, close (or mark
    stale) every tab whose document has a `metadata.path`. Simple, no model
    change; cost is UX data loss — unsaved summary edits in open tabs need a
    guard, otherwise switching folders silently discards work. Chosen if tabs
    are cheap/disposable.
  - (b) *Tag with opening root:* add `rootPath: string` to `DocumentMetadata`
    (`frontend/src/types/domain.ts:196`) at open time (Go knows the absolute
    root; return it from `OpenFile` or expose `LibraryRoot()`), and compare
    against the current root before any resolve/reveal/locate. Mismatch →
    explicit "file belongs to another folder" state, not a silent resolve.
    More code, preserves tabs; preferred if open-tab state (edits, highlights,
    scroll) is valuable — which `ReaderMode.tsx:1` states is a requirement.
- **Status:** not started.

### 1.2 Silent failure collapsing to mock fallback

- **Problem:** Every `OpenFile` failure reason — missing file, permission
  denied, oversized file (`documents.go:91`), unsupported type
  (`documents.go:64`), containment rejection (`filesystem.go:88`) — becomes a
  rejected Promise, caught in `WailsFilesystem.openFile`
  (`frontend/src/services/backend.ts:186`) → `null`, then falls into the same
  mock-factory branch as genuine mock mode, with the identical toast
  `` `Opened ${node.label} (mock extraction)` ``
  (`frontend/src/features/reader/ReaderMode.tsx:172`). A read failure is
  indistinguishable from success.
- **Why it matters:** Once real PDF/DOCX parsers land, a parser failure
  (corrupt file, malformed DOCX, encrypted PDF) will take this same path and
  present as "expected placeholder", masking real bugs behind mock UI. The
  fallback also fabricates a `Source`/`Document` that pollutes `AppState`
  (`frontend/src/types/domain.ts:332` flat `documents` map) with fake content
  attributed to a real filename.
- **Current code:**
  - Collapse point: `catch { return null; }`
    (`frontend/src/services/backend.ts:189`).
  - Fallback branch: REMOVED — `ReaderMode.openFile` reports honest errors
    (toast + typed `reason`) with no fabricated document; `src/mock/` deleted.
  - Failure producers: `ErrUnsupportedType` (`documents.go:43`), size cap
    (`documents.go:81`), `os.ReadFile` errors (`documents.go:87`), `Resolve`
    rejection (`filesystem.go:87`).
- **Suggested fix:** Return a typed failure reason from Go — either a
  `reason` field on `OpenedDocument` (`unsupported | not_found |
  permission_denied | too_large | parse_error`) with `BlocksJSON` empty, or a
  small `OpenError{reason, detail}` struct. TS mock-fallbacks ONLY on
  `unsupported` (and only when the type is genuinely unimplemented); every
  other reason renders a real error toast naming the file and reason, and
  opens nothing. Approach chosen over boolean success/failure because the
  fallback-vs-error decision needs the reason, and string-matching Go error
  text from TS is brittle across Wails serialization.
- **Status:** not started.

---

## Section 2: Architecture — multi-folder / multi-session source→summary tracking

Design constraint. Not implemented. To be respected by any code written from
now on, even before it is built.

- The user will have multiple folders/sessions open over time (per topic or
  project), each containing its own source documents.
- Each folder/session has (or will have) its own summary document that source
  content gets extracted/summarized INTO.
- Required relationship is many-to-one: which source belongs to which
  session, and within a session, which summary document is the current target
  for "include in summary" actions.
- Current code violates this by structure: `runIncludeInSummary`
  (`frontend/src/state/aiController.ts:122`) picks its target with
  `Object.values(state.documents).find((d) => d.kind === "summary")` —
  first-summary-wins, no session scoping, no chooser. The moment two
  session/summary pairs coexist in the flat `documents` map
  (`frontend/src/types/domain.ts:332`), inclusions land in an arbitrary
  summary. Same flat-map hazard applies to `Source` records
  (`frontend/src/types/domain.ts:69`) and `DocumentChange.documentId`
  references.
- Recorded as a known scaling limitation to fix BEFORE multi-session support
  is added: retrofitting session-scoping onto ambient global state is more
  invasive than designing it in. Until then, do not add new features that
  assume "the" summary document or "the" workspace folder.
- Suggested (not implemented): a `sessionId: ID` field on `Document`
  (`frontend/src/types/domain.ts:219`), `Source`, and `DocumentChange`, plus
  session-scoped selectors replacing the flat `find()` / `Object.values()`
  scans. `Document.kind` (`"source" | "summary"`, `domain.ts:221`) stays as
  the editable/read-only discriminator; `sessionId` adds the ownership axis.
- **Status:** not started (design note only).

---

## Section 3: Extraction pipeline — target design

Contract for all future extraction work. Steps 1–5 are the pipeline; step 6
is the as-is inventory.

1. Source file (PDF/DOCX/TXT/etc.) → backend extractor (library per format,
   see Section 4) → output JSON targets the EXISTING `DocumentBlock`/`Segment`
   shape (`frontend/src/types/domain.ts:174`, `DocumentBlock` at `:184`,
   `Segment` at `:177`) directly. No second intermediate format: block/segment
   is already the canonical document representation and already flows through
   reader render (`DocumentView.tsx`), AI selection refs (`aiController.ts`),
   and the change-proposal pipeline (`DocumentChange`,
   `frontend/src/types/domain.ts:248`). Mirrored Go-side decode target is
   `textBlock`/`textSegment` (`backend/services/documents.go:71`); generalize,
   don't duplicate.
2. Block/segment JSON is held in frontend state only (React `AppState`,
   `frontend/src/types/domain.ts:327`) until explicitly saved. No disk writes
   on open/extract.
3. READ-ONLY source documents (`Document.kind === "source"`,
   `frontend/src/types/domain.ts:221`): render blocks directly — current
   `SourceDocumentView` pattern (`frontend/src/features/reader/DocumentView.tsx`)
   unchanged.
4. EDITABLE summary documents (`kind === "summary"`): block/segment JSON is
   converted via a thin adapter at the editor boundary ONLY into Slate.js
   node/leaf values — block → Slate Element, segment → Slate Text with marks
   (`em`/`strong` map to Slate marks; `highlightId` to a custom property).
   Block/segment stays canonical and persisted; Slate's value is a transient
   view, never the source of truth. This preserves the swap-editor-later
   property (storage format independent of the editor library).
5. On save: Slate value → block/segment JSON (inverse of step 4) → a second
  backend library writes the target file type on disk (e.g. DOCX writer for
  summary documents). Read path and write path are separate libraries behind
  the `Documents` service boundary (`backend/extract/extract.go`).
6. Inventory (exists vs unbuilt):
   - Block/segment model: EXISTS (real-authored — extraction arms and the
     Send-to-Reader factory; no hand-written documents remain).
    - Slate integration: EXISTS (`frontend/src/features/reader/slateAdapter.ts`
      `toSlate`/`fromSlate`, `SummaryDocumentView.tsx` Slate editor; canonical
      blocks stay the source of truth, decorations are transient).
    - Extraction to block/segment JSON: EXISTS for `.txt`/`.md` (real,
      `os` Stat-gated + bounded reads, `backend/extract/extract.go`) and
      `.docx` (stdlib ZIP+OOXML, `backend/extract/docx.go`). Block IDs are
      content-addressed and stable (`backend/extract/ids.go`); successful
      extractions are cached in SQLite (`extract_cache`, root-anchored path +
      mtime + size, 200-entry LRU). PDF / HTML extraction: UNIMPLEMENTED
      (`default:` arm returns `ErrUnsupportedType` with a typed reason).
   - DOCX/PDF save-out: DOES NOT EXIST in any form.

---

## Section 4: Extraction library options (Go; cgo noted where relevant)

Target machine is low-spec (4 GB RAM, Celeron): binary size, build time, and
per-file memory matter. Pure-Go preferred over cgo where quality is
comparable — cgo breaks Wails' straightforward Windows cross-compilation and
adds system-library deployment burden. OCR is OUT OF SCOPE: extraction target
is text-layer PDFs only; scanned/image-only PDFs are a documented limitation,
not a library-selection criterion. No OCR libraries evaluated.

### PDF (text-layer extraction)

- **pdfcpu** (`github.com/pdfcpu/pdfcpu`) — pure Go, Apache-2.0, actively
  maintained (pushes through mid-2026, large contributor base). API:
  `api.ExtractText` over `io.Reader`; pages joined with `\f` separators, not
  `\n`. Known limitations: text quality is raw content-stream order (no
  layout/reading-order inference); CJK text requires an external
  `fontmap.yml`; owner-password-locked files are unextractable (no
  open-source Go library bypasses AES-256 permission bits). Low-spec note:
  documented as memory-efficient with minimal dependencies; large module
  surface but no runtime heaviness reported. Fits the pipeline with a
  page-split + paragraph-heuristic layer on top.
- **ledongthuc/pdf** — pure Go, BSD, zero dependencies, tiny. Continuation of
  Russ Cox's archived `rsc.io/pdf`. Text + basic formatting only, no
  positions, no tables. Adequate for simple single-column text-layer PDFs;
  insufficient where heading/paragraph structure must be recovered. Cheapest
  binary/build cost of the options.
- **unidoc/unipdf** — pure Go, highest extraction fidelity (positions, fonts,
  tables, `ExtractionModeLayout` vs `Plain`), 10+ years production use.
  Licensing is the blocker: commercial / AGPL-family terms, and the free tier
  reportedly caps at 200 pages/day with silent empty returns past the limit —
  a failure mode indistinguishable from "empty PDF" without careful handling.
  Low-spec note: heavy binary (~40 MB+ of rendering/image modules) and slow
  builds; overkill for text-only extraction on a Celeron. Evaluate only if a
  commercial license is acceptable AND positioned extraction becomes required.
- **razvandimescu/gopdf** — pure Go, MIT, zero dependencies, positioned
  extraction + table detection in one package. Newer project, essentially
  single-maintainer; the detailed comparison table circulating for it is
  self-authored marketing — verify extraction quality on the project's own
  sample PDFs before adopting. Candidate for evaluation, not default.
- **cgo bindings (go-pdfium, MuPDF bindings)** — full-fidelity rendering +
  extraction via battle-tested C engines, but cgo + system libraries +
  (MuPDF) AGPL licensing. Rejected direction for this project: the Wails
  Windows build and low-spec distribution story both degrade. Revisit only if
  pure-Go extraction proves structurally inadequate.

### DOCX (read; write capability noted for Section 3 step 5)

- **unidoc/unioffice** — pure Go, actively maintained (v2.10.0, April 2026),
  read/write/edit DOCX with paragraph/run/table access and an `X()` escape
  hatch to raw OOXML schema. Licensing requires a check before adoption
  (dual open-source/commercial lineage; `gooxml` packages carry AGPL-3.0).
  Low-spec note: ~33 MB binary overhead from generated OOXML structs (size,
  not runtime RAM). Strong fit: paragraph+run model maps directly onto
  block/segment, and the SAME library covers the Section 3 step-5 DOCX
  writer — one dependency for both directions.
- **baliance/gooxml** (original) — archived/unmaintained. Reference only; do
  not adopt. Community forks (`luckymark84/gooxml`, `ygpkg/gooxml`) inherit
  AGPL-3.0 with uncertain maintenance — evaluate-only, behind unioffice.
- **ieshan/go-ooxml** — pure Go, LGPL-2.1, zero dependencies, security-first
  (ZIP-bomb/XXE/path-traversal limits), text+Markdown extraction, and a
  ProseMirror/TipTap JSON bridge directly relevant to the Section 3 step-4
  editor adapter. Counterweight: brand-new (April 2026), near-zero adoption,
  requires Go 1.26. Evaluate-only; promising second source.
- **mmonterroca/docxgo v2** — pure Go, actively released (v2.12, August
  2026), read-modify-save round-trip with style preservation. Broader scope
  than needed (themes, mail-merge, RPC/Node wrappers). Evaluate-only.
- **stdlib fallback** — DOCX is ZIP+XML: `archive/zip` + `encoding/xml` over
  `word/document.xml` yields paragraph/run text with zero dependencies.
  Loses tables/images/numbering fidelity; viable as a stopgap text-only arm
  (same role `.txt` plays today) while a full library is evaluated.

### Not evaluated (out of scope by decision above)

OCR engines and OCR-bound PDF pipelines (scanned PDFs); HTML extractors
(`golang.org/x/net/html` is already the documented candidate in
`backend/extract/extract.go` and needs no evaluation); cloud/proprietary
extraction APIs (offline-first project).

---

## Section 5: Unified review — DocumentChange through the SummaryDiff path

Scope constraint for all of Section 5: no Slate/ProseMirror work. `slate`,
`slate-react`, `slate-history` (`frontend/package.json:11-19`) are dead
dependencies with zero code imports; the canvas is a custom
`contentEditable="plaintext-only"` editor over canonical `DocumentBlock[]`
(`frontend/src/features/reader/SummaryDocumentView.tsx:151-222`,
`frontend/src/types/domain.ts:304-324`). All designs below work within
that architecture. Nothing here builds a third review system.

### 5.1 Render pending DocumentChange with the existing diff mark system (pre-publish)

- **Problem:** System A (`DocumentChange`, pending/accepted/rejected,
  block-anchored — `frontend/src/types/domain.ts:398-424`,
  `backend/models/storage.go:47-61`) has zero canvas highlighting; the
  only pending indicator is a tab badge (`ReaderTabs.tsx:50,64-68`,
  `selectors.ts:25-29`). System B (`SummaryDiff`, ephemeral post-publish
  — `domain.ts:380-392`) has the full highlight + menu path
  (`DIFF_MARK_CLASS` at `SummaryDocumentView.tsx:82-86`, `MarkedText`
  `:88-149`, `diffByBlock` `:286-295`). Pre-publish proposals are
  therefore invisible in the document they target.
- **Why it matters:** The user reviews exactly where the edit lands.
  Without in-canvas marks, pending inserts (which already exist as blocks
  in `doc.blocks`, pre-created at propose time —
  `aiController.ts:586-612`, `reducePart3.ts:61-76`) read as ordinary
  text, and pending modify/delete (which keep original block content —
  `reducePart3.ts:78-102`) show nothing at all. Every highlighting
  feature added to diffs must otherwise be re-built for cards.
- **Current code:**
  - Diff render assumes publish: `MarkedText` takes
    `diff: SummaryDiff | undefined` (`:88-96`); `diffRuns(segments, span)`
    (`:51-80`) probes `segments`-joined text for `diff.text` (first 120
    chars) and falls back to plain `<span>`s when unmatched (`:108-119`);
    `diffByBlock` (`:286-295`) fuzzy-matches `d.text` against
    `blockPlainText(b)` and skips `delete`/empty; `EditableBlock` threads
    one `diff` per block (`:151-164,:487-496`); list rows match one row
    via `diffRow` (`:241-260`). All call sites type the menu payload as
    `SummaryDiff` (`openMenu :297-304`, menu state `:281`).
  - Cards carry exact anchors diffs lack: `DocumentChange.blockId` is the
    true target (`domain.ts:409-424`); `change.propose` for inserts
    already splices the block into `doc.blocks` with `changeId` set
    (`reducePart3.ts:61-76`); autosave deliberately filters pending
    inserts out of the `.docx` snapshot
    (`SummaryDocumentView.tsx:610-614`). The fuzzy probe is unnecessary
    for cards — and actively wrong for pending modify, where the block
    still holds the OLD text so a `newContent` probe never matches and
    `diffRuns` returns null.
- **Proposed design:** Adapter, not a new renderer. Introduce a
  view-model at the top of `SummaryDocumentView.tsx` (same file, no new
  system):
  - `type ReviewMark = { kind: "change" | "diff"; id: ID; op: "insert" |
    "modify" | "delete"; blockId: ID; text: string; oldText?: string;
    note?: string }` plus `changeToMark(c: DocumentChange):
    ReviewMark` (`text = c.newContent`, `oldText = c.oldContent`,
    `blockId = c.blockId`). `SummaryDiff` converts 1:1 (blockId resolved
    by the existing probe once, at conversion time, not per render).
  - `markByBlock: Map<ID, ReviewMark>` = pending changes by exact
    `blockId` (filter `status === "pending"`, `selectors.ts`
    `pendingChangesFor` is the source) overlaid with `diffByBlock` for
    diffs; on collision (same block has both) the diff wins and the
    change row is still reachable from chat (documented, not silent).
  - Widen `MarkedText`/`EditableBlock`/`openMenu`/menu state from
    `SummaryDiff` to `ReviewMark` (field-identical for render:
    `op/text/oldText/note` + `id/blockId/kind`). `DIFF_MARK_CLASS`
    (`:82-86`) is reused unchanged — `Record<ReviewMark["op"], string>`.
    Probe rule per kind: `diff` keeps `diffRuns(segments, mark.text)`;
    `change` with `op === "insert"` marks the whole block (the block IS
    the proposal); `change` modify/delete probes
    `highlightFragment || oldContent` against current segments (which
    still hold the original), so the mark lands on the span about to
    change. Deletes with no span keep the existing delete-card pattern
    (`:659-699`) generalized to both kinds.
  - Title string (`:135`) changes from `"Published AI edit — …"` to
    kind-aware (`change` → `"Proposed AI edit — …"`, `diff` keeps
    `"Published …"`); `data-diff-id` becomes `data-review-id` with the
    mark id (keep `data-diff-id` as an alias one release if any test
    queries it — grep shows none today).
  - Explicit non-goals: no change to `DocumentChange`/`SummaryDiff`
    persisted shapes, no Slate, no new highlight palette.
- **Status:** not started.

### 5.2 One context menu for both kinds (shared trigger, diverging handlers)

- **Problem:** The working right-click menu
  (`SummaryDocumentView.tsx:504-570`, trigger `:120-141` + `:297-304`,
  suppression of the native Wails menu `:482-485`) only fires on
  `SummaryDiff` marks. Pending cards have no menu anywhere; their
  accept/reject lives wherever `change.decide` is dispatched from chat
  (`aiController.ts:581-584` supersede path; no canvas entry point).
- **Why it matters:** Two gestures for the same three verbs
  (Accept/Reject/Send-to-AI) guarantees one path rots. The
  diff-vs-untouched discrimination already built (`MarkedText` plain
  `<span>`s for untouched `:97-118`, `preventDefault()`-only on pages
  `:463,:485`) must extend to cards without a second menu component.
- **Current code:**
  - Trigger is typed `onOpenMenu(e, diff: SummaryDiff)` end-to-end
    (`MarkedText :95`, `EditableBlock :164`, `DocxCanvas.openMenu
    :297`). Menu state is `{ diff: SummaryDiff; x; y }` (`:281`).
  - Handlers diverge by necessity, not accident: `acceptDiff` (`:399-404`)
    only dispatches `doc.diffs.dismiss` (the `.docx` already carries the
    edit); `rejectDiff` (`:406-441`) does an inverse span replace (or
    re-append for deletes) then dismisses, relying on autosave to write
    the `.docx`. Card semantics (`reducePart3.ts:78-102`,
    `actions.ts:59-60`) are `change.decide(id, accepted|rejected)`:
    accept-modify rewrites segments to `newContent`, accept-delete /
    reject-insert removes the block, otherwise blocks untouched, status
    flips either way.
  - Revise converges already: `sendDiffToAI(diff, instruction)` (`:443-453`)
    calls `runChat(instruction, { documentId, selection: { blockId,
    text: span } })`; the card path needs the same call plus `changeId`
    so the `focusedChange` heuristic (`aiController.ts:425-441`,
    `prompts.go:296-314`) binds the turn to the right proposal.
- **Proposed design:** Same menu component, kind-dispatched handlers:
  - Change menu state to `{ mark: ReviewMark; x; y }` and `onOpenMenu(e,
    mark: ReviewMark)`. Header reads `Published {op}` for `kind ===
    "diff"`, `Proposed {op}` for `kind === "change"` (`:519-528`).
  - `acceptMark(mark)`: `diff` → existing `acceptDiff` body unchanged;
    `change` → `dispatch({ type: "change.decide", id: mark.id, status:
    "accepted" })`. `rejectMark(mark)`: `diff` → existing `rejectDiff`
    body unchanged; `change` → `dispatch({ type: "change.decide", id:
    mark.id, status: "rejected" })`. Toasts stay kind-specific (`"Diff
    accepted"` vs `"Change accepted — …"`), so the audit trail
    distinguishes write-through dismiss from block mutation.
  - `sendMarkToAI(mark, draft)`: both kinds call `runChat(instruction ||
    fallback, { documentId, selection: { documentId, blockId:
    mark.blockId, text: span.slice(0, 2000) }, changeId: mark.kind ===
    "change" ? mark.id : undefined })`. `span` for diffs is the existing
    rule (`:444`: delete → `oldText ?? text`, else `text`); for changes
    it is `newContent` (insert/modify) or `oldContent` (delete). No new
    AI plumbing — this is the existing `AIRequest.selection` +
    `changeId` → `focusedChange` contract (`domain.ts:164,197-210`,
    `aiController.ts:486-494`).
  - Delete-cards (`:661-699`) render from a unified `pendingDeletes =
    diffDeletes + changeDeletes` list with the same kind dispatch; insert
   /modify stay inline per §5.1.
- **Status:** not started (blocked on §5.1 view-model; the handler split
  itself is ~30 lines).

### 5.3 Targeted revise with full-document context (existing contract, one wiring gap)

- **Problem:** Revise must scope the EDIT to one span while the MODEL
  still sees the whole summary (harmonization baseline, not append-only
  — `prompts.go:188-206`). The pieces exist but are mis-wired for the
  diff path.
- **Why it matters:** Without full context the model rephrases in a
  vacuum (repeats claims, breaks terminology, contradicts neighboring
  blocks). Without a tight scope it rewrites the whole summary when
  asked to fix one sentence. Both failure modes are already documented
  in the prompt comments (`prompts.go:181-186`, `contextPack.ts:82-86`).
- **Current code:**
  - Full context: `buildContextPack(doc, selection?, opts?)`
    (`contextPack.ts:71-110`: selection core + ±2-block window via
    `localWindow` + heading outline, 6000-char budget) with a Go mirror
    (`backend/extract/context.go:43-81`) via `buildContextPackAsync`
    (`contextPack.ts:117-140`); backend appends pack + up-to-6000-char
    summary snapshot (`prompts.go:85-228`, truncation `prompts.go:112-113`).
  - Tight scope: `### Anchored region` (`prompts.go:212-219`) carries
    `selection.text + blockId`; `### Focused pending proposal`
    (`prompts.go:296-314`) binds revise-intent wording to one change and
    orders the model to `emit propose_summary_edit with op="modify" (or
    "delete") targeting that block and the FULL revised text` — the exact
    contract §C asks for, already shipped. Frontend resolves the focus
    (`aiController.ts:425-441`: explicit `changeId` → selection-overlap →
    revise-intent regex fallback).
  - The gap: `sendDiffToAI` (`SummaryDocumentView.tsx:443-453`) builds a
    `selection` but `runChat` drops it before context assembly —
    `buildContextPackAsync(doc, undefined)` (`aiController.ts:460-463`)
    hard-codes `undefined`, so the ±2-block window and range offsets
    (`contextPack.ts:46-59,82-86`) never fire on the revise path; the
    model gets title + outline + summary snapshot but no local window.
    Diffs also pass no `changeId`/`targetBlockId`, so a revise of a
    published span relies on fuzzy re-match (`findBlockForSpan`,
    `aiController.ts:548-554`) instead of the exact block already known
    to the canvas (`diffByBlock` / `markByBlock`).
- **Proposed design (no new mechanism):**
  - Thread the known anchor through: `sendMarkToAI` (§5.2) passes its
    `selection` (with `blockId` and `range` when the mark probe yields
    offsets) into `runChat`, and `runChat` forwards `sel` to
    `buildContextPackAsync(doc, sel)` instead of `undefined`
    (one-argument fix at `aiController.ts:460-463`; the `ContextPackOpts`
    and Go signature already accept block+range —
    `contextPack.ts:117-140`, `backend.ts:346-373`). Range offsets ride
    as `(block …, chars s–e)` (`contextPack.ts:85`) and as the
    `(block …)` suffix on the anchored region (`prompts.go:215-217`).
  - For `change` marks pass `changeId` (→ `focusedChange` → the
    `op="modify"` revise contract, `prompts.go:311`). For `diff` marks
    pass the resolved `blockId` in `selection` (the model already scopes
    edits to the anchored region per `prompts.go:213`); optionally also
    synthesize a transient `focusedChange`-shaped hint from the diff
    (`{ id: diff.id, op, blockId, oldContent, newContent: text }`) only
    if prompt logs show the model ignoring bare anchors — no backend
    change required either way since `FocusedChange` is already optional
    (`models/ai.go:63-65`).
  - Success criterion: a "shorten this passage" right-click on one marked
    span returns a single `propose_summary_edit op="modify"` against that
    block (visible as one superseding pending change per
    `aiController.ts:577-585`), with the rest of the summary untouched.
- **Status:** implemented (frontend-only). `runChat` preserves
  `selection.range` from live selection (`aiController.ts:418-433`) and
  forwards the anchored `sel` into `buildContextPackAsync(doc, sel)`
  (`aiController.ts:460-476`) instead of `undefined`, so the ±2-block
  window + range offsets fire on the diff-revise path. In-file diff
  hardening landed with it: shared `diffProbes`/`findBlockIndexForDiff`
  (`SummaryDocumentView.tsx:51-80`) now back `diffByBlock`, `rejectDiff`,
  `sendDiffToAI`, and list-row matching; `diffRuns` prefers the full span
  (long inserts mark wholly); insert-reject on an emptied block removes
  the block. Verified: `tsc --noEmit -p frontend` clean, `go build
  ./...` + `go test ./backend/...` all ok. Card unification (§5.1/§5.2)
  deferred per scope decision — cards untouched.

### 5.4 Two review models — debt entry and recommendation

- **Problem:** Two review models with different accept/reject semantics
  coexist: cards mutate `Document.blocks` in place via `change.decide`
  (`reducePart3.ts:78-102`: accept-modify rewrites segments,
  accept-delete/reject-insert filters the block, status flips always);
  diffs dismiss via `doc.diffs.dismiss` (`reducePart3.ts:43-47`) with
  reject implemented as an inverse text replace + autosave `.docx` write
  (`SummaryDocumentView.tsx:406-441`). Only one path can be "the" review
  at a time by construction: when a turn publishes, `reviewInFile`
  suppresses card materialization entirely (`aiController.ts:532-538`)
  with the comment "materializing parallel card proposals would fork
  reality."
- **Why it matters:** Any review feature (highlighting, menu, revise,
  counts, persistence) built on one model silently misses the other —
  §5.1/5.2 are instances, not exceptions. The header comment in
  `SummaryDocumentView.tsx:578-580` ("Single review path: no proposal
  cards") already disagrees with `aiController.ts:522-536` (cards kept
  for the unpublishable path), so a cold reader gets contradictory
  guidance. Cards also persist (`ChangeRecord`,
  `services/storage.go:37-38,208-225`) while diffs are ephemeral
  (`Document.diffs`, `domain.ts:372-377`, "never persisted"), so crash
  recovery differs by path for no user-visible reason.
- **Current code:** Card lifecycle `change.propose` → `change.decide`
  (`actions.ts:59-60`, `reducePart3.ts:61-102`,
  `aiController.ts:542-613`); diff lifecycle `doc.diffs.set` →
  `doc.diffs.dismiss` + inverse edit (`aiController.ts:360-392` reload,
  `SummaryDocumentView.tsx:399-441,661-699`); single-path guard
  `reviewInFile` (`aiController.ts:537-538,547,615-616`); persistence
  split (`storage.go` vs in-memory `diffs`); badge counts only cards
  (`selectors.ts:25-29`, `ReaderTabs.tsx:50`).
- **Proposed fix — option (b): keep both models, share the
  rendering/interaction layer (§5.1 + §5.2), defer semantic merge.**
  Reasoning: (a) migrate-cards-onto-diff/mirror fails on the case cards
  exist for — no mappable file path (`publishError`, no `summaryPath`,
  browser mode). Diffs require a write-through `.docx`
  (`tools.go:588-641`, `CanPublish :569-581`); forcing every proposal
  through publish to get highlighting destroys the offline/pathless flow
  the card path deliberately preserves (`aiController.ts:539-540,617-635`
  loud-toast handling). A shared `ReviewMark` view-model kills the
  actual bug class (UI features landing on one path) at ~100 lines in
  one file, is fully reversible, and leaves the harder semantic merge
  (persistent vs ephemeral, mutate vs inverse-replace) as an explicit
  follow-up once pathless summaries either gain temp-file publish or are
  formally scoped out. Do not pursue (c) a third unified store until
  that scoping decision is made — it repeats the fork-reality hazard the
  `reviewInFile` guard was built to prevent.
- **Status:** not started; §5.1–5.3 are the implementation of this
  recommendation.

---

## Section 6: Multi-provider backend — OpenRouter gap, error hygiene, auth split

Context: `backend/ai/` has two structs for three providers — `GeminiService`
(`service.go:63`, native REST) and `OpenAICompatBackend`
(`openai_compat.go:15`, shared by Groq + OpenRouter). Contract is
`ModelBackend` (`backend.go:15-19`). Chain logic is `generateWithFailover`
(`service.go:224-357`); error taxonomy lives in `backend/usage/usage.go`
and is aliased into `ai` (`backend.go:65-78`).

### 6.1 OpenRouter key loads in code but not at runtime — diagnose first

- **Problem:** A `groq: 429 …; gemini: 429 …` combined error contains no
  `openrouter` segment, which `trySecondary` (`service.go:253-279`)
  produces only when `OPENROUTER_API_KEY` failed to load
  (`Configured()==false` → skipped at `:256-258`). The implementation is
  complete and registered (`NewOpenRouterBackend`,
  `openai_compat.go:47-59`; `loadSecondaryBackends`, `service.go:127-129`)
  — so the fault is in key LOADING, not provider code, and must be
  confirmed rather than guessed.
- **Why it matters:** Any chain/provider fix built on a wrong root cause
  (e.g. re-registering OpenRouter) changes nothing while the real cause
  persists. There are four live candidates and they need different fixes.
- **Current code / findings (verified, no guessing):**
  - `backend/.ai.env` is well-formed: every key (`AI_MODE`, `AI_PRIMARY`,
    `GEMINI_API_KEY`, `GROQ_API_KEY`, `GROQ_MODEL`, `OPENROUTER_API_KEY`,
    `OPENROUTER_MODEL`) parses with no quoting/spacing/inline-comment
    defects (shape-checked 2026-09-19; values never printed). The parser
    (`readKeyFile`, `service.go:387-410`, last-wins, strips ` #`
    comments/quotes) handles this file correctly.
  - No repo-root `.ai.env` exists; only `backend/.ai.env`. `loadEnvKey`
    (`backend.go:48-63`) resolves cwd-relative: process env → `./.ai.env`
    → `backend/.ai.env` → exe-dir `.ai.env`.
  - Keys load ONCE in `New()` (`service.go:89-102`, singleton via
    `main.go:37` → `services/ai_shim.go:17-21`); there is no reload. Four
    candidates remain: (a) app not restarted after the key was added to
    the file; (b) process launched with cwd ≠ repo root (desktop
    shortcut / Start-menu launch → cwd is exe dir or System32, both
    relative lookups miss) while GEMINI/GROQ arrived via process env;
    (c) a nearer `.ai.env` shadowing (repo-root file beats
    `backend/.ai.env` per-file, first file with the key wins);
    (d) process env holding a stale/empty-overriding value (empty falls
    through, but a stale non-empty value wins over the file).
- **Proposed design:** Permanent one-time startup diagnostic, not a
  throwaway: `keySource(name)` (`backend.go:69-87`) reports WHERE each
  key came from (process env / `.ai.env` / `backend/.ai.env` / exe-dir /
  missing — never the value), and `New()` logs one line with cwd +
  per-provider Configured + source (`service.go:103-124`). Next
  occurrence is diagnosed from the `wails dev` console in one look
  instead of a code read.
- **Status:** implemented. To diagnose the live failure: restart the app
  from the repo root, reproduce, and read the `[ai] config:` line —
  `openrouter=false(missing)` + unexpected `cwd=` confirms candidate
  (b); `openrouter=false(backend/.ai.env)` is impossible (file parses),
  so any `false` with correct cwd points at (a)/(c)/(d) via the source
  column.

### 6.2 Missing-key errors omit OPENROUTER_API_KEY

- **Problem:** All three "no key configured" errors named only
  `GEMINI_API_KEY or GROQ_API_KEY` (`service.go:311,356,424` pre-fix), so
  a misconfigured OpenRouter key was undiagnosable from the message.
- **Why it matters:** The error is the only signal a fresh install sees;
  omitting the third key sends the user down a two-provider checklist.
- **Current code:** Fixed at `service.go:347,397,465` — all three now
  read `add GEMINI_API_KEY, GROQ_API_KEY, or OPENROUTER_API_KEY to
  .ai.env / backend/.ai.env and restart` (the `Run` entry variant names
  `backend/.ai.env` specifically).
- **Proposed design:** Done — string change only, no behavior change.
- **Status:** implemented.

### 6.3 User-facing 429/5xx errors leak raw provider text

- **Problem:** The documented intent (`gemini.go:66-68`: "short, key-free
  errors safe to show in the UI") was contradicted on the failure paths:
  `gemini.go:107,119` and `openai_compat.go:303,309` interpolated raw
  provider bodies (quota numbers, upgrade links, org IDs, full JSON
  detail) into errors propagated verbatim via `service.go:465-466` →
  `AI unavailable: …` (`aiController.ts:663-667`).
- **Why it matters:** Quota internals and account-adjacent strings do not
  belong in the chat panel; truncation (`truncateRunes(_,200)`) limited
  length but not content.
- **Current code:** Category-preserving rewrite, landed:
  - 429 → `429 rate limit (<name>).` with NO detail suffix, full body to
    the server log (`log.Printf("[ai] <name> 429 detail: …", ≤500 runes)`)
    — `gemini.go:102-112`, `openai_compat.go:318-326`. The `429` prefix
    is load-bearing: `isModelGone` (`openai_compat.go:67-87`) and
    `IsRateLimitOrUnavailable` (`usage.go:45-61`) match on it, so the
    intra-backend model chain and single-mode failover gate behave
    exactly as before.
  - Non-429 provider error bodies → `AI unavailable now.` (gemini,
    `gemini.go:132-135`) / `<name> returned an error.` (compat,
    `openai_compat.go:345-349`), with two classifier-preserving
    exceptions checked BEFORE stripping: auth-shaped bodies → the §6.4
    invalid-key error; quota-shaped bodies on a 200-with-error envelope
    → `429 rate limit (<name>).` via `isRateLimitText`
    (`openai_compat.go:267-281,341-344`).
  - Server-side log is stdlib `log` (stderr, visible in `wails dev`
    console); there was no logger in the backend, so no new dependency.
- **Proposed design:** Done as above. Rule for future branches: never
  `fmt.Errorf("…: %s", rawBody)` toward the UI — `log.Printf` the body,
  return the category.
- **Status:** implemented. Regression guard: `usage/auth_test.go`
  (`TestAuthVsRateLimitSplit`) locks the short-form strings on both
  sides of the split; existing `env_test.go:37-57` (`isModelGone`
  long-form cases) still passes unchanged.

### 6.4 Bad-key errors fail over silently — now classified as auth, no failover

- **Problem:** Invalid-key errors fell through `IsRateLimitOrUnavailable`
  (which matched `"billing"`, `"permission denied"`, `"api key not
  valid"`) and triggered cross-backend failover — masking a deterministic
  config fault as a transient outage and burning the next provider's
  quota to launder the turn. No 401/403 branch existed anywhere
  (`postChat` had only 429 / ≥500 / generic).
- **Why it matters:** Failover exists for transient errors (429/5xx/
  timeout). An invalid key never heals by retrying elsewhere; hiding it
  delays the one fix that works (correct the key) and spends quota the
  user did not intend to spend.
- **Current code:** Split taxonomy + abort-on-auth, landed:
  - `usage.IsAuthError` / `IsAuthErrorText` (`usage.go:63-98`, needles:
    invalid/incorrect/wrong API key, unauthorized, forbidden,
    authentication, permission denied, billing, account deactivated —
    bare `401`/`403` digits deliberately excluded to avoid false
    positives); auth needles REMOVED from `IsRateLimitOrUnavailable`
    (`usage.go:45-61`). `ai` alias `isAuthError` (`backend.go:73-78`).
  - Constructors emit matchable short auth errors naming the exact var:
    compat 401/403 → `<name>: invalid API key — check <VAR>.`
    (`openai_compat.go:327-335`, via `keyEnvFor`, `backend.go:89-101`);
    auth-shaped bodies on any status → same shape (`:336-349`); Gemini
    400/401/403-or-auth-body → `gemini: invalid API key — check
    GEMINI_API_KEY.` (`gemini.go:113-127`). Intra-backend model chain
    already returns auth immediately (`isModelGone==false` for these —
    pinned by `env_test.go:48-57`).
  - Chain aborts on auth at every stage instead of aggregating:
    `trySecondary` loop (`service.go:290-298`), preferred-secondary
    result (`:313-320`), gemini-first result (`:354-361`), post-gemini
    result (`:334-340`). The user sees e.g. `AI unavailable: groq:
    invalid API key — check GROQ_API_KEY.` with no quota spent elsewhere.
  - Tradeoff, stated explicitly: if the preferred backend's key is bad
    but another backend is healthy, AI now stops instead of serving via
    the healthy one. Chosen because silent cross-quota laundering is the
    reported harm; the message names the exact fix. Revisit only with a
    "degraded but disclose" UI (e.g. serve via fallback AND toast the
    invalid key) — that needs frontend work, not a backend tweak.
- **Proposed design:** Done as above.
- **Status:** implemented + tested (`usage/auth_test.go`; `go build
  ./...`, `go vet`, `go test ./backend/ai/ ./backend/usage/` green).

### 6.5 Retry-with-backoff for 429 — evaluated, NOT implemented

- **Problem:** 429 handling is immediate-fallback-only at both layers
  (model chain `openai_compat.go:208-221`, backend chain
  `service.go:224-357`). No sleep, no `Retry-After` parse, no backoff —
  confirmed by grep (no `time.Sleep`/`After`/`Ticker`/`RetryAfter` in
  `backend/ai` or `backend/usage`).
- **Why it matters (evaluation):** Backoff-before-failover buys something
  only when the same request would succeed on retry against the SAME
  backend/model — i.e. a brief per-model bucket refill with no healthy
  alternative. Here there are three legs of fallback already: per-model
  fallbacks within a backend (`groqModelFallbacks`,
  `openRouterModelFallbacks`), then cross-backend failover. A sleep
  before trying a healthy alternative trades certain latency for a
  merely possible same-backend success, and free-tier 429s typically
  clear on minute/hour scales, not the 1–5s a user will tolerate
  inline. Honoring `Retry-After` additionally requires plumbing response
  headers through the `Generate(ctx,prompt,maxTokens)` signature (part
  of the `ModelBackend` interface, `backend.go:15-19`) — interface churn
  for a path that fires only when every alternative is also exhausted.
- **Current code:** No change made.
- **Proposed design (if ever):** Retry-with-backoff ONLY as a last resort
  — when the failing backend is the sole configured one (no fallback
  target exists). Shape: parse `Retry-After` (seconds or HTTP date,
  cap 30s) in `postChat`/`generate`, return a typed `retryAfterError`
  carrying the duration, and sleep-then-retry once in `Generate`
  (intra-backend) — never sleep when `trySecondary`/`tryGemini` has an
  untried backend. Do not add blind pre-failover sleep: it regresses
  every multi-provider 429 by the sleep duration for no modelable gain.
- **Status:** evaluated, not implemented — by reasoning above, not by
  deferral. Reopen if single-provider deployments become common.

### 6.6 Adding a fourth OpenAI-compatible provider — confirmed 3-line pattern

- **Problem:** Recurring question each time a provider is considered
  (Together, Fireworks, DeepInfra, Cerebras): what must change?
- **Why it matters:** Answering from architecture each time wastes a
  cycle; the pattern is fixed and should be recorded once.
- **Current code:** `OpenAICompatBackend` (`openai_compat.go:15-21`) is
  fully generic over `name/apiKey/base/model`; per-provider differences
  are constructor + fallback list + (rarely) headers. Registration is one
  line in `loadSecondaryBackends` (`service.go:122-131`, order =
  try-order). `dual` has no chain branch (accepted values only,
  `backend.go:23-31`) so no mode table needs updating.
- **Proposed design — minimal diff for a secondary-only provider
  (3 spots, same file pair):**
  ```go
  // 1. openai_compat.go — constructor + fallbacks (copy NewGroqBackend):
  func NewCerebrasBackend(apiKey string) *OpenAICompatBackend {
      model := strings.TrimSpace(loadEnvKey("CEREBRAS_MODEL"))
      if model == "" { model = "llama3.1-70b" } // pin known-good default
      return &OpenAICompatBackend{name: "cerebras", apiKey: strings.TrimSpace(apiKey),
          base: "https://api.cerebras.ai/v1", model: model,
          client: &http.Client{Timeout: 90 * time.Second}}
  }
  var cerebrasModelFallbacks = []string{"llama3.1-70b", "llama3.1-8b"}
  // 2. openai_compat.go Generate fallback pick (:198-202) — extend the
  //    if/else to select cerebrasModelFallbacks for name=="cerebras".
  // 3. service.go loadSecondaryBackends — one line:
      if k := loadEnvKey("CEREBRAS_API_KEY"); k != "" { out = append(out, NewCerebrasBackend(k)) }
  ```
  Extra ONLY if the provider must be primary-eligible: add the name to
  `loadAIPrimary` (`backend.go:35-43`) AND to `preferSecondary`
  (`service.go:241`), plus a `keyEnvFor` case (`backend.go:89-101`) so
  auth errors name the right var. Auth/429/5xx handling, tool schemas,
  `ProviderName()`, and the startup diagnostic pick the new backend up
  with zero further changes (all key off `Name()`/`Configured()`).
  Constraint: OpenAI-compatible `/chat/completions` + `Authorization:
  Bearer` + (optional) function-calling — a non-compatible wire shape
  needs a new `ModelBackend` impl, not a constructor.
- **Status:** documented, no code change (no fourth provider requested).
