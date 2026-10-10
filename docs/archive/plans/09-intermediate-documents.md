# 09 — Intermediate documents (mirrors + metadata)

**Status:** ACTIVE  
**Depends on:** `docs/release-prep/` (local storage paths must exist)  
**Audience:** Agents implementing document open / edit / AI context / save

## Goal

The AI and the live editor do **not** work directly on the user’s original DOCX/PDF.  
They work on **intermediate files** (fast, plain, metadata-rich) that live in local storage.  
Standard formats (DOCX, PDF, …) are import sources and export targets only.

This enables:

- Fast AI context assembly and tool reads
- Immediate frontend reflection of AI and user edits
- Clear history of who changed what (AI vs user)
- Robust standby change tracking (future git-like tracker can sit on top)

## Mental model

```
User’s original files (read-only sources)
        │ extract once (or on demand)
        ▼
Intermediate mirror set  ←── live editing + AI tools work here
  • content (.md / .txt / structured JSON blocks)
  • metadata sidecar (edit marks, AI history, state)
        │ on explicit Save / Export
        ▼
Standard document (DOCX, PDF, …) written back or to a chosen location
```

Sources stay read-only in the workspace folder.  
The editable summary (and any future editable docs) are owned by the intermediate layer.

## Layout on disk (under app data + per-workspace)

Exact roots are defined in `docs/release-prep/02-local-storage-layout.md`.  
Conceptually:

```
<app-data>/workspaces/<workspace-id>/
  intermediates/
    sources/
      <stable-id>.md          # plain text / markdown extraction
      <stable-id>.meta.json   # block map, page map, extraction info
    summary/
      current.md              # living summary content (or blocks JSON)
      current.meta.json       # edit history, AI actions, open state
      history/                # optional standby snapshots (see below)
  cache/
    extract/                  # raw extraction artefacts if needed
```

Dev/debug mirror (before release): see `docs/release-prep/04-temp-debug-mirror.md`.

## Content of an intermediate file

### Content file (`.md` / `.txt` or blocks JSON)

- Human- and model-readable text.
- For sources: extraction result (already partially exists in `OpenedDocument` / blocks).
- For summary: the live document the user and AI edit.

Prefer a simple format the AI can read with existing tools (`read_source`, `read_summary`).  
Block-oriented JSON is allowed when structure matters (headings, lists); the tool layer flattens for the model when needed.

### Metadata sidecar (`.meta.json`)

Minimum fields (extend as needed, keep versioned):

```json
{
  "version": 1,
  "sourcePath": "relative/path/in/workspace.pdf",
  "stableId": "…",
  "updatedAt": "ISO-8601",
  "blocks": [
    { "id": "b1", "kind": "paragraph", "origin": "extract" }
  ],
  "edits": [
    {
      "id": "e1",
      "at": "ISO-8601",
      "actor": "user" | "ai",
      "op": "insert" | "modify" | "delete",
      "blockId": "b1",
      "oldSnippet": "…",
      "newSnippet": "…",
      "changeId": "DocumentChange-id-if-any",
      "note": "optional short reason"
    }
  ],
  "aiActions": [
    {
      "at": "ISO-8601",
      "tool": "propose_summary_edit",
      "summary": "modified methods section",
      "changeIds": ["…"]
    }
  ],
  "state": {
    "lastUserEditAt": "…",
    "lastAiProposalAt": "…",
    "dirty": true
  }
}
```

Rules:

- `edits` and `aiActions` are append-only for the current intermediate; prune only by retention policy.
- The AI receives a **budgeted** view of recent edits/actions (never the whole history) so it can track issues and avoid repeating mistakes.
- Frontend and Go both treat the intermediate + meta as the source of truth for open documents.

## Standby change tracking (robust now, git-like later)

For the first releases:

- Keep a short ring of snapshots under `summary/history/` (e.g. last N accepted states or time-based).
- On every accepted `DocumentChange`, append to `edits` and optionally write a lightweight snapshot.
- This is enough for the AI to answer “what did we change earlier?” and for the user to recover a recent version.

**FUTURE (deferred):** full git-like tracker (commits, branches, blame) on the intermediate tree. Do not implement until the standby path is solid and release-prep is done.

## Import / export path

| Direction | Owner | Notes |
|-----------|--------|--------|
| Original → intermediate | Backend extract (`documents.go`, `pdf.go`, `docx.go` + optional Python helper) | Runs on open or on demand; result cached under intermediates |
| Intermediate → frontend | Backend services + Wails events | Immediate; Slate / canvas read the intermediate |
| User/AI edit → intermediate | Frontend dispatch → backend write of content + meta | Never skip the meta update |
| Intermediate → DOCX/PDF | Backend export | Explicit Save / Export action; user chooses path if needed |

Custom canvas (future) can render from the same intermediate; traditional Word/PDF remain the interchange formats.

## Agent implementation notes

1. Do not teach the model to edit DOCX bytes directly.
2. Extend existing `propose_summary_edit` / `DocumentChange` so that accepting a change updates both the content intermediate and the meta sidecar.
3. When building AI context, prefer the intermediate content + a short “recent edits” excerpt from meta.
4. Keep extraction and export behind the same service boundaries already used for `OpenFile` / save.
5. All new paths must respect the workspace jail and the app-data roots defined in release-prep.

## Out of scope for this plan

- Full git-like versioning
- OCR / scanned PDF improvement
- Multi-user concurrent editing
- Cloud sync of intermediates
