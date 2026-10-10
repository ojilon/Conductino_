# 11-03 Rust ownership/borrowing (via your pilot)

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: learn by porting one pure function (e.g. block chunking) to Rust.
Concepts: ownership, borrowing, lifetimes, Result<T,E> (~ Go value+error but enforced).
Exercise: port `chunkText` to Rust, property-test against Go version.
Done when: outputs byte-identical on corpus.
