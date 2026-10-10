# 01-08 Selection -> toolbar -> Ask/Go deeper/Include/Save

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: trace text selection. Non-goal: redesigning toolbar.
Where in code: `types/domain.ts:439 TextSelection` (blockId+offsets+x/y), `DocumentView.tsx` selection handler, `aiController.ts:122 runIncludeInSummary` (first-summary-wins hazard per tasks.md s2).
Exercise: select a sentence, log the TextSelection object.
Done when: you can explain block-anchored vs range-precise selections.
