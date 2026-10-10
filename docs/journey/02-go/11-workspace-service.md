# 02-11 Workspace service (library tree owner)

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: curated tree vs raw FS. Non-goal: multi-folder.
Where: `services/workspace.go` composes Filesystem, owns LibraryTree, SetLibraryRoot, primary summary.
Exercise: trace SetLibraryRoot -> persist last_library_root -> Init restore.
Done when: you can explain why Workspace composes Filesystem.
