# Phase 6 notes

SQLite system of record is designed and unit-tested locally. Full file push may follow; until then:

## Design
- Driver: `modernc.org/sqlite` v1.34.5 (pure-Go, no CGO)
- DB path: app data `Conductino/conductino.db` or `$CONDUCTINO_DB`
- Tables: settings, workspaces, documents, document_changes, chat_threads, chat_messages
- `OpenDefaultStorage` falls back to memory if open fails
- Shared storage: Backend + Workspace; Init restores `last_library_root`
- `RunAI` fills SummaryContent from `GetPrimarySummary` when chat omits it
- Wails: SaveDocument, LoadDocument, SaveChange, chat CRUD, SetPrimarySummary, LoadPrimarySummary

## Why this unblocks tools
Once primary summary is stored with `blocks_json` and mapped via `primary_summary_id`, `read_summary` no longer depends on a frontend snapshot. Extractors (Phase 8+) still required for PDF/DOCX `read_source`.

## Apply
See conversation artifacts `phase6/` or re-request Phase 6 push when GitHub API is healthy.
