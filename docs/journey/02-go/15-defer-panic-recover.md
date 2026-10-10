# 02-15 defer/panic/recover (when NOT to panic)

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: errors as values. Where: cache best-effort `_ = PutCachedExtract` (never fail open on cache error).
Exercise: find a panic risk, convert to typed error.
Done when: `go vet` clean + no panics on corrupt inputs.
