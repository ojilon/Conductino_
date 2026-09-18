package extract

import "errors"

// Windowed reads over normalized intermediates (plan 07).
//
// read_source gains optional window args (all clamped server-side):
//   pages="N" / "N-M"  — paginated formats (PDF page map; txt/md/docx
//                        synthesize ~2000-char pages so the verb is uniform)
//   offset/limit       — char window with 200-char overlap between
//                        consecutive windows (never lose a split sentence)
//
// Every window carries totals + next cursor, so "continue reading" is one
// tool call, never a guess. Not wired yet; types reserved here.

var errWindowUnimplemented = errors.New("extract: windowed reads not implemented (plan 07)")

func errNormalizeUnimplemented(blocksJSON string) error {
	if blocksJSON == "" {
		return errWindowUnimplemented
	}
	return errWindowUnimplemented
}

// Window is one slice of an intermediate with its cursors.
type Window struct {
	Text   string   // window text (with `--- page N ---` markers where known)
	Start  int      // char offset of Text in the intermediate
	End    int      // char offset of Text end
	Total  int      // total intermediate chars
	Next   int      // offset to ask for next (-1 when at end)
	Blocks []string // blockIds covered by this window
}

// WindowText slices md with offset/limit clamping. Pure function;
// implemented in plan 07 step 3.
func WindowText(md string, pageMapJSON string, offset, limit int) (Window, error) {
	return Window{}, errWindowUnimplemented
}

// SynthesizedPageSize is the char budget per virtual page for formats
// without real page breaks (txt/md/docx). One verb everywhere.
const SynthesizedPageSize = 2000

// WindowOverlap is the char overlap between consecutive windows.
const WindowOverlap = 200

// MaxWindowChars caps one window (low-spec guard, as today).
const MaxWindowChars = 6000
