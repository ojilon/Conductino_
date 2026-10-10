# 05-05 Extract cache (root-anchored, LRU 200)

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: never re-extract unchanged files. Where: `backend/main.go:87 cacheKey`, `GetCachedExtract/PutCachedExtract`.
Exercise: change mtime, confirm re-extract; verify cross-folder no collision.
Done when: cache hits visible in logs.
