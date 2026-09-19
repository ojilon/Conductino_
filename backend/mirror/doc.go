// Package mirror owns summary .md working copies (plan 10 §2, 11 §3).
//
// The .docx stays canonical; the mirror is the text surface AI tools
// read and edit. Frontend snapshots re-anchor it: when the incoming
// snapshot hash differs from the mirror's source hash (user saved/edited),
// the mirror re-syncs from the snapshot. AI edits accumulate in the mirror
// across turns with an op log, so multi-turn refinement composes — the
// Review UI still gatekeeps every proposal before it reaches the .docx.
//
// Layout: <dir>/<summaryID>.md + .json meta {sourceHash, log}. The dir is
// backend/.work/summaries (git-ignored); 7-day sweep on startup.
//
// Import rule: extract + models + stdlib ONLY. No tools/ai imports —
// tools.ToolHost holds a *mirror.Store, never the reverse.
package mirror
