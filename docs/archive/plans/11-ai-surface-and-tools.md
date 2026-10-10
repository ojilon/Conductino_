# 11 — AI surface, tools, safety, Python helpers

**Status:** ACTIVE  
**Depends on:** release-prep (local storage), 09 (intermediates), 10 (skills)  
**Audience:** Agents extending `backend/services/ai/`

## Goal

Keep the existing Go AI surface (multi-provider, tool loop, phase events, proposals) as the brain and security boundary.  
Extend it so the AI can work robustly with intermediate documents, show clear “thinking” steps, call controlled helpers, and stay strictly inside the workspace + app-data jail.

## Non-negotiable rules

1. **Go owns orchestration.** Python (or any other helper language) is only for small, temporary helper tools.
2. **No direct host shell.** No unrestricted `os/exec` for the model. Any shell-like behaviour goes through a sandboxed tool (future E2B / local container) or a tightly constrained helper.
3. **Every path through `PathResolver.Resolve`.** Intermediate paths and app-data paths still obey containment rules.
4. **Proposals only for user-visible edits.** `propose_summary_edit` (and future propose tools) create `DocumentChange` records; the user accepts/rejects.
5. **Audit is redacted.** Tool audit logs name + safe arg summary only — never full file content or prompts.

## Existing surface (do not break)

- Providers + failover (`service.go`, `openai_compat.go`, …)
- Tool catalog + dispatch (`tools.go`)
- Multi-round chat loop (`chat.go`)
- Phase events already emitted on `ai://event`
- Usage meters + in-memory tool audit ring

## Extensions required by the new plans

### A. Tools that understand intermediates

- Prefer reading/writing intermediate content + meta (see 09).
- Keep `read_source` / `read_summary` working; internally they may resolve to intermediate files when present.
- Add or extend:
  - windowed / range read for large intermediates
  - `document_problem` or `update_guidance` (writes short notes only under allowed locations)
  - optional `run_python_helper` (see below)

### B. Thinking / phase events for the frontend

Extend the event stream (same channel) so the UI can show a modern “Thinking” panel:

- `phase` / `thinking` — label + optional short text
- `tool_start` / `tool_end` — name, redacted args, duration, ok/error
- `token_usage` — in/out (exact or estimate)
- `provider` — which backend answered
- `guidance` — when a skill or self-written note was injected

Frontend renders these under each assistant turn. Backend also persists a subset into `ai_log` / `tool_audit` (SQLite) once local storage is ready.

### C. Python helpers (temporary)

Only for cases that are currently painful in pure Go (complex layout, OCR prep, etc.).

Contract:

- Registered by name inside `ToolHost`.
- Go resolves paths, enforces timeout and output size, sets cwd to a safe directory.
- Helper receives only resolved paths or temp copies; no network by default.
- Result returns as a normal tool result string.
- Scripts live under a controlled tree (or a venv created on first run). Document that they are optional and will be rewritten in Go over time.

Do **not** let the model invent arbitrary Python that runs on the host.

### D. Token accounting and multi-API logic

Stay in Go (`usage.go`, policy, cost classes). Skills may *describe* the preferred policy; the code remains the authority.

## Safety checklist for any new tool

- [ ] Path arguments go through Resolve (or app-data helper that still cannot escape).
- [ ] Writes only to intermediates, `.conductino/`, or app-data locations defined in release-prep.
- [ ] Output size and time are capped.
- [ ] Audit entry is written with redacted args.
- [ ] Failure is returned as a tool result the model can see; no silent success.

## Out of scope

- Replacing the Go tool loop with LangChain / LlamaIndex / etc.
- Host-level unrestricted shell
- Automatic application of AI edits without user gate
