# 10-10 Goroutine sketch (bounded bulk extract)

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: copy-paste worker pool.
```go
sem := make(chan struct{}, runtime.NumCPU()/2)
var wg sync.WaitGroup; for _, f := range files { wg.Add(1); go func(f string){ defer wg.Done(); sem<-struct{}{}; defer func(){<-sem}(); /* extract with ctx */ }(f) }
wg.Wait()
```
Exercise: apply to cache warm-up with ctx cancel.
Done when: `go test -race` green + speedup noted.
