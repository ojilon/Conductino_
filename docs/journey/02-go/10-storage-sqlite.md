# 02-10 Storage: memory vs SQLite + extract_cache

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: understand persistence. Non-goal: schema redesign yet.
Where: `services/storage.go`, `sqlite_storage.go` (pure-Go modernc.org/sqlite, no cgo), `extract_cache` (path+mtime+size, 200-entry LRU).
Exercise: inspect DB file location, query extract_cache after opening a file.
Done when: you can state engine + cache key from memory.
