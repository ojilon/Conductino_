# Backend resplit — from services/ flat to specialized packages

> Status: **plan** (06) — code still lives in `backend/services/` + `backend/services/ai/`.
> Execute in step order; each step leaves `go vet ./...` + `go test` green.
> Companion: `07-source-reading.md` (paged reads + intermediates) builds on step 1.

## 1. Why the flat layout has to go

`backend/services/` holds four growth axes in one package today, and each is
about to get heavier per the product direction (AI reading whole sources
dynamically, shell-like tooling, parallel providers, usage accounting):

| Axis | Today | Pressure |
|---|---|---|
| Extraction + intermediates | `documents.go`, `docx.go`, `pdf.go`, `blockids.go` + cache methods on storage | Normalized `.md`/`.txt` intermediates, paged reads, per-doc page maps, background jobs |
| Tooling | `services/ai/tools.go` (registry + 6 tools + audit) | Shell-like growth: search, paged reads, gated writes — needs its own sandbox policy + tests |
| Usage / metrics | `services/ai/usage.go`, `policy.go` | Parallel providers, budgets, persisted UsageEvents, per-model quota buckets |
| Network (model) access | `services/ai/*` (backends, prompts, chat loop) | Must stay separable from everything above: offline work (extract, tools, storage) must never import net/http |

Rule after the split: **non-network packages never import network ones.**
`ai` (network) depends on `tools`, `extract`, `usage`; nothing depends on `ai`
except `backend/main.go` and `frontend/app.go`.

## 2. Target layout

```
backend/
├── main.go                 # aggregate only: constructs services, thin wrappers
├── models/
│   ├── models.go           # KEEP as re-export shim (wire stability)
│   ├── document.go         # FileTreeNode, OpenedDocument, OpenFailureReason
│   ├── ai.go               # AIRequest, AIEvent, ChatTurn, FocusedChange
│   └── storage.go          # record structs (Workspace/Document/Change/Chat/CachedExtract)
├── extract/                # NEW — pure Go, NO network, NO Wails (cgo-ready, §6)
│   ├── extract.go          # Extractor interface + dispatch (ext → impl)
│   ├── text.go             # txt/md arm (from documents.go)
│   ├── docx.go             # OOXML arm (moved as-is)
│   ├── pdf.go              # text-layer arm (moved as-is)
│   ├── ids.go              # stableBlockID + normalize (from blockids.go)
│   ├── normalize.go        # NEW: canonical doc → intermediate .md text + page/block map (07)
│   └── page.go             # NEW: windowed reads over intermediates (07)
├── tools/                  # NEW — registry + sandbox policy, no model calls
│   ├── tools.go            # ToolHost, Dispatch, catalog, audit (moved from ai/)
│   ├── paths.go            # resolveSourcePath, levenshtein, jail helpers
│   └── search.go           # searchWorkspace + caps
├── usage/                  # NEW — telemetry only, no I/O except storage iface
│   ├── usage.go            # UsageEvent ring, RPM windows, snapshot (moved)
│   └── policy.go           # cost classes, semaphore, explain cache (moved)
├── ai/                     # network ONLY (moved from services/ai/)
│   ├── service.go          # AIService, Run, failover orchestration
│   ├── backends/           # gemini.go, openai_compat.go, backend.go, registry
│   ├── prompts.go          # templates (pure strings, no I/O)
│   └── chat.go             # tool loop (calls tools.ToolHost, never implements tools)
├── services/               # SHRINKS to: filesystem.go, workspace.go, storage.go,
│                           # sqlite_storage.go (+extract_cache), ai_shim.go (re-export)
└── storage/                # optional later: move sqlite/memory impls out of
                            # services/ so services/ is interfaces + fs/workspace
```

`models.go` stays as a shim re-exporting the split files so the Wails JSON
boundary (`domain.ts` mirror) never churns during the move.

## 3. Package contracts (what may import what)

```
frontend/app.go ──► backend/main.go ──► ai ──┬──► tools ──► extract
                                              ├──► usage            (leaf)
                                              └──► storage iface    (leaf)
tools ──► extract, filesystem (Resolve jail), storage iface
extract ──► models ONLY (plus stdlib + ledongthuc/pdf)
usage ──► models ONLY
services/filesystem, workspace ──► models ONLY
```

