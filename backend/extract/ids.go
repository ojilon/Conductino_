package services

import (
	"fmt"
	"hash/fnv"
	"strings"
)

// Stable block IDs across re-extracts (issues 7+27).
//
// Old scheme (`txt-N`, `docx-N`) was positional: reopening a file — or
// inserting one paragraph — renumbered every later block, silently dropping
// highlights, pending changes, and chat anchors keyed by blockId.
//
// New scheme is content-addressed: FNV-1a over
// relKey + type + level + normalized text + occurrence index among identical
// siblings. An unchanged file re-extracts to identical IDs, so highlights,
// pending changes, and the extract cache survive reopening; edited blocks
// get new IDs (correct — the old anchor no longer describes them).
// Duplicate identical paragraphs disambiguate by occurrence order.
func stableBlockID(relKey, typ string, level int, text string, occurrence int) string {
	h := fnv.New64a()
	fmt.Fprintf(h, "%s\x00%s\x00%d\x00%d\x00%s",
		strings.ToLower(strings.TrimSpace(relKey)),
		typ, level, occurrence,
		normalizeBlockText(text),
	)
	return fmt.Sprintf("b-%012x", h.Sum64()&0xffffffffffff)
}

// normalizeBlockText collapses whitespace so "Same." and "Same.\n" hash and
// group identically. Occurrence counting MUST use this (not raw text), or
// duplicate paragraphs with trivially different whitespace collide.
func normalizeBlockText(text string) string {
	return strings.Join(strings.Fields(text), " ")
}
