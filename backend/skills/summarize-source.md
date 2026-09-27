---
skill: summarize-source
version: 3
when: ["user asks to summarize", "add to summary", "AI_MERGE", "AI_CHAT + propose_summary_edit"]
tools: [run_workflow, list_workspace, read_source, read_summary, propose_summary_edit, publish_summary]
budgets: { maxToolCalls: 8, maxChars: 6000 }
---
# Summarize a source into the living summary

0. Autonomous fast path: when the user names a source file ("pick the
   introduction of X", "add X to the summary"), call `run_workflow` ONCE
   with workflow="add-to-summary", source=<path>, topic=<focus> — then
   report the queued proposals. Do NOT narrate tool calls or ask for
   confirmation first; the Review gate is the confirmation. Use the manual
   loop below only for nuanced merges (redefine/restructure across sources).
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
   All proposals in one turn reach Review — earlier ones are NOT lost.
5. Write through: after proposing, call `publish_summary` (no args) so the
   working copy lands in the mapped .docx — then report file + ops. The
   summary is openly editable by default; sources stay read-only. Never
   claim the file changed without a publish ok result in this same turn.
   (If you forget, the harness publishes automatically — but explicit is
   better: report what you published.)
5. Verify every action from its tool result — never from assumption:
   an error means "not done"; on a path miss retry once corrected, else
   report honestly and stop. Never claim success ("added to Overview")
   without an ok result in this same turn.
6. Report briefly: file, op, claim — so the user finds it in Review. The
   user gatekeeps every proposal; do not re-propose an already-pending
   span unasked.
