# 09-00 API economy (quality vs speed vs cost)

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: pick provider per task. Where: `ai/service.go:48 defaultGeminiModel=gemini-2.5-flash`, `NewGroqBackend (openai/gpt-oss-20b)`, `NewOpenRouterBackend (qwen3 free router)`.
Exercise: run same prompt on 3 providers, log quality/latency/cost.
Done when: you have a per-operation provider table.
