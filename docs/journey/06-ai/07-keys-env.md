# 06-07 Keys live in Go only (.ai.env)

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: never leak keys to JS. Where: `resolveAPIKey`, `readKeyFile` (env wins, last line wins, ` #` stripped), exe-dir fallback.
Exercise: rotate a key, restart, verify.
Done when: `git status` never shows .ai.env.
