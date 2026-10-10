# 06-00 Provider boundary (AIProvider, never vendor SDK)

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: app speaks AIProvider. Where: `domain.ts:243 AIProvider`, `services/ai.ts`, `backend/services/ai/service.go:34 Service`.
Exercise: implement a stub provider returning canned phases.
Done when: swap takes <30 min.
