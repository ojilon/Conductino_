# 01-09 Change proposal flow: propose -> review -> accept/reject/revise

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: understand DocumentChange lifecycle. Non-goal: real diff engine.
Where in code: `types/domain.ts:372 DocumentChange` (insert/modify/delete, pending/accepted/rejected), reducer `change.propose/decide/revise`, `SummaryDocumentView.tsx` inline review, `slateAdapter.ts` toSlate/fromSlate.
Flow: AI proposes -> pending block/card rendered -> accept commits / reject reverts / revise asks model for new proposal.
Exercise: propose Include-in-summary, accept then reject, watch blocks.
Done when: you can state what each status renders as.
