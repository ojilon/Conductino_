# 01-06 OpenFile flow: resolve -> cache -> extract -> dump -> read back

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: trace file open with cache + dump. Non-goal: new extractors yet.
Where in code: `backend/main.go` OpenFile, `cacheKey=root+"\x00"+path`, store `GetCachedExtract/PutCachedExtract`, `backend/read/open.go` dispatch, `dump.go` DumpFor (root-scoped names), `frontend/app.go` ReadDumpFile, `ReaderMode.tsx` openFile.
Concept (Go): `os.Stat` mtime+size as cache key; best-effort cache (failure falls through); content ALWAYS lands in `temporary/extract_dump/`, UI reads it back by name.
Flow: Resolve -> stat -> cache hit? dump+return : extract -> cache store -> dump -> Doc{Title, Dump, PageCount, Kind, Root} -> ReadDumpFile -> parse -> render.
Exercise: open same .txt twice (second hit skips extraction); delete a dump mid-session, reopen, watch it recreated.
Done when: you can state cache key + invalidation rule + why inline JSON is cleared.
