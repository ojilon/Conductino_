# 05-04 Storage engines (memory vs SQLite)

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: choose engine deliberately. Where: `backend/store/storage.go`, `sqlite_storage.go` (modernc.org/sqlite pure-Go), `OpenDefaultStorage`, records in `backend/model/storage.go`.
Exercise: insert + restart + verify persistence; force fallback, observe.
Done when: you know where DB file lives + engine string in Settings.
