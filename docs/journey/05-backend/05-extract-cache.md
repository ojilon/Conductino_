# 05-05 Extract cache (root-anchored) + dump files

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: never re-extract unchanged files; every open lands on disk too. Where: `backend/main.go` OpenFile (`cacheKey=root+"\x00"+path`, `GetCachedExtract/PutCachedExtract`), `backend/read/dump.go` DumpFor (root-scoped names).
Exercise: change mtime, confirm re-extract; verify cross-folder no collision; open a file, read its dump in `.dev-data/temporary/extract_dump/`.
Done when: cache hits visible + you can state key + invalidation rule.
