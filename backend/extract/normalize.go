package extract

// Normalized intermediate (plan 07) — one canonical text per file version.
//
// Every successfully extracted document will also materialize:
//   mdText    — Markdown-ish view (headings as `#`, lists as `- `,
//               tables as `a | b`, pages as `--- page N ---` markers)
//   pageMap   — JSON: page → first blockId, blockId → page/offset
//
// Storage: two columns on extract_cache (md_text, page_map_json), same
// root-anchored key (path + mtime + size), same LRU. No new table.
//
// Both AI tools and a future "view as text" toggle read this intermediate —
// never two divergent renderings. Not wired yet; signatures reserved here so
// callers (tools/read_source, App.ReadSourcePage) have a stable target.

// NormalizeBlocksJSON converts wire blocks ({blocks:[...]}) to the canonical
// intermediate. Pure function; implemented in plan 07 step 2.
func NormalizeBlocksJSON(blocksJSON string) (mdText, pageMapJSON string, err error) {
	return "", "", errNormalizeUnimplemented(blocksJSON)
}

// PageMapEntry is one row of the page/block map (JSON-serializable).
type PageMapEntry struct {
	Page    int    `json:"page"`
	BlockID string `json:"blockId"`
	Offset  int    `json:"offset"`
}
