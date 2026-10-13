package read

import (
	"fmt"
	"hash/fnv"
	"os"
	"path/filepath"
	"strings"

	"Conductino/backend/apppaths"
)

// DumpFor writes blocksJSON to <data>/temporary/extract_dump/ and returns
// the file NAME (not the path — the UI sends it back to ReadDumpFile).
// dataRoot "" resolves via apppaths; scope namespaces the name (pass the
// library root so same-named files from different folders never collide;
// "" keeps the legacy plain name). The dir is created on demand: deleting
// it by hand never crashes the app, the next dump recreates it. Dumping
// never fails an open — callers ignore the error.
func DumpFor(dataRoot, scope, relKey, blocksJSON string) (string, error) {
	if strings.TrimSpace(dataRoot) == "" {
		dataRoot = apppaths.DataRoot()
	}
	dumpDir := filepath.Join(dataRoot, "temporary", "extract_dump")
	if err := os.MkdirAll(dumpDir, 0o755); err != nil {
		return "", err
	}
	name := dumpName(scope, relKey)
	if err := os.WriteFile(filepath.Join(dumpDir, name), []byte(blocksJSON), 0o644); err != nil {
		return "", err
	}
	return name, nil
}

// DumpExtractJSON is the legacy spelling (scope ""). Kept so existing tests
// and callers keep working; new code passes a scope via DumpFor.
func DumpExtractJSON(dataRoot, relKey, blocksJSON string) (string, error) {
	if strings.TrimSpace(dataRoot) == "" {
		dataRoot = apppaths.DataRoot()
	}
	dumpDir := filepath.Join(dataRoot, "temporary", "extract_dump")
	if err := os.MkdirAll(dumpDir, 0o755); err != nil {
		return "", err
	}
	name := sanitizeDumpName(relKey) + ".json"
	out := filepath.Join(dumpDir, name)
	if err := os.WriteFile(out, []byte(blocksJSON), 0o644); err != nil {
		return "", err
	}
	return out, nil
}

// dumpName is scope-hash + readable tail: unique across folders, still
// greppable in the dump dir.
func dumpName(scope, relKey string) string {
	safe := sanitizeDumpName(relKey)
	if strings.TrimSpace(scope) == "" {
		return safe + ".json"
	}
	h := fnv.New64a()
	fmt.Fprintf(h, "%s\x00%s", strings.ToLower(strings.TrimSpace(scope)), relKey)
	if len(safe) > 60 {
		safe = safe[:60]
	}
	return fmt.Sprintf("%012x_%s.json", h.Sum64()&0xffffffffffff, safe)
}

// ReadDump returns one dump file's content. Name only (no separators, must
// end .json) so callers can never escape the dump dir.
func ReadDump(dataRoot, name string) (string, error) {
	if strings.TrimSpace(dataRoot) == "" {
		dataRoot = apppaths.DataRoot()
	}
	name = strings.TrimSpace(name)
	if name == "" || strings.ContainsAny(name, `/\`) || strings.Contains(name, "..") || !strings.HasSuffix(name, ".json") {
		return "", fmt.Errorf("bad dump name %q", name)
	}
	raw, err := os.ReadFile(filepath.Join(dataRoot, "temporary", "extract_dump", name))
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

// sanitizeDumpName flattens "notes/my paper.pdf" to "notes_my_paper.pdf".
// Keeps letters, digits, -, _, .; everything else becomes _; capped so one
// evil filename can't break the filesystem.
func sanitizeDumpName(relKey string) string {
	relKey = strings.TrimSpace(relKey)
	if relKey == "" {
		return "untitled"
	}
	flat := strings.ReplaceAll(filepath.ToSlash(relKey), "/", "_")
	var b strings.Builder
	for _, r := range flat {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_', r == '.':
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}
	name := strings.Trim(b.String(), "._")
	if name == "" {
		return "untitled"
	}
	if len(name) > 120 {
		name = name[:120]
	}
	return name
}
