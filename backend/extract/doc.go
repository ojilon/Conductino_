// Package extract owns all file-to-blocks extraction: pure Go, no network,
// no Wails, no model calls.
//
// Layout:
//   extract.go  — Extractor dispatch (ext → arm) + OpenError/ReasonOf contract
//   text.go     — txt/md arm (blank-line paragraphs, # headings)
//   docx.go     — stdlib ZIP+OOXML arm (+ DOCX writer for summaries)
//   pdf.go      — text-layer arm (ledongthuc/pdf, pure Go)
//   ids.go      — stable content-addressed block IDs
//   normalize.go — (plan 07) canonical md intermediate + page/block map
//   page.go     — (plan 07) windowed reads over intermediates
//
// Import rule: this package imports models + stdlib (+ ledongthuc/pdf) ONLY.
// It must never import backend/ai, backend/tools, backend/usage, or frontend.
// Offline work (extract, tools, storage) stays testable without API keys;
// verify with: go list -deps ./backend/extract | findstr /i "http wails".
package extract
