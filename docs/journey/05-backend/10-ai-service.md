# 05-10 backend/services/ai/service.go (Run + failover entry)

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: own Run. Where: `:364 Run` (phase/done/error, RequestID echo, OpSearch honest error, OpChat tool loop, one-shot cache), `:171 generateWithFailover`.
Exercise: add a per-op maxTokens table.
Done when: Run narratable blind.
