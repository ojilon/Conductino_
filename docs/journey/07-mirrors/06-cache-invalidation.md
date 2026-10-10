# 07-06 Cache invalidation rules (when mirrors die)

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: stale never silent. Rules: folder switch -> tag-and-warn (tasks 1.1); mtime/size change -> re-extract; delete -> typed not_found.
Exercise: write the invalidation table for tree/extract/mirror.
Done when: table posted in code comments.
