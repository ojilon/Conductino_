package docpipe

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"Conductino/backend/apppaths"
	"Conductino/backend/extract"
)

// This file builds the mirror pair (content.json + meta.json) from an
// extracted document, saves/loads it under the app-data cache, and renders
// a page-oriented description the AI can read.

// ---------------------------------------------------------------------------
// Where mirrors live
// ---------------------------------------------------------------------------

// KeyFor turns (workspace root, relative path) into a short stable folder
// name. Two different folders can contain "notes.txt" — the root in the
// hash keeps their mirrors apart.
func KeyFor(root, relPath string) string {
	sum := sha256.Sum256([]byte(root + "\x00" + filepath.ToSlash(relPath)))
	return fmt.Sprintf("%x", sum[:8])
}

// DirFor is the mirror folder of one document:
// <app-data>/cache/extract/mirrors/<key>/ holding content.json + meta.json.
func DirFor(key string) string {
	return filepath.Join(apppaths.CacheDir(), "mirrors", key)
}

// ContentPath / MetaPath are the two file names inside a mirror folder.
func ContentPath(dir string) string { return filepath.Join(dir, "content.json") }
func MetaPath(dir string) string    { return filepath.Join(dir, "meta.json") }

// ---------------------------------------------------------------------------
// Build: wire blocks → mirror pair (the "process" step)
// ---------------------------------------------------------------------------

// Build converts the extractor's wire blocks JSON into the format-free
// mirror content plus the metadata guide. It reads nothing from disk except
// a Stat of the source file (for the freshness stamp) and, for .docx, a
// best-effort peek at the real page size. Pure enough to call on every open.
//
// kind is the extractor kind ("docx" | "pdf" | "text"), relPath the
// workspace-relative token, absPath the resolved file, pageCount what the
// extractor reported, blocksJSON the {blocks:[…]} wire payload.
func Build(kind, title, relPath, absPath string, pageCount int, blocksJSON string) (MirrorContent, MirrorMeta, error) {
	// 1. Decode the wire blocks (same shape every extractor arm emits).
	blocks, err := extract.ParseBlocksJSON(blocksJSON)
	if err != nil {
		return MirrorContent{}, MirrorMeta{}, fmt.Errorf("docpipe: bad blocks json: %w", err)
	}

	// 2. Decide the page budget and rendering defaults per kind.
	charsPerPage := extract.SynthesizedPageSize
	pageW, pageH := 612.0, 792.0 // US Letter in points — the fallback everywhere
	font := FontInfo{Family: "Georgia, 'Times New Roman', serif", SizePt: 11}
	switch kind {
	case "pdf":
		font = FontInfo{Family: "Georgia, 'Times New Roman', serif", SizePt: 10}
	case "text":
		if ext := strings.ToLower(filepath.Ext(relPath)); ext == ".md" || ext == ".markdown" {
			font = FontInfo{Family: "Georgia, 'Times New Roman', serif", SizePt: 11}
		} else {
			font = FontInfo{Family: "ui-monospace, 'Cascadia Code', Consolas, monospace", SizePt: 10.5}
		}
	case "docx":
		font = FontInfo{Family: "Cambria, Georgia, serif", SizePt: 11}
		// Real Word page size when the file carries one (twips → pt: /20).
		if w, h, ok := readDocxPageSize(absPath); ok {
			pageW, pageH = w, h
		}
	}

	// 3. Walk the blocks, assigning each one to a page.
	//    PDFs carry explicit "page" marker blocks; docx/txt/md have no real
	//    pages, so we synthesize them with the char budget — the same rule
	//    the frontend canvas uses, so both sides agree on "page 3".
	mirrorBlocks := make([]MirrorBlock, 0, len(blocks))
	origins := make([]BlockOrigin, 0, len(blocks))
	page := 1
	used := 0
	for _, b := range blocks {
		if b.Type == "page" {
			// Page-break anchor: keep it in the content (frontend uses it as
			// a break) and move to the next page number.
			mirrorBlocks = append(mirrorBlocks, MirrorBlock{ID: b.ID, Type: "page", Page: page})
			origins = append(origins, BlockOrigin{ID: b.ID, Type: "page", Page: page, TextHash: HashText("")})
			page++
			used = 0
			continue
		}
		text := blockText(b.Segments, b.ListItems)
		if kind != "pdf" {
			// Synthetic pagination: start a new page when the budget is full.
			if used+len(text) > charsPerPage && used > 0 {
				page++
				used = 0
			}
			used += len(text)
		}
		mirrorBlocks = append(mirrorBlocks, MirrorBlock{
			ID:        b.ID,
			Type:      b.Type,
			Level:     b.Level,
			Segments:  convertSegments(b.Segments),
			ListItems: convertListItems(b.ListItems),
			Page:      page,
		})
		origins = append(origins, BlockOrigin{ID: b.ID, Type: b.Type, Page: page, TextHash: HashText(text)})
	}
	if pageCount < page {
		pageCount = page
	}

	// 4. Per-page size list (only PDFs really have one page size per page).
	var pages []PageInfo
	if kind == "pdf" {
		for p := 1; p <= pageCount; p++ {
			pages = append(pages, PageInfo{Number: p, WidthPt: pageW, HeightPt: pageH})
		}
	}

	// 5. Freshness stamp of the source file: mtime + size. WriteBack compares
	//    it before saving so an external edit never gets silently overwritten.
	stamp := ""
	if info, serr := os.Stat(absPath); serr == nil {
		stamp = Stamp(info.ModTime().UnixMilli(), info.Size())
	}

	content := MirrorContent{
		Version: Version,
		Kind:    kind,
		Title:   title,
		Blocks:  mirrorBlocks,
	}
	meta := MirrorMeta{
		Version:      Version,
		Kind:         kind,
		SourcePath:   filepath.ToSlash(relPath),
		SourceStamp:  stamp,
		ExtractedAt:  time.Now().UnixMilli(),
		PageWidthPt:  pageW,
		PageHeightPt: pageH,
		Margins:      MarginsPt{Top: 72, Right: 72, Bottom: 72, Left: 72},
		DefaultFont:  font,
		Pages:        pages,
		CharsPerPage: charsPerPage,
		Editable:     kind == "docx" || kind == "text",
		Blocks:       origins,
	}
	return content, meta, nil
}

