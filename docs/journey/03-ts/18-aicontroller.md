# 03-18 aiController (runX -> provider -> dispatch)

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: own streaming glue. Where: `state/aiController.ts`: runIncludeInSummary (first-summary-wins hazard), runExplain family, chat revise.
Exercise: fix one hazard: scope include to workspace primarySummaryId.
Done when: multi-summary no longer arbitrary.
