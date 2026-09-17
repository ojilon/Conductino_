# Architecture — academic harness (reader ↔ AI)

## 1. Mental model

```
┌─────────────────────────────────────────────────────────────┐
│  Workspace (open folder root)                               │
│  ├── sources/  (PDF, DOCX, HTML, TXT — read-only)           │
│  └── summary.docx (or multiple) — editable by user + AI     │
└─────────────────────────────────────────────────────────────┘
         ▲ read tools (guarded)              │ edits as DocumentChange
         │                                   ▼
┌────────────────────┐              ┌─────────────────────────┐
│  AI harness (Go)   │◄── chat ────►│  Chat UI (React)        │
│  tools + context   │              │  history, @mentions     │
│  Gemini / stream   │              │  selection + custom     │
└────────────────────┘              └─────────────────────────┘
```

- **Backend** owns: workspace root, file tools, context assembly, model
  calls, long-term persistence, change proposals as data.
- **Frontend** owns: rendering, selection, Slate editing surface, accept/
  reject UX, just-in-time UI state (open panels, draft message, scroll).

## 2. Identity & scoping (fixes first-summary-wins)

Introduce a first-class **WorkspaceSession** (name TBD):

```ts
WorkspaceSession = {
  id: ID,
  rootPath: string,           // absolute, backend-only authority
  primarySummaryId?: ID,      // mapped summary document
  sourceIds: ID[],            // documents opened/known under this root
  chatThreadId?: ID,
}
```

Rules:
- All `Document`, `Source`, `DocumentChange`, `AIActivity` that belong to
  a folder carry `workspaceId` (or `sessionId`).
- `runIncludeInSummary` and chat-driven merges resolve target via
  `workspace.primarySummaryId`, never `Object.values(documents).find(...)`.
- Switching folder creates or resumes a WorkspaceSession; stale tabs keep
  `rootPath` tags (existing tasks.md 1.1 option b).

## 3. Chat surface

### 3.1 Placement
- **Primary:** evolve the right AI panel into a dual-mode panel:
  - **Chat** (default for conversation + custom prompts)
  - **Review** (pending changes for the active summary)
- Optional later: a thinner left-rail chat entry; same thread backend.

### 3.2 Message model (persisted)

```ts
ChatMessage = {
  id, workspaceId, role: "user" | "assistant" | "system",
  content: string,
  // structured attachments
  selection?: { documentId, blockId?, range?, text },
  mentions?: { kind: "source"|"summary"|"file", id, path? }[],
  changeIds?: ID[],          // proposals this turn produced
  activityId?: ID,
  createdAt,
}
```

Frontend keeps only the **active thread view** + composer draft in React
state; history loads from backend.

### 3.3 @ mentions
- Composer parses `@token` against library tree + open sources + summary
  (`state/mentions.ts` — title/label/path substring match, first hit wins).
- On send, resolved mentions become explicit `mentionIds` in the request so
  the harness does not guess; the raw `@token` stays in the query text.
- A mentioned summary overrides `workspace.primarySummaryId` as the
  proposal target. The live selection rides along as the region anchor.
- Status: LANDED (autocomplete dropdown in `ChatFace` is a typing aid;
  send-time resolution is ground truth).

## 4. Context assembly (algorithms on backend)

When the user asks to explain / expand / chat:

1. **Selection core** — exact text + anchors.
2. **Local window** — N blocks before/after (or page window for PDF).
3. **Document outline** — heading structure + short abstracts.
4. **Mentioned / open sources** — titles + truncated extracts (budgeted).
5. **Summary snapshot** — current summary text (or structured blocks)
   when the task is merge/improve/shorten.
6. **Token budget** — hard caps; drop lowest-priority layers first.

Output of assembly is a single **ContextPack** string/structured payload
appended to the model prompt. Never dump entire multi-MB files blindly.

## 5. Tool surface (harness)

Minimal tool set (Go, invoked only by the AI service loop):

| Tool | Effect | Guard |
|---|---|---|
| `list_workspace` | tree under root | root only |
| `read_source` | text extract for path/id | Resolve containment + allowed ext |
| `read_summary` | current summary content | workspace’s summary only |
| `propose_summary_edit` | create DocumentChange(s): `op=insert\|modify\|delete`, `target`, `text` | summary id must match mapping; user accepts in UI |
| `search_in_workspace` | quoted keyword snippets over readable files | root jail + ext allowlist + file/match/output caps |

No shell, no arbitrary path, no write to sources. The summary is a living
document: the model may redefine, restructure, or remove — never
append-only. The user gatekeeps every proposal with approve / revise.

No shell, no arbitrary path, no write to sources.
Tool results are injected into the same turn’s model context. Streaming
phases already exist (`onPhase`); extend with tool-phase labels.

## 6. AI request contract extensions

Extend (do not replace) existing `AIRequest`:

```ts
AIRequest += {
  workspaceId?: ID,
  customPrompt?: string,
  mentionIds?: ID[],
  changeId?: ID,              // already used for rewrite; generalize
  includeDocumentContext?: boolean,
  mode?: "oneshot" | "chat",
  threadId?: ID,
  messageHistory?: { role, content }[],  // or load by threadId server-side
}
```

Operations stay; add:
- `AI_CHAT` — multi-turn with tools.
- Optional: `AI_SUMMARY_IMPROVE`, `AI_SUMMARY_SHORTEN` as specialized
  chat prompts that always produce DocumentChange proposals.

## 7. Change / diff workflow (unchanged core, richer entry points)

Existing `DocumentChange` + accept/reject/revise stays the review gate.
New entry points:
- Selection “Include in summary” (existing).
- Chat: “add this to the summary”, “improve the methods section”, etc.
- Chat or panel: reference `changeId` → revise / expound / narrow.

Frontend still never auto-applies edits to the summary without user
accept (or an explicit user setting later).

## 8. Frontend vs backend responsibility

| Concern | Owner |
|---|---|
| Workspace root, path security | Backend |
| Extraction, context packs, tools, Gemini | Backend |
| SQLite / workspace sidecar persistence | Backend |
| Chat history storage | Backend |
| DocumentChange persistence | Backend |
| Slate document value (transient) | Frontend |
| Selection, toolbar, chat composer | Frontend |
| Accept/reject UI | Frontend |
| Live editing of summary blocks | Frontend (JIT), flush to backend on save |

In-memory `AppState.documents` becomes a **cache** of what is open or
recently used, not the system of record.
