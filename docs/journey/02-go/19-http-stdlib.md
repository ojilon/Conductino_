# 02-19 net/http stdlib client (no vendor SDKs)

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: own model HTTP. Where: `ai/gemini.go generateContent`, `openai_compat.go` Groq/OpenRouter, no extra deps (low-spec story).
Exercise: log request/response sizes for one op; add a timeout test.
Done when: you can add a header/retry without docs.