// convertSegments maps the extractor's segments onto mirror segments.
func convertSegments(in []extract.DocxSegment) []MirrorSegment {
	out := make([]MirrorSegment, 0, len(in))
	for _, s := range in {
		out = append(out, MirrorSegment{Text: s.Text, Em: s.Em, Strong: s.Strong})
	}
	return out
}

// convertListItems maps list rows (one []Segment per bullet).
func convertListItems(in [][]extract.DocxSegment) [][]MirrorSegment {
	if len(in) == 0 {
		return nil
	}
	out := make([][]MirrorSegment, 0, len(in))
	for _, row := range in {
		out = append(out, convertSegments(row))
	}
	return out
}

// blockText is the plain text of one wire block (list items joined by \n).
func blockText(segments []extract.DocxSegment, listItems [][]extract.DocxSegment) string {
	var sb strings.Builder
	if len(listItems) > 0 {
		for i, row := range listItems {
			if i > 0 {
				sb.WriteString("\n")
			}
			for _, s := range row {
				sb.WriteString(s.Text)
			}
		}
		return sb.String()
	}
	for _, s := range segments {
		sb.WriteString(s.Text)
	}
	return sb.String()
}

// MirrorBlockText is the plain text of one MIRROR block (exported: the
// frontend save path and WriteBack verification both hash exactly this).
func MirrorBlockText(b MirrorBlock) string {
	var sb strings.Builder
	if len(b.ListItems) > 0 {
		for i, row := range b.ListItems {
			if i > 0 {
				sb.WriteString("\n")
			}
			for _, s := range row {
				sb.WriteString(s.Text)
			}
		}
		return sb.String()
	}
	for _, s := range b.Segments {
		sb.WriteString(s.Text)
	}
	return sb.String()
}

// HashText is the short content hash used to verify what changed (16 hex
// chars of sha256 — collisions are irrelevant at document scale).
func HashText(text string) string {
	sum := sha256.Sum256([]byte(text))
	return fmt.Sprintf("%x", sum[:8])
}

// Stamp formats the source freshness stamp (mtime millis + size bytes).
func Stamp(mtimeMillis, size int64) string {
	return strconv.FormatInt(mtimeMillis, 10) + "-" + strconv.FormatInt(size, 10)
}

// pgSzRe finds Word's page-size element, e.g. <w:pgSz w:w="12240" w:h="15840"/>.
// Values are in twentieths of a point (twips); we divide by 20 for points.
var pgSzRe = regexp.MustCompile(`pgSz[^>]*?w:w="(\d+)"[^>]*?w:h="(\d+)"`)

// readDocxPageSize peeks into word/document.xml for the real page size.
// Best-effort: any problem (no zip, no sectPr, odd values) → ok=false and
// the caller keeps the Letter default.
func readDocxPageSize(absPath string) (wPt, hPt float64, ok bool) {
	zr, err := zip.OpenReader(absPath)
	if err != nil {
		return 0, 0, false
	}
	defer zr.Close()
	for _, f := range zr.File {
		if f.Name != "word/document.xml" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return 0, 0, false
		}
		defer rc.Close()
		// Bounded read: page size lives near the end (sectPr) but files can
		// be big — 4 MiB is plenty for the XML head/tail we regex over.
		raw, err := io.ReadAll(io.LimitReader(rc, 4<<20))
		if err != nil {
			return 0, 0, false
		}
		m := pgSzRe.FindSubmatch(raw)
		if m == nil {
			return 0, 0, false
		}
		wTwips, err1 := strconv.ParseFloat(string(m[1]), 64)
		hTwips, err2 := strconv.ParseFloat(string(m[2]), 64)
		if err1 != nil || err2 != nil || wTwips <= 0 || hTwips <= 0 {
			return 0, 0, false
		}
		return wTwips / 20.0, hTwips / 20.0, true
	}
	return 0, 0, false
}

