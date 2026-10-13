// Package save writes blocks back to disk. Mirror image of read: blocks
// JSON in -> file out, same jail idea, same caps, same dump dir. Parent
// folders are created (MkdirAll); existing files overwritten in place —
// that overwrite IS the update path, and it never touches neighbours.
// The .docx writer (for summaries) lives in docx.go.
package save

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"Conductino/backend/model"
	"Conductino/backend/read"
)

// MaxWriteBytes mirrors read.MaxTextBytes: render first, measure, THEN
// write — never half-write a giant file.
const MaxWriteBytes = 2 << 20

// SaveText renders blocksJSON and writes it to token under libraryRoot.
// Returns the absolute path written.
func SaveText(libraryRoot, token, blocksJSON string) (string, *model.Fail) {
	abs, fail := resolveTarget(libraryRoot, token)
	if fail != nil {
		return "", fail
	}
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(abs), "."))
	if ext != "txt" && ext != "md" {
		return "", &model.Fail{Reason: model.ReasonUnsupported, Detail: fmt.Sprintf("save supports .txt/.md, got .%s", ext)}
	}
	blocks, err := model.ParseBlocks(blocksJSON)
	if err != nil {
		return "", &model.Fail{Reason: model.ReasonParseError, Detail: err.Error()}
	}
	var parts []string
	for _, blk := range blocks {
		var s string
		switch blk.Type {
		case "heading":
			lvl := blk.Level
			if lvl < 1 || lvl > 3 {
				lvl = 2
			}
			s = strings.Repeat("#", lvl) + " " + join(blk.Segments)
		case "list":
			rows := make([]string, 0, len(blk.ListItems))
			for _, item := range blk.ListItems {
				rows = append(rows, "- "+join(item))
			}
			s = strings.Join(rows, "\n")
		case "page":
			continue // render anchor, not content — contributes nothing
		default:
			s = join(blk.Segments)
		}
		if s == "" {
			continue // the parser never stores blanks; keep files clean
		}
		parts = append(parts, s)
	}
	out := strings.Join(parts, "\n\n")
	if len(out) > MaxWriteBytes {
		return "", &model.Fail{Reason: model.ReasonTooLarge, Detail: fmt.Sprintf("rendered text too large (%d bytes)", len(out))}
	}
	if err := os.WriteFile(abs, []byte(out), 0o644); err != nil {
		return "", model.FailOf(err)
	}
	_, _ = read.DumpFor("", libraryRoot, token, blocksJSON) // saved snapshot, same dump dir
	return abs, nil
}

func join(segs []model.Segment) string {
	var s strings.Builder
	for _, g := range segs {
		s.WriteString(g.Text)
	}
	return s.String()
}

// resolveTarget jails the token like read.Resolve, but the file may NOT
// exist yet — so we MkdirAll the PARENT instead of Statting the file.
func resolveTarget(libraryRoot, token string) (string, *model.Fail) {
	token = strings.TrimSpace(token)
	if token == "" {
		return "", &model.Fail{Reason: model.ReasonParseError, Detail: "empty path"}
	}
	// Same token grammar as read.Resolve: root-relative nicknames only.
	if filepath.IsAbs(token) || strings.HasPrefix(token, "/") || strings.HasPrefix(token, `\`) {
		return "", &model.Fail{Reason: model.ReasonParseError, Detail: fmt.Sprintf("path escapes workspace: %q", token)}
	}
	abs, err := filepath.Abs(libraryRoot)
	if err != nil {
		return "", &model.Fail{Reason: model.ReasonParseError, Detail: err.Error()}
	}
	clean := token
	if !filepath.IsAbs(token) {
		clean = filepath.Join(abs, filepath.FromSlash(token))
	}
	clean = filepath.Clean(clean)
	rel, err := filepath.Rel(abs, clean)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", &model.Fail{Reason: model.ReasonParseError, Detail: fmt.Sprintf("path escapes workspace: %q", token)}
	}
	if info, err := os.Stat(clean); err == nil && info.IsDir() {
		return "", &model.Fail{Reason: model.ReasonParseError, Detail: fmt.Sprintf("not a file: %s", clean)}
	}
	if err := os.MkdirAll(filepath.Dir(clean), 0o755); err != nil {
		return "", &model.Fail{Reason: model.ReasonParseError, Detail: err.Error()}
	}
	return clean, nil
}
