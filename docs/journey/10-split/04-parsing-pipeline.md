# 10-04 Parsing pipeline as heavy (pdf/docx/html/txt)

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: parsers = pure bytes->blocks. Where: `documents.go`, `docx.go`, `pdf.go`, `blockids.go`.
Exercise: move parsers to `heavy/parse/` with fuzz tests on corrupt inputs.
Done when: corrupt file never panics, returns typed reason.
