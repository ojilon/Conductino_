// Package extract — dump helper (debug/temporary processing).
//
// DumpExtractJSON writes one extraction result (the blocksJSON wire string)
// to <data>/temporary/extract_dump/<safe>.json so you can inspect what the
// extractor produced BEFORE it reaches the frontend.
//
// Design rules for beginners:
//
//  1. The caller decides NOTHING about paths: pass the data root (or "" to
//     resolve via apppaths.DataRoot) and the workspace-relative relKey.
//  2. The dump dir is created on demand with MkdirAll. If you deleted
//     temporary/extract_dump by hand, the next dump recreates it — never a crash.
//  3. Dumping NEVER fails an open: callers ignore the error with `_ =`.
//     A dump problem must not break reading a real file.
//  4. Filenames are sanitized (no / \ : ..) so a weird relKey can't escape
//     the dump dir.
package extract

import (
	"os"
	"path/filepath"
	"strings"

	"Conductino/backend/apppaths"
)

// DumpExtractJSON writes blocksJSON to the extract-dump dir and returns the
// absolute file path written. dataRoot "" means "resolve via apppaths".
//
// Safe to call from Backend.OpenFile right after a successful extraction:
//
//	_ = extract.DumpExtractJSON("", path, opened.BlocksJSON)
//
// To test with the repo sandbox instead of the real install, set
// CONDUCTINO_DATA to the repo .dev-data dir (see scripts/test_dump.ps1 idea
// or just `set CONDUCTINO_DATA=.dev-data` in cmd) — then dumps land in
// .dev-data/temporary/extract_dump/.
func DumpExtractJSON(dataRoot, relKey, blocksJSON string) (string, error) {
	if strings.TrimSpace(dataRoot) == "" {
		dataRoot = apppaths.DataRoot()
	}
	dumpDir := filepath.Join(dataRoot, "temporary", "extract_dump")
	// Rule 2: internal programs may always create their own temp folders.
	if err := os.MkdirAll(dumpDir, 0o755); err != nil {
		return "", err
	}
	name := sanitizeDumpName(relKey)
	out := filepath.Join(dumpDir, name+".json")
	if err := os.WriteFile(out, []byte(blocksJSON), 0o644); err != nil {
		return "", err
	}
	return out, nil
}

// sanitizeDumpName turns "notes/my paper.pdf" into "notes_my-paper_pdf".
// Keeps letters, digits, -, _; everything else becomes _; caps length so one
// evil filename can't break the filesystem.
func sanitizeDumpName(relKey string) string {
	relKey = strings.TrimSpace(relKey)
	if relKey == "" {
		return "untitled"
	}
	// Flatten subfolders into one filename so dumps stay listable.
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
