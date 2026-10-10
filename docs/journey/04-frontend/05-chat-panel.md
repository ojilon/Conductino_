# 04-05 Chat panel (@doc targeting, region anchor)

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: own chat UX. Where: `AIReadingPanel.tsx`, `AIBrowsePanel.tsx`, `domain.ts:153 AIRequest mentionIds/focusedChange`.
Flow: `@Title` -> mentionIds, selected text -> anchor region, 'shorten this proposal' -> revise.
Exercise: add `@` autocomplete from open documents.
Done when: targeting never guesses (ids, not text match).
