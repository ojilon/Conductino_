package read

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"Conductino/backend/model"
)

// OpenFile is the whole UI loop in one function:
// jail-check -> size caps -> parse -> dump. dataRoot "" means "ask
// apppaths": dev.bat lands dumps in .dev-data, the installed app in
// Conductino_Data. Every file lands in the dump dir with a root-scoped
// name, so the UI reads content back from disk (ReadDumpFile) instead of
// holding every open file in memory.
func OpenFile(libraryRoot, dataRoot, token string) (model.Doc, *model.Fail) {
	abs, fail := Resolve(libraryRoot, token)
	if fail != nil {
		return model.Doc{}, fail
	}
	if info, err := os.Stat(abs); err == nil && info.IsDir() {
		return model.Doc{}, &model.Fail{Reason: model.ReasonParseError, Detail: fmt.Sprintf("not a file: %s", abs)}
	}
	root, err := filepath.Abs(libraryRoot)
	if err != nil {
		root = libraryRoot
	}
	d, fail := openAbs(abs, token)
	if fail != nil {
		return model.Doc{}, fail
	}
	d.Root = root
	if name, err := DumpFor(dataRoot, root, token, d.BlocksJSON); err == nil {
		d.Dump = name
	}
	return d, nil
}

// OpenDirect opens an already Resolve-checked absolute path (AI tools path:
// the host jailed it before calling). Same caps and dump, no jail of its own.
func OpenDirect(absPath, relKey, dataRoot string) (model.Doc, *model.Fail) {
	if relKey == "" {
		relKey = filepath.Base(absPath)
	}
	d, fail := openAbs(absPath, relKey)
	if fail != nil {
		return model.Doc{}, fail
	}
	if name, err := DumpFor(dataRoot, filepath.Dir(absPath), relKey, d.BlocksJSON); err == nil {
		d.Dump = name
	}
	return d, nil
}

// openAbs dispatches on the lowercased, dot-stripped extension.
func openAbs(absPath, relKey string) (model.Doc, *model.Fail) {
	switch strings.ToLower(strings.TrimPrefix(filepath.Ext(absPath), ".")) {
	case "txt", "md", "markdown":
		return OpenText(absPath, relKey)
	case "pdf":
		return OpenPDF(absPath, relKey)
	case "docx":
		return OpenDocx(absPath, relKey)
	default:
		ext := strings.TrimPrefix(filepath.Ext(absPath), ".")
		return model.Doc{}, &model.Fail{Reason: model.ReasonUnsupported, Detail: fmt.Sprintf("no extractor for .%s yet", ext)}
	}
}

// CacheProbe stats a resolved path for the extract cache (mtime+size key).
// Returns ok=false when the file can't be keyed (caller extracts anyway).
func CacheProbe(absPath string) (mtime, size int64, ok bool) {
	info, err := os.Stat(absPath)
	if err != nil || info.IsDir() {
		return 0, 0, false
	}
	return info.ModTime().UnixMilli(), info.Size(), true
}
