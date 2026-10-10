# 10-05 Goroutines (middle stage, weeks 5-6)

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: concurrency for extraction/indexing. Non-goal: channels everywhere day 1.
Concepts: `go f()`, `sync.WaitGroup`, `errgroup`, `semaphore` (see `policy.go` semaphore), bounded workers for low-spec.
Where: extract many files, build index, warm cache.
Exercise: parallelize folder walk with worker pool (cap = NumCPU/2 on Celeron), benchmark.
Done when: speedup measured + no data race (`go test -race`).
