# 02-23 Table-driven tests (your first Go test)

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: write one. Pattern: `tests := []struct{name,in,want}{...}; for _,tt:=range tests { t.Run(...) }`.
Where: `ai/policy_test.go`, `blockids_test.go` as models.
Exercise: table-test Resolve jail (5 cases).
Done when: `go test -race ./backend/...` green.
