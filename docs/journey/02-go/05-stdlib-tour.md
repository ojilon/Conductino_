# 02-05 Stdlib tour: os, path/filepath, archive/zip, encoding/xml, net/http

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: know what needs no dependency. Non-goal: new deps.
Where: `filesystem.go` os/exec/filepath, `docx.go` archive/zip+encoding/xml, `ai/gemini.go` net/http stdlib client (no vendor SDK).
Exercise: find 3 stdlib uses, explain why each avoids a dep (low-spec Windows story).
Done when: you can justify 'stdlib first' for any new need.
