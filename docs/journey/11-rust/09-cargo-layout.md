# 11-09 Cargo layout for pilot (one crate)

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: minimal Rust. Layout: `native/parse/Cargo.toml`, `src/lib.rs` (pure) + `src/bin/sidecar.rs` (stdio loop). No network deps.
Exercise: `cargo new native/parse`, implement `chunk_text` + test.
Done when: `cargo test` green.
