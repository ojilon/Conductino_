# 11-00 Why Rust (safety + non-GC for heavy parts)

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: understand trade. Non-goal: rewriting everything.
Why: memory safety without GC pauses, fearless concurrency for parsing/indexing, single binary via FFI/staticlib.
When: AFTER profiling proves Go GC/throughput is the bottleneck (see 10-07), advanced stage (weeks 7-8+).
Exercise: write down 3 Rust candidates (e.g. pdf text extraction, docx parse, file indexer) + why each is heavy + non-network.
Done when: candidates ranked by profile data.
