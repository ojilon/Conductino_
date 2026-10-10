# 02-03 Errors + typed failures (tasks.md 1.2)

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: values-not-throws for classified failures. Non-goal: panics.
Where: `backend/main.go:73 OpenFile` returns OpenedDocument{Reason} as VALUE; `documents.go:43 ErrUnsupportedType`; `services.ReasonOf`.
Concept: `error` = unexpected; `Reason` string = expected taxonomy (unsupported/not_found/permission/too_large/parse_error).
Exercise: trigger unsupported type, observe reason in UI toast (no mock fallback).
Done when: you can add a new reason end-to-end.
