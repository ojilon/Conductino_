package docpipe

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"Conductino/backend/extract"
)

// WriteBack is the SAVE BRIDGE: it takes the edited mirror content (the
// JSON the frontend canvas produced), verifies WHICH blocks actually
// changed against meta.json, and writes the source file back in its own
// format (.docx via OOXML, .txt/.md as plain text). PDFs are read-only.
//
// How verification works (the "only the parts that changed" step):
//
//  1. meta.json carries a TextHash per block from extraction time.
//  2. We re-hash every edited block; same hash = untouched, different hash
//     = changed. Blocks missing from the mirror were removed; unknown IDs
//     were added.
//  3. The source file's mtime+size stamp is compared too: if the file was
//     edited outside the app since it was opened, we refuse to save rather
//     than silently overwrite someone else's change.
//
// Note on .docx: the container is a zip, so the file is always rewritten
// whole — but only after the diff check, and the report tells the caller
// exactly which blocks differed. Unchanged blocks are carried through
// byte-identical in text.
func WriteBack(root, relPath, absPath, contentJSON string) (WriteReport, error) {
	report := WriteReport{Path: filepath.ToSlash(relPath)}

	// 1. Load the mirror metadata (built when the document was opened).
	_, meta, ok := LoadFor(root, relPath)
	if !ok {
		return report, fmt.Errorf("no mirror metadata for %q — open the document in the app first", relPath)
	}
	if meta.Kind == "pdf" {
		return report, fmt.Errorf("PDF files are read-only — highlight them, or copy text into a summary .docx")
	}
	report.Kind = meta.Kind

	// 2. Decode the edited content coming from the frontend.
	var content MirrorContent
	if err := json.Unmarshal([]byte(contentJSON), &content); err != nil {
		return report, fmt.Errorf("bad mirror content json: %w", err)
	}

	// 3. Freshness check: refuse to overwrite external edits.
	if info, err := os.Stat(absPath); err == nil && meta.SourceStamp != "" {
		now := Stamp(info.ModTime().UnixMilli(), info.Size())
		if now != meta.SourceStamp {
			return report, fmt.Errorf("the file changed on disk since it was opened — close and reopen %q, then save again", relPath)
		}
	}

	// 4. Verify what changed, block by block, using the stored hashes.
	originHash := map[string]string{}
	originSeen := map[string]bool{}
	for _, o := range meta.Blocks {
		originHash[o.ID] = o.TextHash
	}
	present := map[string]bool{}
	for _, b := range content.Blocks {
		present[b.ID] = true
		h := HashText(MirrorBlockText(b))
		if old, known := originHash[b.ID]; !known {
			report.Added = append(report.Added, b.ID)
		} else if old != h {
			report.Changed = append(report.Changed, b.ID)
		} else {
			report.UnchangedCount++
		}
	}
	for id := range originHash {
		if !present[id] {
			report.Removed = append(report.Removed, id)
		}
	}

	nothingChanged := len(report.Added) == 0 && len(report.Changed) == 0 && len(report.Removed) == 0
	if nothingChanged {
		// Honest no-op: the file on disk already matches the editor.
		report.Written = false
		return report, nil
	}

	// 5. Write the source file in its own format.
	switch meta.Kind {
	case "docx":
		blocks := mirrorToDocxBlocks(content.Blocks)
		if err := extract.WriteDOCX(absPath, blocks); err != nil {
			return report, fmt.Errorf("write docx: %w", err)
		}
	case "text":
		text := mirrorToPlainText(content.Blocks, strings.EqualFold(filepath.Ext(relPath), ".md") ||
			strings.EqualFold(filepath.Ext(relPath), ".markdown"))
		if err := os.WriteFile(absPath, []byte(text), 0o644); err != nil {
			return report, fmt.Errorf("write text: %w", err)
		}
	default:
		return report, fmt.Errorf("write-back is not supported for kind %q", meta.Kind)
	}
	report.Written = true

	// 6. Refresh the mirror: new hashes, new stamp, editedAt — so the next
	//    save verifies against THIS version and the AI reads fresh metadata.
	content.Version = Version
	content.Kind = meta.Kind
	content.EditedAt = time.Now().UnixMilli()
	newOrigins := make([]BlockOrigin, 0, len(content.Blocks))
	for _, b := range content.Blocks {
		newOrigins = append(newOrigins, BlockOrigin{
			ID: b.ID, Type: b.Type, Page: b.Page, TextHash: HashText(MirrorBlockText(b)),
		})
	}
	meta.Blocks = newOrigins
	meta.ExtractedAt = content.EditedAt
	if info, err := os.Stat(absPath); err == nil {
		meta.SourceStamp = Stamp(info.ModTime().UnixMilli(), info.Size())
	}
	if _, err := SaveFor(root, relPath, content, meta); err != nil {
		// The source file is already saved; a mirror-refresh failure only
		// means the next save re-verifies against older hashes. Report it.
		return report, fmt.Errorf("saved the file, but could not refresh the mirror: %w", err)
	}
	return report, nil
}

