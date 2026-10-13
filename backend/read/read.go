// Package read owns everything about file content: jail-checked opening,
// text/pdf/docx parsing, block intermediates, context packs, and the
// extract_dump files the UI reads back. One package, plain functions, two
// shared structs from backend/model (Doc, Fail).
//
// Layout:
//   open.go      — OpenFile (UI path: jail+caps+parse+dump) + OpenDirect (tools path)
//   resolve.go   — token → absolute path jail (from services/filesystem.go)
//   text.go      — txt/md arm + stable IDs (from extract/text.go + ids.go)
//   pdf.go       — pdf text-layer arm (from extract/pdf.go)
//   docx.go      — docx read arm (from extract/docx.go; writer lives in save)
//   normalize.go — blocks ↔ markdown intermediate (from extract/normalize.go)
//   context.go   — budgeted AI context packs (from extract/context.go)
//   dump.go      — extract_dump files (from extract/dump.go)
//
// Rule: read imports model + apppaths + stdlib (+ ledongthuc/pdf) ONLY.
// Never ai/tools/usage/frontend — offline work stays testable without keys.
package read

// PageBudget mirrors the frontend paginateBlocks budget (DocumentView.tsx):
// ~2000 chars per virtual page for formats without real page breaks.
const PageBudget = 2000
