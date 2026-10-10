# 10-02 What stays close to network (do NOT split)

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: avoid over-splitting. Stays: prompt assembly with live summary, tool-loop orchestration, failover decisions, usage metering.
Where: `ai/chat.go`, `service.go:364 Run`, `prompts.go`.
Exercise: for each AI file, mark split vs stay + why.
Done when: no split breaks streaming correlation (RequestID).
