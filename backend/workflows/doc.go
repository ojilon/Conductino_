// Package workflows runs named multi-step routines (plan 11 §2).
//
// A workflow is a skill with a runner: an ordered, gated routine the chat
// loop invokes via the run_workflow tool. v1 routines are READ-ONLY
// (list + bounded reads → digest report); they compose information but
// never propose, so they are inherently idempotent and safe to retry.
// Propose-capable routines arrive with the (thread, step) idempotency
// ledger — not before.
//
// Import rule: this package imports tools (+ models/stdlib) ONLY. tools
// holds a WorkflowRunner interface; the runner implements it. Budgets and
// audit reuse the tool layer (each Dispatch is guarded + audited there).
package workflows
