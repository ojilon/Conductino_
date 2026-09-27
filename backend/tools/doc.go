// Package tools is the folder-scoped harness tool layer: registry, sandbox
// policy, and audit. No model calls, no network, no shell, no writes.
//
// Tools (stable wire identifiers):
//   list_workspace, read_source, read_summary, propose_summary_edit,
//   search_in_workspace
//
// Guards (plan 02 §5, plan 01 §5):
//   1. Every path joins against the CURRENT workspace root only via
//      PathResolver.Resolve (usually Filesystem.Resolve).
//   2. Reads outside the root or with disallowed extensions are rejected.
//   3. Read sizes are capped (reuses extract bounds).
//   4. Audit logs tool name + relative path only — never file content,
//      prompts, or proposal text.
//
// Import rule: this package imports models + extract (+ stdlib) ONLY.
// The model loop (backend/ai) calls tools.ToolHost — never the reverse.
// Shell stays rejected: search_in_workspace is the blessed "find in docs"
// path (no grep/rg on Windows targets, never leaves the Resolve jail).
//
// Layout (plan 07 fills search.go / paths.go as read_source grows windows):
//   tools.go — ToolHost, Dispatch, catalog, tag parsing, audit
//   paths.go — resolveSourcePath, fuzzy did-you-mean, jail helpers
//   search.go — searchWorkspace + caps
package tools
