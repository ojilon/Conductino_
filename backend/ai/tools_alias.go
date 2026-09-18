package ai

import (
	"Conductino/backend/tools"
)

// Folder-tool compatibility aliases (plan 06 step 2).
//
// Canonical implementations live in backend/tools (registry + sandbox
// policy, no model calls). These aliases keep service.go, chat.go,
// openai_compat.go, and their tests compiling untouched during the
// migration — the tool loop already calls tools.ToolHost semantics.
// New code must import backend/tools directly. Delete this file once all
// callers point at tools (plan 06 step 6).

// ToolHost holds per-request deps for folder-scoped tools.
type ToolHost = tools.ToolHost

// ToolResult is what the model sees after a tool runs.
type ToolResult = tools.ToolResult

// PathResolver is the workspace containment boundary.
type PathResolver = tools.PathResolver

// FileOpener extracts displayable content from a Resolve-checked path.
type FileOpener = tools.FileOpener

// Proposal is one structured summary edit (insert | modify | delete).
type Proposal = tools.Proposal

// proposal is the legacy in-package spelling (chat.go); same type.
type proposal = tools.Proposal

// Tool name wire identifiers.
const (
	ToolListWorkspace      = tools.ToolListWorkspace
	ToolReadSource         = tools.ToolReadSource
	ToolReadSummary        = tools.ToolReadSummary
	ToolProposeSummaryEdit = tools.ToolProposeSummaryEdit
	ToolSearchWorkspace    = tools.ToolSearchWorkspace
)

// ToolCatalog returns the system description of available tools.
func ToolCatalog() string { return tools.ToolCatalog() }

// ParseToolCalls extracts tool invocations from model text.
func ParseToolCalls(text string) []struct {
	Name string
	Args map[string]string
} {
	return tools.ParseToolCalls(text)
}

// StripToolTags removes tool XML so the final answer is clean prose.
func StripToolTags(text string) string { return tools.StripToolTags(text) }

// AuditTool records a tool invocation (name + redacted arg only).
func AuditTool(name string, args map[string]string, ok bool) {
	tools.AuditTool(name, args, ok)
}

// auditTool is the legacy in-package spelling (chat.go); same behavior.
func auditTool(name string, args map[string]string, ok bool) {
	tools.AuditTool(name, args, ok)
}

// ToolAuditSummary counts calls per tool for the usage line.
func ToolAuditSummary() string { return tools.ToolAuditSummary() }

// toolAuditSummary is the legacy in-package spelling (service.go).
func toolAuditSummary() string { return tools.ToolAuditSummary() }
