# The Mirror Pipeline — how documents are displayed, edited and saved

This is the beginner-friendly guide to the document rendering system. It
answers: *how does a .docx/.pdf/.txt on disk end up as editable pages in the
app, and how do edits get back into the file?*

## The idea in one paragraph

Real document formats are hard: .docx is a ZIP of XML, .pdf is a binary page
language. The frontend should not fight them. So every document is
**mirrored** into two small JSON files that the app fully owns:

- **`content.json`** — the document text, *free* of pdf/docx rules: plain
  blocks (heading / paragraph / list) with light styling. This is what the
  canvas renders and edits, and what the AI reads.
- **`meta.json`** — the **guide**: how the content sat in the source file
  (kind, page size, fonts, margins, which page each block was on, a text
  hash per block, a freshness stamp of the source file).

The frontend uses `meta.json` to *mirror the look* of the source its own way
(page sizes, fonts — a custom canvas, not a Word clone), and the save bridge
uses the hashes to **verify only the parts that changed** before writing the
source file back.

```
source file (.docx/.pdf/.txt/.md)
    │  backend/extract          reads the real format (ZIP+OOXML, pdf lib…)
    ▼
wire blocks JSON                {blocks:[{id,type,segments},…]}
    │  backend/docpipe.Build    the "process" step (frees content, records meta)
    ▼
content.json + meta.json        <app-data>/cache/extract/mirrors/<key>/
    │
    ▼
frontend MirrorCanvas.tsx       custom pages/fonts from meta, direct editing,
    │                           colored highlights, right-click menus
    │  while typing (debounced) → App.UpdateMirrorContent → content.json
    │                             (the AI's read_mirror sees edits LIVE)
    │  on Save (Ctrl+S / button)
    ▼
App.SaveDocumentMirror → docpipe.WriteBack
    ├─ verifies each block hash against meta.json (changed/added/removed)
    ├─ refuses if the source file changed on disk since it was opened
    ├─ rewrites the source: .docx via OOXML writer, .txt/.md as plain text
    └─ refreshes meta.json (new hashes + stamp) for the next save
```

## Where everything lives

| Piece | File | Job |
|---|---|---|
| Format readers/writers | `backend/extract/` | docx (ZIP+OOXML), pdf (text layer), txt/md |
| Mirror builder | `backend/docpipe/build.go` | blocks → `content.json` + `meta.json` |
| Save bridge | `backend/docpipe/writeback.go` | verified write-back into the source file |
| Summary target marker | `backend/docpipe/target.go` | the workspace's "all summaries go here" .docx |
| Bridge methods | `backend/main.go` (MIRROR PIPELINE section) | `OpenFile`, `SaveDocumentMirror`, `UpdateMirrorContent`, `CreateFile`, `RenameFile`, `SetSummaryTarget`, skills + shell |
| Wails bindings | `frontend/app.go` | same names on `window.go.frontend.App.*` |
| TS mirror types | `frontend/src/types/mirror.ts` | twin of `docpipe/types.go` |
| TS service helpers | `frontend/src/services/backend.ts` | bridge-first wrappers (mock-safe) |
| Custom canvas | `frontend/src/features/reader/MirrorCanvas.tsx` | pages, fonts, editing, highlights, save |
| Selection toolbar | `frontend/src/features/reader/SelectionToolbar.tsx` | Ask AI / highlight colors / To summary |
| Skills browser | `frontend/src/features/reader/SkillsPanel.tsx` | read/edit `skills/`, sandbox shell |
| AI tools | `backend/tools/mirror_tools.go`, `backend/tools/shell.go` | `read_mirror`, `get_summary_target`, `read_skill`, `shell` |
| AI user guide | `backend/skills/app-guide.md` | the AI's internal guide to the app |

## Editing rules (what is editable, what is not)

- **.docx / .txt / .md** → editable on the canvas (`meta.editable = true`).
  Typing never touches the source file; **Save** verifies the changed blocks
  and writes it back. Enter splits a paragraph; Ctrl+S saves.
- **.pdf** → read-only (`meta.editable = false`): rendered by pdf.js
  (`PdfView.tsx`), selectable for highlights, AI actions and "To summary".
  Highlights are listed under the PDF header (the pages themselves are
  images).
- **The summary target** (right-click a .docx in the library → *Make this the
  target summary document*) opens in the full summary editor
  (`SummaryDocumentView.tsx`) with autosave and yellow in-file review of AI
  edits: right-click a yellow span → Accept (keep permanently), Reject, or
  send a custom instruction to the AI.

## Highlights

Colors live in `types/mirror.ts` (`HIGHLIGHT_COLORS`): yellow = AI/review,
green = read, blue = important, pink = question. Select text → click a color
dot in the floating toolbar. Right-click any highlight to keep it
permanently, recolor, remove it, or ask the AI about that exact passage (the
prompt carries the quoted text + the document context).

## The AI's view

The AI never reads .docx bytes. It calls:

- `read_mirror path="notes/paper.docx"` — the document **as the user sees
  it**: metadata header + `--- page N ---` sections (`docpipe.DescribeForAI`).
- `get_summary_target` — which .docx receives summaries. `publish_summary`
  writes proposals into it even when it is closed (the "siphon").
- `read_skill path="my_instructions/rules.md"` — user instruction files from
  the `skills/` folder (mention `@skills/…` in chat to point the AI there).
- `shell command="ls -la"` — basic sandboxed command (cwd = open folder,
  10s timeout, capped output, deny-list).

Because the editor pushes `content.json` on every pause in typing
(`UpdateMirrorContent`), the AI reads the user's LIVE in-app state — the
mirror is the shared truth between user and AI. When the AI publishes edits,
the file on disk changes, the app reloads it through `OpenFile` (which
rebuilds both mirror files), and the yellow review decorations appear.

## Failure behavior (honest, never fabricated)

- Mirroring is best-effort: if `docpipe.Build` fails, the document still
  opens from `blocksJSON` exactly as before (the canvas falls back to
  default page/font settings).
- `WriteBack` refuses to save when `meta.json`'s stamp says the file changed
  on disk since it was opened — reopen the document, then save again.
- Saving with zero changes is a reported no-op (`written: false`), never a
  pointless rewrite.
