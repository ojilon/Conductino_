# 04-03 Reader tabs + stale-tab guard (tasks.md 1.1)

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: fix stale tabs on folder switch. Where: `ReaderMode.tsx:32 tree state`, `:139 metadata.path`, `domain.ts:309 DocumentMetadata rootPath`.
Options: (a) invalidate on switch (close/mark stale) vs (b) tag with opening root + explicit 'belongs to another folder'.
Exercise: implement (b): compare rootPath vs current root before resolve/reveal/locate.
Done when: switching folders never silently shows wrong file.
