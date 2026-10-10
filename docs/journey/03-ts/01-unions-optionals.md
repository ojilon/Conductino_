# 03-01 Unions + optionals (SourceKind, ChangeType, reason?)

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: model states precisely. Where: `SourceKind`, `ChangeType`, `ChangeStatus`, `OpenedFile.reason?`.
Exercise: add a new SourceKind, find all switches that must handle it (tsc will error — good).
Done when: exhaustive handling feels natural.
