# 01-04 AI data flow: runX -> provider -> stream -> dispatch

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: trace AI chat. Non-goal: prompt tuning.
Where in code: `state/aiController.ts` runX, `services/ai.ts` WailsAIProvider, `backend/services/ai/service.go:364 Run`, `:171 generateWithFailover`, `chat.go` tool loop, `models.AIEvent` (`ai://event`).
Concept: streaming = many `phase` events + one terminal `done`/`error`, correlated by RequestID.
Exercise: ask chat a question, log phase labels in order.
Done when: you can list event types and who emits each.
