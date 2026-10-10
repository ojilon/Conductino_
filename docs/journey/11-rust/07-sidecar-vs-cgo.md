# 11-07 Sidecar vs cgo vs WASM (decision)

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: choose with eyes open. cgo = fast calls, breaks Wails Windows x-compile (reject per tasks.md s4). Sidecar = process isolation, JSON bridge, easy fallback. WASM = sandbox, slower, portable.
Recommendation: sidecar first.
Exercise: prototype sidecar echo (`parse_ping`) over stdio JSON.
Done when: ping round-trips + fallback to Go works.
