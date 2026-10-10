# 02-20 Env + dotenv parsing (readKeyFile)

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: keys without leaks. Where: `service.go:311 resolveAPIKey` (env > .ai.env > backend/.ai.env > exe-dir), `:334 readKeyFile` (skip #/blank, strip quotes, last wins, strip ` #`).
Exercise: add `AI_PRIMARY` parsing test with trailing comment.
Done when: parsing covered by tests.
