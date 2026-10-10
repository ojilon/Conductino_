# 11-04 Safety for parsing (untrusted bytes)

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: corrupt files never panic/escape. Rust: `Result`, bounds checks, `zip` crate limits (ZIP-bomb/XXE guards like `ieshan/go-ooxml`).
Exercise: fuzz pilot with corpus + random mutations.
Done when: fuzzer runs in CI.
