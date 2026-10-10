# 02-24 go vet + go build (what each catches)

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: gate every change. `go vet` = suspicious code (printf, copies); `go build` = compile. Run from root.
Exercise: introduce a vet warning deliberately, observe, fix.
Done when: pre-commit hook runs both.
