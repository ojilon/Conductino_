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

## 2. Temp mirrors (planned — Phase 17)

AI summary editing needs a text surface the model can read/write while the
`.docx` stays canonical. Decision: **project-local scratch dir**
`backend/.work/` (git-ignored, see root `.gitignore`), NOT the OS temp dir
(debuggable, survives restarts, one place to wipe).

- Key: root-anchored path + mtime + size (same scheme as `extract_cache`).
- Content: normalized `.md` mirror of the summary (from `extract.normalize`,
  plan 07) — the ONLY text the edit tools read/write.
- Lifecycle: created on summary open/create; refreshed when the `.docx`
  mtime moves under it; deleted with the workspace or by age (7-day sweep
  on startup). Never user-facing, never synced.
- `read_summary` / `propose_summary_edit` operate on the mirror; the
  accept path writes mirror → blocks → `.docx` (existing `WriteSummaryDOCX`).

## 3. Page canvas renderer (planned — Phase 17)

Goal: docx/md/txt render as **pages**, not one big text block. Scope:
custom canvas for docx first (we own writer+reader), md/txt reuse it via
the normalized mirror; **PDF stays on pdf.js** (no second PDF renderer).

- Paginate the canonical blocks (`~2000-char` virtual pages, same constant
  as `extract.SynthesizedPageSize`); page containers anchor to blockIds so
  selection/highlights/chat anchors keep working unchanged.
- Block/segment stays the wire format; pagination is a view concern only.
- Empty document = one empty page (editors already handle the blank block).

## 4. Explicit non-goals

OS file watcher (focus-refresh + manual is enough until proven otherwise);
OCR; editing source PDFs/DOCX (summaries only).
