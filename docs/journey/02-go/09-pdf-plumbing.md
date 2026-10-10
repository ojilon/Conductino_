# 02-09 PDF plumbing: ReadRawFile + canvas leaf

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: separate bytes vs text. Non-goal: extraction quality.
Where: `backend/main.go:124 maxRawFileBytes=30MiB`, `:131 ReadRawFile` base64, `frontend PdfView.tsx` canvas + text layer.
Concept: renderers needing originals get bytes; chat/tools use extracted blocks.
Exercise: open a PDF, confirm canvas renders + chat can reference it.
Done when: you can explain the 30MiB cap.
