# Source reading — paged, normalized, resilient

> Status: **plan** (07) — builds on 06 step 1 (`extract/` package).
> Problem it solves: today `read_source` dumps up to 6k chars from the top
> of a file. A summary spanning pages 1→2, a definition redefined on page 9,
> or a typo'd filename all defeat it. The AI needs to *read through* a
> source the way you do: locate, page, continue.

## 1. Core ideas (three, in dependency order)

### 1a. Normalized intermediates (fast, canonical, once per file version)

Every successfully extracted document also materializes a **normalized
intermediate**: plain-text Markdown-ish view (headings kept as `#`, lists as
`- `, tables as `a | b`, pages as `--- page N ---` markers) plus a
**page/block map** (`page → first blockId`, `blockId → page/offset`).

* Storage: extend `extract_cache` (new columns `md_text`, `page_map_json`)
  — same key (root-anchored path + mtime + size), same LRU. No new table,
  no sidecar files to sync.
* Cost: one extra string pass at extract time (already in memory); reads
  after that never re-parse OOXML/PDF structures.
* Both AI tools and your future "view as text" toggle read the same
  intermediate — one canonical text per file version, never two
  divergent renderings.

### 1b. Paged / windowed reads (the dynamic-reading API)

`read_source` gains optional window args (all clamped server-side):

```
<tool name="read_source" path="paper.pdf" pages="1-3"/>
<tool name="read_source" path="notes.md" offset="4000" limit="2000"/>
```

Response envelope (always):

```
File "paper.pdf" — pages 1-3 of 14, chars 0-6000 of 41200, blocks b-… … b-…
--- page 1 ---
…text…
--- page 2 ---
…text…
[continued — ask pages="4-6" or offset=6000 for more]
```

* `pages="N"` / `"N-M"` for paginated formats (PDF page map; docx/txt
  synthesize ~2000-char "pages" so the same verb works everywhere).
* `offset`/`limit` (chars) with **200-char overlap** between consecutive
  windows — the model never loses a sentence split across a boundary.
* Every window carries **totals + next cursor**, so "continue reading" is
  one tool call, never a guess. Unknown page → honest range error with the
  valid range restated.
* Truncation markers (`[…continued]`) stay machine-readable: the loop
  treats them as "more available", not end-of-file.

### 1c. Resilient naming (LANDED — keep extending here, not elsewhere)

`resolveSourcePath` (tools.go): exact case-insensitive → unique stem
(`"Summary"` → `"Summary.docx"`) → Levenshtein-ranked did-you-mean
(capped 3). All future path-taking tools reuse it — never a second
matcher. Candidate next: content search by title (`read_source
title="…"`), decided later.

## 2. Overlap handling (your page-1→2 case, made mechanical)

1. Model reads `pages="1-2"` → gets text + totals + next cursor.
2. Summary-relevant span crosses the boundary → model asks `pages="2-3"`
   (overlap is structural: page 2 repeats, keyed by identical blockIds,
   so the model sees continuity, not duplication).
3. `read_summary` snapshot rides along (already does) → harmonize against
   existing content → `propose_summary_edit` (insert/modify/delete).
4. UI marks the proposal; you approve or revise. Unpredictable spans are
   contained because every proposal is anchored to blockIds the model
   actually saw in a window.

Rule for prompts (catalog): *never summarize a page you have not read in a
window; if the span may continue, read the next window first.* One line,
enforced by the envelope (totals make bluffing visible).

## 3. Tool + bridge changes

* `read_source` args: `path` (required), `pages` (`"N"`/`"N-M"`),
  `offset`, `limit`. Legacy bare calls behave exactly as today (first
  window, totals appended — a strict improvement, no break).
* New bridge endpoint `App.ReadSourcePage(path, pages|offset, limit)` for
  direct UI use later (a "view as text at page N" affordance); TS helper
  beside `openRawFile`. Phase 1: tools-only, no UI.
* `search_in_workspace` results gain page numbers (`path:page`) from the
  page map, so hits are directly readable via `pages="N"`.
* Caps (low-spec): window ≤ 6000 chars (as today), pages/window call ≤ 3
  pages, intermediates ≤ 2 MiB/file (beyond → windowed-only, no full
  materialization).

## 4. Execution steps

1. **Schema**: `extract_cache` += `md_text TEXT`, `page_map_json TEXT`;
   `CachedExtract` struct + both impls; migration is `CREATE TABLE IF NOT
   EXISTS` additive — old DBs upgrade silently.
2. **Normalize**: `extract/normalize.go` — blocks → md text + page map
   (page-break blocks are the source of truth; formats without breaks get
   synthesized 2000-char pages). Unit tests on txt + docx fixtures.
3. **Window**: `extract/page.go` — `Window(md, map, pages|offset, limit)`
   pure function returning `{text, start, end, total, next, blockIds}`;
   table tests for edges (page 0, beyond-end, overlap bytes).
4. **Tools**: `read_source` parses window args, reads via intermediate,
   emits the envelope; catalog documents `pages`/`offset`; fuzzy path
   stays first. Tests: page 2 differs from page 1; continuation cursor
   round-trips; did-you-mean still fires on typos.
5. **Bridge**: `App.ReadSourcePage` + TS helper (tools use it internally
   first; UI affordance later).
6. **Search**: `path:page` hits. Docs: this file → move to past tense;
   `01` tool table + `ai-integration.md` operation notes updated.

## 5. Done criteria

* A 14-page PDF summarizes correctly across a page-1→2 boundary in a live
  chat (the original failing shape), with ≤ 3 tool calls.
* Typo'd/missing-extension reads self-heal via did-you-mean (already true;
  regression-tested).
* `go vet`, backend tests, `tsc`, `pnpm build` green; extract package has
  no network/Wails imports (`go list -deps` check in CI spirit).
