# 02-22 base64 bytes for canvas (ReadRawFile)

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: bytes without files. Where: `backend/main.go:131 ReadRawFile` (Resolve jail + 30MiB cap + base64), `backend.ts:289 openRawFile` atob->Uint8Array.
Exercise: open oversized PDF, confirm refusal (never partial stream).
Done when: cap + encoding explained blind.
