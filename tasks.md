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
    mock factory (`frontend/src/mock/data.ts:660`); consumed by Show-in-library
    and Show-containing-folder without root validation.
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
  - Fallback branch: `ReaderMode.tsx:151` (`catch` → fall through),
    mock factory `ReaderMode.tsx:154`.
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
   the `Documents` service boundary (`backend/services/documents.go:29`).
6. Inventory (exists vs unbuilt):
   - Block/segment model: EXISTS (mock-authored today — `mock/data.ts:500`
     hand-written documents; real `.txt` arm emits conforming JSON via
     `openTextFile`, `backend/services/documents.go:86`).
    - Slate integration: EXISTS (`frontend/src/features/reader/slateAdapter.ts`
      `toSlate`/`fromSlate`, `SummaryDocumentView.tsx` Slate editor; canonical
      blocks stay the source of truth, decorations are transient).
    - Extraction to block/segment JSON: EXISTS for `.txt`/`.md` (real,
      `os` Stat-gated + bounded reads, `backend/services/documents.go`) and
      `.docx` (stdlib ZIP+OOXML, `backend/services/docx.go`). Block IDs are
      content-addressed and stable (`backend/services/blockids.go`); successful
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
`backend/services/documents.go:20` and needs no evaluation); cloud/proprietary
extraction APIs (offline-first project).
