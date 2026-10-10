# 10-03 Shell + internal tooling (heavy side)

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: own exec safely. Where: `filesystem.go:111 ShowContainingFolder` exec per-OS, future shell tools.
Concept (Go): `os/exec`, `runtime.GOOS`, never shell-out with unsanitized input (Resolve first).
Exercise: add a `heavy/shell.go` wrapper with allowlist + tests.
Done when: all exec goes through wrapper.
