package docpipe

// The shapes of the two mirror files (content.json + meta.json) and the
// save report. Every struct here crosses the bridge to TypeScript as JSON —
// keep field names in sync with frontend/src/types/mirror.ts.

// Version of the mirror file format. Bump when fields change meaning.
const Version = 1

// ---------------------------------------------------------------------------
// content.json — the format-free document content
// ---------------------------------------------------------------------------

// MirrorSegment is one run of text with optional simple styling. This is
// deliberately tiny: the frontend renders/em/strong as italic/bold and
// never needs any OOXML run properties.
type MirrorSegment struct {
	Text   string `json:"text"`
	Em     bool   `json:"em,omitempty"`     // italic
	Strong bool   `json:"strong,omitempty"` // bold
}

// MirrorBlock is one block of content: a heading, a paragraph, a list, or a
// page-break marker (PDFs). Page says which page of the source the block
// sat on (1-based) — that is the "how the user sees it" hint for the AI and
// the anchor the frontend canvas paginates by.
type MirrorBlock struct {
	ID        string          `json:"id"`
	Type      string          `json:"type"`                // heading | paragraph | list | page
	Level     int             `json:"level,omitempty"`     // heading level 1..3
	Segments  []MirrorSegment `json:"segments"`            // inline runs (paragraph/heading)
	ListItems [][]MirrorSegment `json:"listItems,omitempty"` // one entry per bullet (type=list)
	Page      int             `json:"page,omitempty"`      // 1-based source page
}

// MirrorContent is the whole content.json file.
type MirrorContent struct {
	Version  int           `json:"version"`
	Kind     string        `json:"kind"`  // docx | pdf | text (matches extract Kind)
	Title    string        `json:"title"` // file name without extension
	Blocks   []MirrorBlock `json:"blocks"`
	EditedAt int64         `json:"editedAt,omitempty"` // Unix ms of last WriteBack (0 = never edited)
}

// ---------------------------------------------------------------------------
// meta.json — how the content sat in the source file (the rendering guide)
// ---------------------------------------------------------------------------

// FontInfo describes a font in points, like the source formats do.
type FontInfo struct {
	Family string  `json:"family"` // CSS font-family list, e.g. "Cambria, Georgia, serif"
	SizePt float64 `json:"sizePt"` // point size (frontend converts pt → px its own way)
}

// MarginsPt are page margins in points (1 inch = 72 pt).
type MarginsPt struct {
	Top    float64 `json:"top"`
	Right  float64 `json:"right"`
	Bottom float64 `json:"bottom"`
	Left   float64 `json:"left"`
}

// PageInfo is one source page's real size in points. PDF pages can differ
// per page; docx/txt/md get a single default size (see MirrorMeta.Page*).
type PageInfo struct {
	Number   int     `json:"number"` // 1-based
	WidthPt  float64 `json:"widthPt"`
	HeightPt float64 `json:"heightPt"`
}

// BlockOrigin records how one block appeared at extraction time. TextHash
// is a short sha256 of the block's plain text — WriteBack re-hashes the
// edited mirror and only the blocks whose hash differs count as "changed".
type BlockOrigin struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Page     int    `json:"page,omitempty"`
	TextHash string `json:"textHash"`
}

// MirrorMeta is the whole meta.json file: the attached metadata that tells
// the frontend (and the AI) how the source document was structured.
type MirrorMeta struct {
	Version     int    `json:"version"`
	Kind        string `json:"kind"`        // docx | pdf | text
	SourcePath  string `json:"sourcePath"`  // workspace-relative path of the source file
	SourceStamp string `json:"sourceStamp"` // "<mtimeMillis>-<sizeBytes>" of the source at extraction
	ExtractedAt int64  `json:"extractedAt"` // Unix ms when the mirror was built

	// Rendering guide (the frontend mirrors the source look its own way):
	PageWidthPt   float64    `json:"pageWidthPt"`   // default page width  (docx reads real pgSz when present)
	PageHeightPt  float64    `json:"pageHeightPt"`  // default page height
	Margins       MarginsPt  `json:"marginsPt"`     // page margins
	DefaultFont   FontInfo   `json:"defaultFont"`   // body font of the source
	Pages         []PageInfo `json:"pages,omitempty"` // per-page sizes (PDF); empty = use the defaults
	CharsPerPage  int        `json:"charsPerPage"`  // synthetic page budget for formats without pages
	Editable      bool       `json:"editable"`      // false for pdf (read + highlight only)

	// Verification data for the save bridge:
	Blocks []BlockOrigin `json:"blocks"` // one entry per block at extraction time
}

// ---------------------------------------------------------------------------
// Save report — what WriteBack verified and wrote
// ---------------------------------------------------------------------------

// WriteReport is the result of saving the mirror back into the source file.
// The lists hold block IDs so the frontend can point at what changed.
type WriteReport struct {
	Written        bool     `json:"written"`        // false = nothing had changed, file untouched
	Changed        []string `json:"changed"`        // existing blocks whose text differs
	Added          []string `json:"added"`          // new blocks (not in the original mirror)
	Removed        []string `json:"removed"`        // original blocks deleted in the editor
	UnchangedCount int      `json:"unchangedCount"` // blocks verified identical (skipped semantically)
	Kind           string   `json:"kind"`           // docx | text
	Path           string   `json:"path"`           // workspace-relative source path
}
