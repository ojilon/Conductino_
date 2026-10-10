# 02-21 Time as Unix millis (INTEGER in SQLite)

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: consistent timestamps. Where: `backend/main.go:239 doc.UpdatedAt=time.Now().UnixMilli()`, `SaveChange CreatedAt`, storage Upsert defaults.
Exercise: find a zero-timestamp bug, fix with defaulting.
Done when: all writes stamp millis.
