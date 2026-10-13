package read

import (
	"fmt"
	"path/filepath"
	"strings"

	"Conductino/backend/extract"
)

// OpenFile: jail-check -> size caps -> parse -> dump JSON beside data root.
// dataRoot "" means "ask apppaths": dev.bat lands in .dev-data, the
// installed app lands in Conductino_Data. Dump never fails the open.
func OpenFile(libraryRoot, dataRoot, token string) (Doc, *Fail) {
	abs, fail := Resolve(libraryRoot, token)
	if fail != nil {
		return Doc{}, fail
	}
	root, err := filepath.Abs(libraryRoot)
	if err != nil {
		root = libraryRoot
	}
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(abs), "."))
	var d Doc
	var f *Fail
	switch ext {
	case "txt", "md":
		d, f = OpenText(abs, token)
	default:
		return Doc{}, &Fail{ReasonUnsupported, fmt.Sprintf("no extractor for .%s yet", ext)}
	}
	if f != nil {
		return Doc{}, f
	}
	d.Root = root
	_, _ = extract.DumpExtractJSON(dataRoot, token, d.Blocks)
	return d, nil
}
