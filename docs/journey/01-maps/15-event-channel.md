# 01-15 ai://event channel (shared bus, RequestID routing)

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: concurrent AI without crosstalk. Where: `runtime.EventsEmit(ctx,"ai://event",ev)`, `StreamAIRequest`, frontend listener routes by RequestID. Exercise: run 2 chats concurrently, verify routing. Done when: race explained.
