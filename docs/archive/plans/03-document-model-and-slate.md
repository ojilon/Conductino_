# Document model, Slate, highlights, and editing

## 1. Tension to resolve

Today’s **block/segment** model is excellent for:
- Source rendering (read-only)
- AI selection anchors (`blockId` + text)
- Simple pending change cards

It is **weak** for:
- Rich editing (lists, nested marks, multi-block selection)
- Character-range highlights that survive edits
- DOCX fidelity (runs, styles)
- Slate’s tree value as source of truth (must not become that)

`tasks.md` §3 already states the rule: **block/segment stays canonical**;
Slate is a transient view. This plan doubles down on that and names the
highlight/edit risks.

## 2. Canonical format (persist + AI + extract)

Keep and evolve:

```
DocumentBlock { id, type, level?, segments[], listItems?, changeId? }
Segment { text, em?, strong?, highlightId? }
```

Add when needed:
- Optional `range` on Highlight / TextSelection: `{ start, end }` **within
  block** (UTF-16 or rune index — pick one and document it).
- Stable `blockId` across sessions (uuid), never index-based.

Extraction (PDF/DOCX/TXT) continues to emit this shape. AI context packs
consume this shape. Storage persists this shape.

## 3. Slate boundary (summary only)

```
Canonical blocks  ──toSlate──►  Slate Value  (editor only)
Canonical blocks  ◄─fromSlate──  Slate Value  (on save / blur / accept)
```

Rules:
- **Never** store Slate JSON in SQLite or ship it over Wails as the
  document body.
- `SummaryDocumentView` becomes a Slate editor that:
  - Hydrates from `doc.blocks` when the tab activates or remote changes
    arrive (accept/reject).
  - On local edit, updates a **draft** in frontend state; periodic or
    explicit save calls `fromSlate` → `backend.saveDocument`.
- Pending AI inserts/modifies still appear as overlays or temporary
  nodes tagged with `changeId`; accept merges into canonical blocks then
  re-hydrates Slate.

## 4. Highlights under editing

### Sources (read-only)
- Block-anchored highlights remain valid.
- Upgrade path: character range within block once renderers support it
  (pdf.js text layer, etc.).

### Summary (editable)
- User highlights must not break when neighboring text is edited.
- Prefer **mark-based** highlights inside Slate mapped to `highlightId`
  on segments after `fromSlate`.
- AI “expound on this” should send:
  - either `changeId` (stable), or
  - current selected text + block ids **after** mapping selection out of
    Slate via the same adapter.

Risk if ignored: contentEditable/Slate rewrites DOM → lost `data-block-id`
anchors. Mitigation: always derive selection payload through the adapter,
never from raw DOM ids alone once Slate owns the surface.

## 5. DocumentChange vs Slate

| Change type | Today | With Slate |
|---|---|---|
| insert | New block + pending card | Insert node(s) marked pending; accept commits to canonical |
| modify | Underline fragment | Decorations or inline marks; accept rewrites segments |
| delete | Strike-through (supported type) | Same |

Revise/expound from chat targets `changeId` → backend returns new
`newContent` → frontend updates change record → Slate decoration updates.

## 6. DOCX

- **Read sources:** Go extractor → blocks (unioffice or zip+xml stopgap).
- **Write summary:** on save, blocks → DOCX via same library family.
- Summary “file under folder” = real path under workspace root + metadata
  in storage; opening folder can list the summary like any other file but
  open path uses the editable editor, not the read-only source view.

## 7. What not to do

- Do not dual-write Slate value and blocks without a single conversion
  boundary.
- Do not make AI return Slate ops; AI returns text / structured
  insertion / revision against the **canonical** model.
- Do not drop block ids when migrating to Slate; they are the join key for
  changes and highlights.

## 8. Implementation order (document side)

1. Stabilize block ids + optional ranges on selection (no Slate yet).
2. Context assembly using blocks (harness value without editor rewrite).
3. Introduce `toSlate` / `fromSlate` adapters behind Summary view.
4. Replace contentEditable with controlled/uncontrolled hybrid Slate.
5. DOCX save-out for summary.
6. Richer highlights (marks + range).
