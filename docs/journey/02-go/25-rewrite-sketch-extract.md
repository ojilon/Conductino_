# 02-25 Rewrite sketch: extract function to heavy/

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: copy-paste starter. Before: `func (d *Documents) OpenFile(...)` mixed IO+parse. After sketch:
```go
// backend/internal/heavy/parse.go
package heavy
func ParseText(name string, raw []byte) (blocks []Block, kind string) // pure, no os/net
// backend/bridge/bridge.go
type ParseRequest struct { PathToken, Root, KindHint string }
type ParseResult struct { BlocksJSON, Title, Kind string; Reason string }
```
Exercise: move txt/md parse to heavy, keep Documents as thin router.
Done when: old path re-exports + tests pass.
