// Package skills is RESERVED for plan 08 §1 (SKILL.md instruction layer).
//
// Tools are code; skills are prose: versioned Markdown files that teach the
// harness how to use tools, perform tasks, and run providers in parallel.
// The prompt assembler will inject at most 2 matched skill excerpts (~1500
// chars total) instead of one ever-growing system prompt.
//
// Discovery layers (08 §1.2): bundled `backend/skills/*.md` (embedded,
// versioned in git) < workspace `<root>/.conductino/skills/*.md` < user
// app-data `Conductino/skills/*.md`. No skill content exists yet — this
// package holds the reservation so 07 (windowed reading) lands first and
// skills build on stable tool verbs. Do not invent skill files here early.
package skills
