# 01-06 OpenFile flow: resolve -> cache -> extract -> blocks

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: trace file open with cache. Non-goal: new extractors yet.
Where in code: `backend/main.go:73 OpenFile`, `:90 cacheKey=root+path`, `storage GetCachedExtract/PutCachedExtract`, `services/documents.go`, `blockids.go` (content-addressed IDs).
Concept (Go): `os.Stat` mtime+size as cache key; best-effort cache (failure falls through).
Flow: Resolve -> stat -> cache hit? return : extract -> ParseBlocksJSON shape -> cache store -> OpenedDocument{Title,BlocksJSON,PageCount,Kind,Root}.
Exercise: open same .txt twice, add log to prove second hit skips extraction.
Done when: you can state cache key + invalidation rule.
