# Lumen — Document rendering

> **Where do I plug in the PDF renderer / DOCX parser?** → replace the
> block rendering inside the two reader view components, and feed the
> `DocumentModel` from the extraction pipeline. App state, AI, panels and
> tabs do not change.

## The contract: DocumentModel

Renderers consume `Document` (see `src/types/domain.ts`):

```
Document → blocks[] → { heading | paragraph | list | page }
                            → segments[] → { text, em?, strong?, highlightId? }
```

Plus `metadata` (title/author/venue/format/pageCount/path), `highlights`
(block-anchored) and, for summaries, `sourceIds`.

Everything that is NOT pixels — selection state, AI actions, changes,
navigation, tabs — works on the model and survives a renderer swap.

## Today (mock renderer + one real extractor)

- `src/features/reader/DocumentView.tsx` — **source documents**: maps
  blocks to styled headings/paragraphs/lists, renders highlights, maps
  native text selections to `TextSelection` (block-anchored). Footer still
  reads "Mock structured render…" on mock-opened files.
- `src/features/reader/SummaryDocumentView.tsx` — **summary document**:
  same block mapping but with plain-text `contentEditable` blocks
  (BASIC WORKING IMPLEMENTATION — no rich formatting, no Slate) and the
  pending-change treatments (inserted card / modified underline).
- Extraction is split by type. `.txt`/`.md`/`.docx` are REAL in the desktop build:
  click → `App.OpenFile(path)` → extract-cache lookup (root-anchored path +
  mtime + size; 200-entry LRU) → miss → `Filesystem.Resolve` (workspace containment)
  → `Documents.OpenFile` extension dispatch → `openTextFile` (Stat-gated,
  `LimitReader`-bounded) / stdlib OOXML → `encoding/json`-marshalled blocks
  with **stable content-addressed IDs** → new tab with genuine paragraphs
  (`backend/extract/extract.go`, `ids.go`). Everything else
  returns `ErrUnsupportedType` with a typed `reason` value (no mock fallback).
  `DocumentService.Extract` (abstract-wrapped stub) is superseded by
  `OpenFile` and has zero callers.

## Integration points by format

| Format | Suggested library | Where |
|---|---|---|
| **PDF** | Extraction: `ledongthuc/pdf` LANDED (pure Go, per-page text rows → paragraph recovery by vertical gap, `page`-break blocks, stable IDs). Canvas: `pdfjs-dist` LANDED (`PdfView`, lazy raster + text layer, worker via `public/pdf.worker.min.mjs`). Limits: text-layer PDFs only; tables/images/layout dropped | `backend/extract/pdf.go`; `frontend/src/features/reader/PdfView.tsx` |
| **DOCX** | `unidoc/unioffice` in Go → blocks (check license before adopting; same library later covers DOCX save-out) — stdlib `archive/zip`+`encoding/xml` text-only arm as stopgap; full evaluation in `tasks.md` §4 | Same slot: new `OpenFile` arm. |
| **HTML / web** | Real browser engine for web sources (see architecture.md §Browser); `golang.org/x/net/html` for saved HTML files | `MockWebPage` slot in the browser; new `OpenFile` arm for saved files. |
| **Plain text** | REAL today (`openTextFile`: blank-line paragraphs, caps) | no work needed. |

Editor contract (from `tasks.md` §3, recorded here so renderer work respects
it): block/segment stays canonical and persisted. If Slate.js lands for
summaries, convert at the editor boundary only (block → Slate Element,
segment → Slate Text with marks) and convert back on save — never store
Slate's internal value, never invent a second wire format.

Rule: the renderer is a **leaf component per format**, selected by
`document.metadata.format`. The wrapper (page header, selection toolbar,
scroll container, `data-block-id` anchors for selection mapping) stays.

## Selection & highlights during the upgrade

- Current precision: block-level anchors + **range offsets** (`TextSelection.range`
  / `Highlight.range`: UTF-16 offsets in the block's concatenated text, measured
  from the native DOM Range when the selection sits in one block; multi-block
  stays block-anchored). Saved notes with ranges underline exactly the span.
- Block IDs are **content-addressed and stable** (`b-<hash>` over path + type +
  text + occurrence) — reopening an unchanged file yields identical IDs, so
  highlights, pending changes, and chat anchors survive. Edited blocks get new
  IDs, correctly orphaning stale anchors.
- Pending summary modifies render as **inline Slate decorations** (click the
  underlined span for accept/reject/revise); deletes/inserts stay card-based.
- Next: pdf.js text-layer coordinates for PDF pages (block offsets are the
  interim system); range-precise `doc.highlight.add` already isolates the UI
  code in `DocumentView.tsx`.
- Existing highlights/notes remain valid: range-less entries match the whole block.

## What stays untouched when you swap renderers

`state/documents`, `state/changes`, `AIReadingPanel`, `ReaderTabs`,
`ReaderSidebar` (TOC is generated from `level===2` heading blocks — the
new renderer just has to produce heading blocks or feed an outline),
and the AI pipeline.
