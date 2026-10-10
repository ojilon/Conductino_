# 02-04 Packages + modules (Conductino/...)

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: never break imports. Non-goal: publishing.
Where: `go.mod` module Conductino; `Conductino/backend/models`, `Conductino/backend/services`.
Concept: package path = directory; exported = Capitalized.
Exercise: move a helper between packages, fix imports.
Done when: `go build ./...` passes after move.
