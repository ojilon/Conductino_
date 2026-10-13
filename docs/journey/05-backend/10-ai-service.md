# 05-10 backend/ai/service.go (Run + failover entry)

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: own Run. Where: `backend/ai/service.go` Run (phase/done/error, RequestID echo, OpSearch honest error, OpChat tool loop, one-shot cache), generateWithFailover. Narrow seams: `NewWithTools(tools.PathResolver, tools.FileOpener)` — AI takes tiny interfaces, never concrete structs.
Exercise: add a per-op maxTokens table.
Done when: Run narratable blind.
