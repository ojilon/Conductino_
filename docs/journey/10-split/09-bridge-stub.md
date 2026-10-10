# 10-09 Bridge stub you can compile (copy-paste)

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: start the split today. Stub:
```go
// backend/bridge/bridge.go
package bridge
type ParseRequest struct { PathToken, Root, KindHint string; Raw []byte }
type ParseResult struct { BlocksJSON, Title, Kind, Reason, Detail string }
// backend/internal/heavy/parse_text.go: func ParseText(...) (ParseResult)
// backend/internal/net/... : never imports heavy/parse directly, only bridge types
```
Exercise: commit stub + move one function; keep old export as wrapper.
Done when: `go build ./...` green with both paths.
