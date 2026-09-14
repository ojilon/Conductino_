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

## Today (mock renderer)

- `src/features/reader/DocumentView.tsx` — **source documents**: maps
  blocks to styled headings/paragraphs/lists, renders highlights, maps
  native text selections to `TextSelection` (block-anchored).
- `src/features/reader/SummaryDocumentView.tsx` — **summary document**:
  same block mapping but with plain-text `contentEditable` blocks
  (BASIC WORKING IMPLEMENTATION — no rich formatting yet) and the
  pending-change treatments (inserted card / modified underline).
- Extraction is mocked: opening a file builds a small `Document` in
  `src/mock/data.ts` (`makeDocumentFromSource`). In a Wails build the Go
  side owns extraction: `backend/services/documents.go`
  (`DocumentService.Extract`).

## Integration points by format

| Format | Suggested library | Where |
|---|---|---|
| **PDF** | `pdfjs-dist` (render pages to canvas inside the webview) **or** Go extraction with `pdfium`/`unipdf` + text in `DocumentModel` | Replace the block loop in `DocumentView.tsx` with a `<PdfPage>` component per page; keep `metadata.pageCount` + `currentPage` as the page contract. Two viable shapes: (a) canvas pages + text-layer selection, (b) extracted blocks rendered by the existing DOM path. (a) is closer to "real PDF", (b) reuses selection/AI immediately. |
| **DOCX** | `docx-preview` (WYSIWYG-ish, webview) **or** `unidoc/unioffice` in Go → blocks | Same slot as PDF: a `<DocxView>` inside `DocumentView.tsx`, or extraction into `DocumentModel`. |
| **HTML / web** | Real browser engine for web sources (see architecture.md §Browser); `jsdom`/`x/net/html` for saved HTML files | `MockWebPage` slot in the browser; `DocumentView` slot for saved files. |
| **Plain text** | line split (already in the model) | no work needed. |

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
