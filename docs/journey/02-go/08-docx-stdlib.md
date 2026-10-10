# 02-08 DOCX via stdlib ZIP+OOXML

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: understand current docx arm. Non-goal: full fidelity.
Where: `services/docx.go`, `documents.go` dispatch, `blockids.go`.
Concept: DOCX = zip containing word/document.xml paragraphs/runs -> blocks/segments.
Exercise: open a .docx, map one paragraph to blocks JSON.
Done when: you can state what fidelity is lost (tables/images/numbering).
