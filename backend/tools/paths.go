package tools

import (
	"path/filepath"
	"sort"
	"strings"

	"Conductino/backend/models"
)

// Path resolution + jail helpers. All future path-taking tools reuse
// resolveSourcePath — never a second matcher (plan 07 §1c).

// resolveSourcePath maps a model-typed path to the canonical tree path.
// Returns ("", nil) when the tree is unavailable (caller tries direct) or
// ("", suggestions) when nothing matches. Matching order: exact
// case-insensitive → unique stem (extension-insensitive) → normalized
// fuzzy containment, capped at 3 suggestions.
func (h *ToolHost) resolveSourcePath(rel string) (string, []string) {
	if h.FS == nil {
		return "", nil
	}
	root, err := h.FS.ListRoot()
	if err != nil || root == nil {
		return "", nil
	}
	var files []string
	var walk func(n *models.FileTreeNode)
	walk = func(n *models.FileTreeNode) {
		if n == nil {
			return
		}
		if n.Kind != "folder" {
			p := n.Path
			if p == "" {
				p = n.Label
			}
			if p != "" {
				files = append(files, p)
			}
		}
		for i := range n.Children {
			walk(&n.Children[i])
		}
	}
	walk(root)
	for _, f := range files {
		if strings.EqualFold(f, rel) {
			return f, nil
		}
	}
	stem := strings.ToLower(strings.TrimSuffix(rel, filepath.Ext(rel)))
	var stemHits []string
	for _, f := range files {
		if strings.ToLower(strings.TrimSuffix(f, filepath.Ext(f))) == stem {
			stemHits = append(stemHits, f)
		}
	}
	if len(stemHits) == 1 {
		return stemHits[0], nil
	}
	if len(stemHits) > 1 {
		return "", stemHits[:min(3, len(stemHits))]
	}
	norm := func(s string) string {
		var b strings.Builder
		for _, r := range strings.ToLower(s) {
			if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
				b.WriteRune(r)
			}
		}
		return b.String()
	}
	// Stems: the extension ("x" vs "x.docx") must not count as distance.
	nq := norm(strings.TrimSuffix(filepath.Base(rel), filepath.Ext(rel)))
	type scored struct {
		path string
		d    int
	}
	var ranked []scored
	for _, f := range files {
		nf := norm(strings.TrimSuffix(filepath.Base(f), filepath.Ext(f)))
		if nq == "" || nf == "" {
			continue
		}
		d := levenshtein(nq, nf)
		// Containment is an exact-enough hit regardless of length gap.
		if strings.Contains(nf, nq) || strings.Contains(nq, nf) {
			d = 0
		}
		allow := len(nq) / 6
		if allow < 2 {
			allow = 2
		}
		if ll := len(nf); ll/6 > allow {
			allow = ll / 6
		}
		if d <= allow {
			ranked = append(ranked, scored{f, d})
		}
	}
	sort.Slice(ranked, func(i, j int) bool { return ranked[i].d < ranked[j].d })
	var fuzzy []string
	for i := 0; i < len(ranked) && i < 3; i++ {
		fuzzy = append(fuzzy, ranked[i].path)
	}
	if len(fuzzy) > 0 {
		return "", fuzzy
	}
	return "", nil
}

// levenshtein is rune-wise edit distance for did-you-mean ranking.
// Inputs are short normalized stems, so the O(n·m) table is trivial.
func levenshtein(a, b string) int {
	ar, br := []rune(a), []rune(b)
	if len(ar) == 0 {
		return len(br)
	}
	if len(br) == 0 {
		return len(ar)
	}
	prev := make([]int, len(br)+1)
	for j := range prev {
		prev[j] = j
	}
	for i, ca := range ar {
		cur := make([]int, len(br)+1)
		cur[0] = i + 1
		for j, cb := range br {
			cost := 0
			if ca != cb {
				cost = 1
			}
			del, ins, sub := prev[j+1]+1, cur[j]+1, prev[j]+cost
			cur[j+1] = del
			if ins < cur[j+1] {
				cur[j+1] = ins
			}
			if sub < cur[j+1] {
				cur[j+1] = sub
			}
		}
		prev = cur
	}
	return prev[len(br)]
}
