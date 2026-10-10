# 11-02 Go<->Rust bridge (how the two languages work in backend)

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: narrow FFI. Options: cgo + staticlib, or sidecar process over stdio/Unix socket/HTTP loopback, or WASM sandbox.
Recommended start: sidecar (no cgo, keeps Wails Windows cross-compile simple; tasks.md s4 already rejects cgo for same reason).
Bridge types mirror `bridge.go` (JSON or protobuf); versioned; heavy never calls network.
Exercise: spec bridge for one function: `parse_pdf(bytes)->blocks JSON` with error taxonomy matching ReasonOf.
Done when: bridge spec reviewed + test harness planned.
