# 05-02 Library owns the curated tree

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: raw FS vs curated UI tree. Where: `backend/list/library.go` Library (SetRoot/Root/Tree), `backend/main.go` adoptWorkspaceID + SetLibraryRoot, primary summary in SetPrimarySummary.
Exercise: log tree before/after SetLibraryRoot.
Done when: you can explain why root state lives in ONE struct.
