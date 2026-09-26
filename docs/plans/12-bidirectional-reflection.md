# 12 — Bidirectional reflection (AI ↔ frontend)

**Status:** ACTIVE  
**Depends on:** 09 (intermediate documents), 11 (events + tools)  
**Audience:** Agents wiring live document state between Go, intermediates, and React/Slate

## Goal

- When the AI proposes or (after accept) applies a change, the frontend updates immediately.
- When the user edits the summary (or other intermediate), the next AI turn sees that change without a manual “refresh”.
- Saving is reliable: accepted changes and user edits land in the intermediate files + meta, and can be exported to DOCX/PDF.

This is the harness-style live loop, implemented on top of intermediate files rather than direct DOCX mutation.

## Data flow

```
User edits in Slate / canvas
        │ dispatch + short debounce or explicit save
        ▼
Backend writes intermediate content + updates meta.edits (actor=user)
        │
        ▼
Next AIRequest includes latest intermediate snapshot (or recent block IDs)
        │
AI tools / chat loop
        │ propose_summary_edit → DocumentChange (pending)
        ▼
Frontend shows proposal (highlight / card)
        │ user Accept / Reject / Revise
        ▼
On Accept: apply to intermediate + meta.edits (actor=ai) + optional history snapshot
           emit event so UI is already in sync
```

## Frontend responsibilities

- Treat the intermediate (loaded via backend) as the document the editor shows.
- On user edit: update local Slate state *and* notify backend (content + which blocks changed).
- On incoming AI proposal: render via existing `DocumentChange` UI; never auto-apply.
- On accept: optimistically update UI; backend remains source of truth after write.
- Show “Thinking” / tool phases from the extended event stream (11).

## Backend responsibilities

- Own the intermediate files and meta sidecars (09).
- Accept “document updated” payloads from the frontend and persist them.
- When building an `AIRequest` context pack, include:
  - current intermediate content (budgeted),
  - short recent `edits` / `aiActions` from meta,
  - any live selection or recently-edited block IDs supplied by the frontend.
- On accepted proposal: write intermediate + meta, append standby history entry if configured, emit confirmation event.

## Saving

- **Autosave (accepted AI + user edits):** write intermediate + meta under the workspace’s intermediates tree.
- **Explicit Export / Save as DOCX (or PDF):** run the export path from intermediate → standard format; user may choose destination.
- Dirty flag lives in meta.state and can be shown in the UI.

## Agent rules

1. Never let the model write the intermediate (or the original DOCX) without going through a proposal + user gate (except pure meta/guidance notes under allowed paths).
2. Keep the existing `DocumentChange` shape; extend it only if needed for intermediate block IDs.
3. Prefer block-level or range-level updates so the UI can highlight precisely.
4. Do not require a full file rewrite on every keystroke; debounce user→backend sync reasonably.
5. If intermediate and in-memory frontend state diverge, backend intermediate wins after the next successful load/write.

## Out of scope

- Real-time multi-user collaboration
- Character-level OT/CRDT across clients
- Automatic background export on every keystroke