// ---------------------------------------------------------------------------
// Save / Load
// ---------------------------------------------------------------------------

// SaveFor writes the mirror pair for one document under the app-data cache.
// Creating folders and writing are best-effort but errors are returned so
// callers can log; a failed mirror save must never fail an open.
func SaveFor(root, relPath string, content MirrorContent, meta MirrorMeta) (dir string, err error) {
	dir = DirFor(KeyFor(root, relPath))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return dir, err
	}
	cj, err := json.MarshalIndent(content, "", "  ")
	if err != nil {
		return dir, err
	}
	mj, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return dir, err
	}
	if err := os.WriteFile(ContentPath(dir), cj, 0o644); err != nil {
		return dir, err
	}
	if err := os.WriteFile(MetaPath(dir), mj, 0o644); err != nil {
		return dir, err
	}
	return dir, nil
}

// LoadFor reads the mirror pair of one document. ok=false means "no mirror
// yet" (the file was never opened successfully) — not a hard error.
func LoadFor(root, relPath string) (content MirrorContent, meta MirrorMeta, ok bool) {
	dir := DirFor(KeyFor(root, relPath))
	cj, err := os.ReadFile(ContentPath(dir))
	if err != nil {
		return content, meta, false
	}
	mj, err := os.ReadFile(MetaPath(dir))
	if err != nil {
		return content, meta, false
	}
	if err := json.Unmarshal(cj, &content); err != nil {
		return content, meta, false
	}
	if err := json.Unmarshal(mj, &meta); err != nil {
		return content, meta, false
	}
	return content, meta, true
}

// SaveMeta rewrites just meta.json (WriteBack refreshes hashes + stamp).
func SaveMeta(root, relPath string, meta MirrorMeta) error {
	dir := DirFor(KeyFor(root, relPath))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	mj, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(MetaPath(dir), mj, 0o644)
}

// ---------------------------------------------------------------------------
// The AI view — "what the user is seeing"
// ---------------------------------------------------------------------------

// DescribeForAI renders the mirror as a page-oriented plain-text view:
// header lines from the metadata (kind, pages, page size, font) followed by
// "--- page N ---" sections of block text. This is what read_mirror returns,
// so the AI sees the document the way the canvas shows it to the user.
func DescribeForAI(content MirrorContent, meta MirrorMeta, maxChars int) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "Mirror of %q\n", meta.SourcePath)
	fmt.Fprintf(&sb, "kind=%s pages=%d pageSize=%.0fx%.0fpt font=%s %.1fpt editable=%t\n",
		meta.Kind, meta.PageCount(), meta.PageWidthPt, meta.PageHeightPt,
		meta.DefaultFont.Family, meta.DefaultFont.SizePt, meta.Editable)
	if content.EditedAt > 0 {
		fmt.Fprintf(&sb, "last edited in-app: %s\n", time.UnixMilli(content.EditedAt).Format(time.RFC3339))
	}
	sb.WriteString("\n")

	page := 0
	for _, b := range content.Blocks {
		if b.Type == "page" {
			continue // the marker itself prints nothing
		}
		p := b.Page
		if p < 1 {
			p = 1
		}
		if p != page {
			page = p
			fmt.Fprintf(&sb, "\n--- page %d ---\n", page)
		}
		text := MirrorBlockText(b)
		switch b.Type {
		case "heading":
			sb.WriteString(strings.Repeat("#", max(min(b.Level, 3), 1)) + " " + text + "\n")
		case "list":
			for _, row := range b.ListItems {
				var line strings.Builder
				for _, s := range row {
					line.WriteString(s.Text)
				}
				sb.WriteString("- " + line.String() + "\n")
			}
		default:
			sb.WriteString(text + "\n")
		}
		if maxChars > 0 && sb.Len() > maxChars {
			sb.WriteString("\n… [mirror truncated]\n")
			break
		}
	}
	return sb.String()
}

// PageCount is the number of pages the mirror describes: the highest block
// page number (PDFs count their markers; docx/txt the synthetic budget).
func (m MirrorMeta) PageCount() int {
	pages := 1
	for _, b := range m.Blocks {
		if b.Page > pages {
			pages = b.Page
		}
	}
	if len(m.Pages) > pages {
		pages = len(m.Pages)
	}
	return pages
}
