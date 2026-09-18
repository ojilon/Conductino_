package ai

import (
	"strings"
	"unicode/utf8"
)

// CharBudget is the soft character ceiling for a context pack (mirrors TS).
const CharBudget = 6000

// TruncatePack cuts a context pack to budget with an ellipsis marker.
func TruncatePack(pack string, budget int) string {
	if budget <= 0 {
		budget = CharBudget
	}
	if utf8.RuneCountInString(pack) <= budget {
		return pack
	}
	// Approximate by bytes then clean to valid UTF-8 boundary.
	if len(pack) > budget {
		pack = pack[:budget]
	}
	for len(pack) > 0 && !utf8.ValidString(pack) {
		pack = pack[:len(pack)-1]
	}
	return strings.TrimRight(pack, " \n\t") + "\n\n[…truncated…]"
}

// FormatContextSections joins labeled sections into a pack string.
func FormatContextSections(sections map[string]string, order []string) string {
	var parts []string
	for _, key := range order {
		body := strings.TrimSpace(sections[key])
		if body == "" {
			continue
		}
		parts = append(parts, "### "+key+"\n"+body)
	}
	return strings.Join(parts, "\n\n")
}
