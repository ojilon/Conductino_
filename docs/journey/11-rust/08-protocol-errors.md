# 11-08 Bridge protocol + error mapping (Rust<->Go Reason)

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: Rust errors become typed Reasons. Schema: `{version, op, payload, requestId}` -> `{version, requestId, ok, blocksJSON|reason+detail}`.
Reason taxonomy shared: unsupported/not_found/permission_denied/too_large/parse_error.
Exercise: write JSON examples for success + 5 failures.
Done when: both sides validate against schema.
