package read

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Resolve turns a frontend token ("notes/a.txt") into an absolute disk
// path, refusing anything that escapes libraryRoot. Tokens are the
// nicknames the UI passes back; Go alone knows the real paths.
func Resolve(libraryRoot, token string) (string, *Fail) {
	abs, err := filepath.Abs(libraryRoot)
	if err != nil {
		return "", &Fail{ReasonParseError, err.Error()}
	}

	clean := token
	if !filepath.IsAbs(token) {
		clean = filepath.Join(abs, filepath.FromSlash(token))
	}

	clean = filepath.Clean(clean)
	rel, err := filepath.Rel(abs, clean)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", &Fail{ReasonParseError, fmt.Sprintf("path escapes workspace: %q", token)}
	}

	if _, err := os.Stat(clean); err != nil {
		if os.IsNotExist(err) {
			return "", &Fail{ReasonNotFound, err.Error()}
		}

		if os.IsPermission(err) {
			return "", &Fail{ReasonPermissionDenied, err.Error()}
		}

		return "", &Fail{ReasonParseError, err.Error()}
	}

	return clean, nil
}
