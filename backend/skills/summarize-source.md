---
skill: summarize-source
version: 1
when: ["user asks to summarize", "AI_MERGE", "AI_CHAT + propose_summary_edit"]
tools: [list_workspace, read_source, read_summary, propose_summary_edit]
budgets: { maxToolCalls: 6, maxChars: 6000 }
---
# Summarize a source into the living summary

1. Confirm the summary file exists: `list_workspace` first. If the user
   names a file that is not listed, say so and stop — never invent content
   for a file you have not seen.
2. `read_summary` before drafting anything (harmonize, never append blindly).
   An empty summary is valid — it means "no content yet", not an error.
3. Read the source in windows (`read_source` pages/offset). Never summarize
   a window you have not read; if a span may continue past the window,
   read the next window first.
4. One `propose_summary_edit` per claim. Modify/delete are allowed, not
   just insert: redefine a stale definition, restructure a section, remove
   what a new source disproves. Cite the source path (and page where known).
5. Verify every action from its tool result — never from assumption:
   - each tool replies ok or error; an error means "not done".
   - on a path miss, use the did-you-mean list and retry once, corrected.
   - if the retry also fails, report honestly what is missing and stop.
   - never claim success ("added to Overview") without a `propose_summary_edit`
     ok result naming that file in this same turn.
6. Report briefly: file, op (insert/modify/delete), and the claim — so the
   user can find the pending proposal in Review. The user gatekeeps every
   proposal; do not re-propose an already-pending span unasked.
