# Backend structure — services layout & storage

> **Superseded by `06-backend-resplit.md`** (history preserved, not rewritten).
> The `services/ai/` split it describes has since grown into top-level
> `backend/{ai,extract,tools,usage}` packages plus a split `models/`
> (`document/ai/storage`). Read 06 for the current layout; read this file
> for the original principles (backend as system of record, thin Wails
> boundary, storage migration path) — all still in force.

## 1. Principles

1. **Backend is the system of record** for workspaces, sources metadata,
   documents (canonical JSON), changes, chat threads, and AI activity
   history.
2. **Frontend is JIT**: open tabs, Slate editor state, selection, panel
   widths, draft messages. Flush durable edits via explicit save or
   auto-save hooks.
3. **AI grows behind a package boundary**, not a single `ai.go`.
4. **Keep Wails boundary thin**: `frontend/app.go` only binds; pure logic
   stays under `backend/`.

## 2. Target layout

```
backend/
├── main.go                      # Backend aggregate (existing pattern)
├── models/
│   └── models.go                # shared DTOs (+ ChatMessage, Workspace, …)
├── services/
│   ├── filesystem.go            # walk / resolve / reveal (existing)
│   ├── workspace.go             # library tree, session root (extend)
│   ├── documents.go             # OpenFile extraction dispatch (extend)
│   ├── storage.go               # → becomes interface + sqlite impl
│   ├── storage/
│   │   ├── memory.go            # keep briefly for tests / bootstrap
│   │   └── sqlite.go            # long-term
│   └── ai/                      # NEW package (was services/ai.go)
│       ├── service.go           # AIService interface, NewAI(), Run entry
│       ├── gemini.go            # HTTP client, generateContent
│       ├── prompts.go           # buildPrompt, templates per operation
│       ├── context.go           # ContextPack assembly, token budget
│       ├── tools.go             # list/read/propose tool registry
│       ├── tools_workspace.go   # workspace-scoped tool implementations
│       ├── chat.go              # multi-turn loop, tool calls, history
│       └── types.go             # internal request/result shapes
```

Rationale:
- `services/ai/` isolates model vendor code, prompt policy, and tool
  execution so `documents` / `filesystem` stay readable.
- Further backend top-level packages (e.g. `backend/extract/`) can land
  later when PDF/DOCX parsers grow; not required for the harness MVP.

## 3. Storage migration path

### Today
- `storage.go` in-memory maps.
- React `AppState` holds the live documents/sources/changes.

### Target
- `StorageService` methods roughly:
  - Workspace: Get/Set root, list known workspaces
  - Sources / Documents / Changes / Activities / ChatMessages CRUD
  - Query by `workspaceId`
- SQLite file location options (pick one in implementation):
  - App data dir (global library index), **or**
  - `.conductino/` sidecar inside the open folder (per-workspace)
- Frontend `createInitialState` loads from backend on startup / folder
  open; reducer still applies local patches for open docs, then
  `backend.storage.saveDocument` on accept/save.

### What stays ephemeral (frontend only)
- `selection`, toast, panel open/width, companion one-shot payload
  (recomputable), Slate `editor.children` until save.

## 4. AI package responsibilities

| File | Role |
|---|---|
| `service.go` | `Run(ctx, req, sink)`; operation switch; chat vs oneshot |
| `gemini.go` | network, key, MAX_TOKENS handling (move from current ai.go) |
| `prompts.go` | templates; customPrompt merge; no file I/O |
| `context.go` | given document ids + selection → ContextPack |
| `tools.go` | schema, dispatch, security checks |
| `tools_workspace.go` | uses Filesystem + Documents + Storage |
| `chat.go` | loop: model → optional tool → model → done event |

Streaming contract with frontend (`ai://event`) remains; add event types
if needed (`tool_start`, `tool_result`) without breaking existing
phase/done/error.

## 5. Security guard (folder tools)

All tool paths:
1. Join against **current workspace root** only via existing
   `Filesystem.Resolve`.
2. Reject if outside root or disallowed extension.
3. Cap read size (reuse documents 2 MiB / block caps).
4. Log tool name + relative path only (never absolute path with user
   home in UI errors if avoidable).

## 6. Reducing frontend in-memory authority

| Data | Today | Target |
|---|---|---|
| Library tree | React useState cache | Keep as cache; source of truth Go |
| Documents | Full text in AppState | Open docs cached; rest on demand |
| Changes | Full map in AppState | Load pending for active summary |
| AI activities | Full array | Recent window + query |
| Chat | N/A | Backend thread; UI window |
| Mock seed (`mock/data.ts`) | Startup corpus | Fixtures for tests only |

## 7. Migration steps (ordering)

1. Split `ai.go` → `services/ai/*` **without** behavior change (compile +
   same events).
2. Add `workspaceId` fields to models + TS domain (nullable first).
3. Implement SQLite storage behind interface; dual-write or feature-flag.
4. Wire chat + tools once storage can hold threads.
5. Deprecate first-summary-wins path once primarySummaryId is set on
   workspace open/create.
