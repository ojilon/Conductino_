# 02-02 Interfaces (FilesystemService, StorageService, AIService)

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: program to boundaries. Non-goal: generics.
Where: `filesystem.go:21 FilesystemService`, `storage.go StorageService`, `ai/service.go:34 Service`.
Concept: interface = method set; implicit implementation; enables mock vs sqlite swap.
Exercise: list which methods Backend calls on storage vs docs.
Done when: you can extract a new interface for documents.
