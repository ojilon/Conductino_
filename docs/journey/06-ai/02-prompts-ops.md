# 06-02 Prompts per operation (SEARCH/SUMMARIZE/EXPLAIN/EXPAND/MERGE/REWRITE/VERIFY/CHAT)

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: operation->prompt->encodeResult. Where: `prompts.go buildPrompt/encodeResult`, `chat.go` tool loop.
Exercise: improve one prompt, A/B the output.
Done when: each op has a documented prompt + maxTokens.
