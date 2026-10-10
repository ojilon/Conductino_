# 02-17 Slices + maps (Children, Record mirrors)

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: own tree + cache containers. Where: `models.FileTreeNode.Children []FileTreeNode`, `storage extract_cache 200-entry LRU`, TS `Record<ID,Document>`.
Concept: slice = ordered list; map = key lookup. Go nil slice vs empty matters for JSON (`omitempty`).
Exercise: add a test that empty Children serializes as missing, not null.
Done when: JSON shapes match TS expectations.
