# 04-08 Push heavy work to backend services

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: frontend accesses heavy stuff as services. Where: every `services/backend.ts` method = candidate.
Rule: parsing, search, file walk, extraction, diff -> Go; frontend keeps render + intent.
Exercise: move one helper (e.g. file-type detection) from TS to Go, expose via Backend.
Done when: frontend bundle shrinks + no DOM blocking on large files.
