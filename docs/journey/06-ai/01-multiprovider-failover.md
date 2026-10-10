# 06-01 Multi-provider failover (Gemini/Groq/OpenRouter)

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: survive 429s/dead tiers. Where: `service.go:171 generateWithFailover`, `gemini.go`, `openai_compat.go`, `backend.go`, `AI_MODE/AI_PRIMARY`.
Rules: AI_PRIMARY picks first; auto->failover when secondary keyed; failover tries next on ANY error.
Exercise: kill primary key, confirm secondary serves + phase label shows switch.
Done when: you can narrate selection rules.