Forbidden: `extract` importing `ai`/`tools`/`usage`; any backend package
importing `frontend/` or Wails runtime; tools executing shell/network
(shell stays rejected — see 01; `search_in_workspace` is the blessed path).

## 4. Bridge improvements (same move, small surface)

`frontend/app.go` regroups without behavior change, then gains the paged
read endpoint from 07:

* Library group: `ListLibraryTree`, `SelectFolder`, `LibraryRoot`
* Open group: `OpenFile`, `ReadRawFile`, **`ReadSourcePage`** (07: paged text)
* AI group: `StreamAIRequest` (+ legacy `RunAI`), `AIMeters`
* Storage group: `Save/Load/List*`, `SetPrimarySummary`
* Each group gets a doc comment naming its backend owner (`extract`,
  `tools`, `ai`, storage) so future methods land in the right package.

No signature changes to existing methods during the move (mechanical
relocation only — verified by unchanged `frontend/wailsjs` bindings diff,
or just `go build` + existing tests).

## 5. Execution steps (foundational first)

1. **Carve `extract/`** — move `documents.go` (dispatch only), `docx.go`,
   `pdf.go`, `blockids.go`, `*_test.go` verbatim; fix imports
   (`services.` → `extract.`); keep `services.Documents` as a type alias
   + delegating `OpenFile` so `backend/main.go`, `tools.go`, tests compile
   untouched. Land + test.
2. **Carve `tools/`** — move `tools.go`, audit, `resolveSourcePath`,
   search; `ToolHost.FS/Docs` become narrow interfaces over
   `services.Filesystem` / `extract.Extractor` (define the interfaces in
   `tools/`, implement in `services/`+`extract/`). `ai/chat.go` imports
   `tools`. Alias `ai.ToolHost = tools.ToolHost` for one release, then
   drop. Land + test.
3. **Carve `usage/`** — move `usage.go`, `policy.go`; `ai` calls
   `usage.Record/CostClass/Acquire`. No behavior change. Land + test.
4. **Split `models/`** into `document.go` / `ai.go` / `storage.go`;
   `models.go` keeps aliases. Mechanical. Land.
5. **Bridge regroup + `ReadSourcePage`** (07 §4): `App` method groups +
   new paged endpoint backed by `extract` intermediates. TS helper
   `readSourcePage` next to `openRawFile`.
6. **Delete the aliases** (`services.Documents` delegation, `ai.ToolHost`
   alias) once all callers point at the new packages. Final `go vet`,
   full tests, `tsc`, `pnpm build`.

Each step is independently shippable; stop after any step with a green tree.

## 6. cgo preparation (no new language yet — boundary only)

Heavy non-network work (PDF layout, DOCX fidelity, OCR-later) will
eventually outgrow pure Go on the low-spec target. Prepare without paying:

* `extract.Extractor` is an **interface** (`Open(abs, relKey)`,
  `Page(key, n)`, `Window(...)` in 07) — a cgo implementation later
  satisfies it with zero caller changes.
* cgo code, when it comes, lives in `extract/cgo/` behind a build tag
  (`//go:build cgo_extract`), default build stays pure-Go so Wails
  Windows cross-compilation never breaks by surprise.
* FFI rule (recorded now, enforced later): only C ABI + JSON/blocks cross
  the boundary; no Go pointers into C, no C allocations owned by Go.
* Trigger to revisit (do NOT preempt): sustained p95 extraction > 5s on
  target hardware or a format pure-Go cannot parse. Until then, stdlib +
  `ledongthuc/pdf` stay.

## 7. Done criteria

* `backend/extract`, `backend/tools`, `backend/usage` exist with package
  doc comments; `backend/services/ai/` is gone (contents moved, network-only).
* Import graph check passes: `go list -deps` on `extract|tools|usage`
  shows no `net/http`, no Wails packages.
* All existing tests pass unmodified in behavior (moved, not rewritten).
* `docs/plans/02-backend-structure.md` gets a "superseded by 06" header
  (history preserved, not rewritten).
