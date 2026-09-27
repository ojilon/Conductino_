// Package models mirrors the frontend domain types (src/types/domain.ts).
// Keep the two in sync manually — or generate one from the other — so the
// Wails JSON boundary stays unambiguous.
//
// Split (plan 06 step 4; this file is the doc shim, wire-stable):
//   document.go — Source, FileTreeNode, OpenedDocument, OpenFailureReason
//   ai.go       — AIRequest, AIEvent, ChatTurn, FocusedChange, AIOperation
//   storage.go  — Workspace/Document/Change/Chat/CachedExtract records
package models