// UpdateContent refreshes ONLY content.json from the live editor state —
// no source-file write, no verification. The frontend calls this (debounced)
// while the user types so the mirror always shows what the user sees RIGHT
// NOW: the AI's read_mirror tool reads these files, so it watches edits
// live. meta.json stays untouched (it keeps describing the SOURCE file —
// that is what makes WriteBack's change verification trustworthy).
func UpdateContent(root, relPath, contentJSON string) error {
	dir := DirFor(KeyFor(root, relPath))
	if _, err := os.Stat(MetaPath(dir)); err != nil {
		return fmt.Errorf("no mirror for %q — open the document in the app first", relPath)
	}
	var content MirrorContent
	if err := json.Unmarshal([]byte(contentJSON), &content); err != nil {
		return fmt.Errorf("bad mirror content json: %w", err)
	}
	content.Version = Version
	raw, err := json.MarshalIndent(content, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(ContentPath(dir), raw, 0o644)
}

// mirrorToDocxBlocks converts edited mirror blocks back into the extractor's
// canonical DOCX blocks (the writer in backend/extract speaks this shape).
// "page" markers are dropped — a .docx has no literal page blocks.
func mirrorToDocxBlocks(blocks []MirrorBlock) []extract.DocxBlock {
	out := make([]extract.DocxBlock, 0, len(blocks))
	for _, b := range blocks {
		if b.Type == "page" {
			continue
		}
		db := extract.DocxBlock{ID: b.ID, Type: b.Type, Level: b.Level}
		db.Segments = make([]extract.DocxSegment, 0, len(b.Segments))
		for _, s := range b.Segments {
			db.Segments = append(db.Segments, extract.DocxSegment{Text: s.Text, Em: s.Em, Strong: s.Strong})
		}
		for _, row := range b.ListItems {
			item := make([]extract.DocxSegment, 0, len(row))
			for _, s := range row {
				item = append(item, extract.DocxSegment{Text: s.Text, Em: s.Em, Strong: s.Strong})
			}
			db.ListItems = append(db.ListItems, item)
		}
		out = append(out, db)
	}
	if len(out) == 0 {
		// Never write an empty <w:body>: one blank paragraph keeps Word happy.
		out = append(out, extract.DocxBlock{ID: "empty", Type: "paragraph", Segments: []extract.DocxSegment{{Text: ""}}})
	}
	return out
}

// mirrorToPlainText flattens mirror blocks into the text/markdown file body.
// markdown=true keeps headings as "# Heading" lines so .md files round-trip.
func mirrorToPlainText(blocks []MirrorBlock, markdown bool) string {
	var sb strings.Builder
	first := true
	for _, b := range blocks {
		if b.Type == "page" {
			continue
		}
		if !first {
			sb.WriteString("\n\n")
		}
		first = false
		switch b.Type {
		case "heading":
			level := b.Level
			if level < 1 || level > 3 {
				level = 1
			}
			if markdown {
				sb.WriteString(strings.Repeat("#", level) + " ")
			}
			sb.WriteString(MirrorBlockText(b))
		case "list":
			for i, row := range b.ListItems {
				if i > 0 {
					sb.WriteString("\n")
				}
				var line strings.Builder
				for _, s := range row {
					line.WriteString(s.Text)
				}
				sb.WriteString("- " + line.String())
			}
		default:
			sb.WriteString(MirrorBlockText(b))
		}
	}
	sb.WriteString("\n")
	return sb.String()
}
