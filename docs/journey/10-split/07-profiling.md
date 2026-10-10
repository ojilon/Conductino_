# 10-07 Profiling before Rust (prove what is heavy)

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: numbers before new language. Tools: `go test -bench`, `pprof`, `go vet`, manual timing on 4GB Celeron.
Exercise: profile opening 100 files, list top 3 hotspots.
Done when: Rust candidates chosen by data, not gut.
