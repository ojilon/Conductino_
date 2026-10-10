# 03-14 Discriminated unions (kind/status/reason)

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: exhaustive switches. Where: `Document.kind`, `ChangeType/Status`, `OpenFailureReason`, `AppMode`.
Exercise: add a compile-time exhaustive check (`never` default) to one switch.
Done when: tsc catches missing cases.
