# 10-01 Bridge between heavy and network parts (backend-internal)

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: heavy and network talk via narrow bridge, not spaghetti.
Design: `internal/heavy/` (pure functions, no net/http) <-> `bridge.go` (request/response structs, no business logic) <-> `internal/net/` (HTTP, streaming).
Rules: heavy never imports net/http; net never parses bytes directly; bridge types versioned.
Exercise: create `backend/internal/heavy/` + `backend/internal/net/` + `backend/bridge/bridge.go` with one moved function (e.g. block chunking).
Done when: `go build ./...` passes with new layout (old code can re-export).
