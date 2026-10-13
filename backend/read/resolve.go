package read

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"Conductino/backend/model"
)

// Resolve turns a frontend token ("notes/a.txt") into an absolute disk
// path, refusing anything that escapes libraryRoot. Tokens are the
// nicknames the UI passes back; Go alone knows the real paths.
func Resolve(libraryRoot, token string) (string, *model.Fail) {
	abs, err := filepath.Abs(libraryRoot)
	if err != nil {
		return "", &model.Fail{Reason: model.ReasonParseError, Detail: err.Error()}
	}
	// Tokens are root-relative nicknames: absolute paths and drive/drive-
	// rooted spellings (`\x`, `/x`, `D:\x`) are never valid tokens.
	if filepath.IsAbs(token) || strings.HasPrefix(token, "/") || strings.HasPrefix(token, `\`) {
		return "", &model.Fail{Reason: model.ReasonParseError, Detail: fmt.Sprintf("path escapes workspace: %q", token)}
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
	if _, err := os.Stat(clean); err != nil {
		return "", model.FailOf(err)
	}
	return clean, nil
}
