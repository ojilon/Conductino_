# 10-00 Heavy vs network split (from day 1)

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: separate separable non-network work from network-bound Go from the START, even before Rust.
Heavy/non-network: shell execution, internal tooling, document parsing (pdf/docx/txt/md/html), file walk, hashing, caching, text chunking.
Network: AI HTTP clients, failover, streaming events, update checks.
Some parts STAY with network (prompt building needing live context, tool loop orchestration) — documented per file below.
Where: `backend/services/*` today mixes both; your job is to draw the line file by file.
Exercise: label every backend file H (heavy), N (network), B (both) + reason.
Done when: map exists + bridge sketched.
