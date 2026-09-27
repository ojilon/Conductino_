# Live library, empty docs, intermediates, page canvas

> Status: **partially landed**. Born from a real session: a locally created
> `Summary.docx` (0 bytes) was invisible to search AND unopenable
> (parse_error), because the tree was stale and the extractor rejected
> empty files.

## 1. Landed

- **Empty files open as blank pages** (`extract.emptyDocument` + arms in
  `text.go`/`docx.go`/`pdf.go`, tested in `empty_test.go`). A 0-byte file of
  any supported type yields title + one empty paragraph — never an error.
  This also unblocks `search_in_workspace`, which silently skips
  unopenable files.
- **Tree refresh on window focus** (`ReaderMode.tsx`): re-lists when the
  window regains focus (visible-only, single-flight). Files created in
  Explorer appear without a manual refresh. Manual refresh button kept.
- **Open focuses the existing tab** (`ReaderMode.openFile`): click matches
  open tabs by path token + opening root before minting new IDs. Same-named
  files from another folder still open fresh (stale-root guard, tasks.md
  §1.1).

## 2. Temp mirrors (landed — Phase 17)

`backend/mirror/` owns summary `.md` working copies under `backend/.work/`
(git-ignored): snapshot-hash re-sync on user save, op log per edit,
7-day sweep on `Backend.Init`. `read_summary` prefers the mirror (empty
summary is valid content); `propose_summary_edit` applies to it, so
multi-turn refinement composes. Review accept still writes the `.docx`
(frontend blocks → `WriteSummaryDOCX`), whose mtime bump re-syncs the
mirror next read. No new bridge was needed.

## 3. Page canvas renderer (landed — Phase 17)

`paginateBlocks` in `DocumentView.tsx`: greedy ~2000-char virtual pages
(same constant as `extract.SynthesizedPageSize`), headings never stranded,
`page` blocks force breaks, empty doc = one empty page. Blocks keep
`data-block-id` wrappers, so selection/highlights/chat anchors work
unchanged. Applies to docx/md/txt (PDF routes to `PdfView`, untouched).

## 4. Explicit non-goals

OS file watcher (focus-refresh + manual is enough until proven otherwise);
OCR; editing source PDFs/DOCX (summaries only).
