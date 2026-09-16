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
- Extraction is split by type. `.txt`/`.md` are REAL in the desktop build:
  click → `App.OpenFile(path)` → `Filesystem.Resolve` (workspace containment)
  → `Documents.OpenFile` extension dispatch → `openTextFile` (`os.ReadFile`,
  2 MiB / 1000-block caps, `encoding/json`-marshalled blocks) → new tab with
  genuine paragraphs (`backend/services/documents.go`). Everything else
  returns `ErrUnsupportedType` and the UI falls back to the mock factory
  (`makeDocumentFromSource` in `src/mock/data.ts`, `(mock extraction)` toast).
  `DocumentService.Extract` (abstract-wrapped stub) is superseded by
  `OpenFile` and has zero callers.

## Integration points by format

| Format | Suggested library | Where |
|---|---|---|
| **PDF** | Go extraction-first (reuses selection/AI immediately): `pdfcpu` (Apache-2.0, memory-efficient) or `ledongthuc/pdf` (tiny, text-only) — full evaluation in `tasks.md` §4, which also rules out `unipdf` (license/weight) and cgo bindings. Canvas alternative: `pdfjs-dist` per page | New `case` arm in `Documents.OpenFile` emitting block/segment JSON; on failure surface a typed reason (see `tasks.md` §1.2) instead of falling into the mock path. |
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

- Current precision: block-level. `TextSelection { blockId, text }` and
  `Highlight { blockId, text }` already separate *what was selected* from
  *where it lives*.
- Target: add `range: { start, end }` (or pdf.js text-layer coordinates)
  to `TextSelection`/`Highlight`. UI code that reads them is already
  isolated in `DocumentView.tsx` + the `doc.highlight.add` action.
- Existing highlights/notes remain valid: they are block-anchored until
  ranges are available.

## What stays untouched when you swap renderers

`state/documents`, `state/changes`, `AIReadingPanel`, `ReaderTabs`,
`ReaderSidebar` (TOC is generated from `level===2` heading blocks — the
new renderer just has to produce heading blocks or feed an outline),
and the AI pipeline.
