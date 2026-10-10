# 02-28 SQLite without cgo (modernc.org/sqlite)

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: why pure-Go driver. Constraint: cgo breaks Wails Windows x-compile + low-spec deploy. Exercise: open DB with sqlite3 CLI, `.schema`. Done when: tables listed blind.
