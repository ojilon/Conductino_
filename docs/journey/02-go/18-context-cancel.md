# 02-18 context.Context (cancel AI + IO)

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: propagate cancel. Where: `ai/service.go:364 Run(ctx,...)`, `Backend.RunAI(ctx,req,sink)`, `http.Client{Timeout:90s}`.
Concept: ctx = deadline/cancel signal; always pass as first arg; never store in struct.
Exercise: cancel a chat mid-stream, verify prompt return + no goroutine leak.
Done when: every blocking call respects ctx.
