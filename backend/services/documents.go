package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	// NOTE (import fix): was `github.com/lumen/desktop/backend/models`, which
	// does not match the `Conductino` module in go.mod. Fixed to the real
	// module path so `go build` resolves the local models package.
	"Conductino/backend/models"
)

// DocumentService owns format extraction: turning raw files (PDF/DOCX/
// HTML/TXT) into the structured DocumentModel the frontend renders.
//
// FUTURE INTEGRATION (per format — see docs/document-rendering.md):
//
//	PDF   → pdfium / unipdf to extract text+layout, or ship pdf.js in the
//	       webview and keep extraction client-side
//	DOCX  → baliance/gooxml or unidoc/unioffice to parse .docx into blocks
//	HTML  → golang.org/x/net/html → block/segment tree
//	TXT   → line split
//
// Phase 8: .docx is handled via stdlib ZIP+OOXML (docx.go). PDF remains
// unsupported until a pure-Go arm lands.
type DocumentService interface {
	Extract(ctx context.Context, source models.Source) (blocksJSON string, pageCount int, err error)
	// OpenFile reads a resolved absolute path and returns displayable
	// content. relKey is the workspace-relative token (or base name when
	// unknown) — it seeds stable content-addressed block IDs so reopening
	// an unchanged file yields identical IDs (see blockids.go). absPath
	// must already be resolved + containment-checked (Filesystem.Resolve).
	OpenFile(absPath, relKey string) (models.OpenedDocument, error)
}

// ErrUnsupportedType signals "no extractor for this extension yet" (PDF,
// … until their arms land). DOCX is supported as of Phase 8.
var ErrUnsupportedType = fmt.Errorf("unsupported file type for extraction")

// OpenError is a classified open failure (tasks.md 1.2): the Reason lets the
// UI decide fallback-vs-error without string-matching error text.
type OpenError struct {
	Reason models.OpenFailureReason
	Detail string
	Err    error
}

func (e *OpenError) Error() string {
	if e.Detail != "" {
		return fmt.Sprintf("%s: %s", string(e.Reason), e.Detail)
	}
	return string(e.Reason)
}

func (e *OpenError) Unwrap() error { return e.Err }

// ReasonOf classifies any open error into a wire reason + detail. Unknown
// errors become parse_error so the UI shows an honest failure, never a
// fabricated document.
func ReasonOf(err error) (models.OpenFailureReason, string) {
	var oe *OpenError
	if errors.As(err, &oe) {
		return oe.Reason, oe.Detail
	}
	if errors.Is(err, ErrUnsupportedType) {
		return models.ReasonUnsupported, err.Error()
	}
	if os.IsNotExist(err) {
		return models.ReasonNotFound, err.Error()
	}
	if os.IsPermission(err) {
		return models.ReasonPermissionDenied, err.Error()
	}
	return models.ReasonParseError, err.Error()
}

type Documents struct{}

func NewDocuments() *Documents { return &Documents{} }

func (d *Documents) Extract(_ context.Context, source models.Source) (string, int, error) {
	// Mock: one paragraph block containing the source abstract.
	blocks := `{"blocks":[{"id":"mock-1","type":"paragraph","segments":[{"text":"` + source.Abstract + `"}]}]}`
	return blocks, 1, nil
}

// OpenFile dispatches on the (lowercased, dot-stripped) extension.
func (d *Documents) OpenFile(absPath, relKey string) (models.OpenedDocument, error) {
	if relKey == "" {
		relKey = filepath.Base(absPath)
	}
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(absPath), "."))
	switch ext {
	case "txt", "md":
		// Trivial arm: plain text reads directly, no parser dependency.
		// Proves the click → path → content pipe end-to-end.
		return openTextFile(absPath, relKey)
	case "docx":
		// Phase 8: stdlib ZIP+OOXML text extraction (paragraph/run + bold/italic).
		return openDocxFile(absPath, relKey)
	default:
		return models.OpenedDocument{}, &OpenError{
			Reason: models.ReasonUnsupported,
			Detail: fmt.Sprintf("no extractor for .%s yet", ext),
			Err:    fmt.Errorf("%w: .%s", ErrUnsupportedType, ext),
		}
	}
}

// textBlock mirrors the TS DocumentBlock shape just enough for paragraphs.
// Marshalled with encoding/json — never string-concatenated — so arbitrary
// file content cannot break the wire format.
type textBlock struct {
	ID       string        `json:"id"`
	Type     string        `json:"type"`
	Level    int           `json:"level,omitempty"`
	Segments []textSegment `json:"segments"`
}

type textSegment struct {
	Text string `json:"text"`
}

const (
	maxTextBytes  = 2 << 20 // 2 MiB — refuse anything bigger, it's not prose
	maxTextBlocks = 1000    // cap block count so one file can't flood the UI
)

func openTextFile(absPath, relKey string) (models.OpenedDocument, error) {
	info, err := os.Stat(absPath)
	if err != nil {
		reason, _ := ReasonOf(err)
		// ReasonOf falls back to parse_error for unclassified os errors,
		// which is correct here; keep not_found / permission_denied exact.
		return models.OpenedDocument{}, &OpenError{Reason: reason, Detail: err.Error(), Err: err}
	}
	if info.IsDir() {
		err := fmt.Errorf("not a file: %s", absPath)
		return models.OpenedDocument{}, &OpenError{Reason: models.ReasonParseError, Detail: err.Error(), Err: err}
	}
	if info.Size() > maxTextBytes {
		// Size-gated BEFORE reading: oversized files are refused without
		// ever loading them into memory (issue 23).
		err := fmt.Errorf("text file too large (%d bytes)", info.Size())
		return models.OpenedDocument{}, &OpenError{Reason: models.ReasonTooLarge, Detail: err.Error(), Err: err}
	}
	f, err := os.Open(absPath)
	if err != nil {
		reason, _ := ReasonOf(err)
		return models.OpenedDocument{}, &OpenError{Reason: reason, Detail: err.Error(), Err: err}
	}
	defer f.Close()
	// Bounded read: LimitReader caps RSS at maxTextBytes+1 even if the file
	// grew between Stat and Open (TOCTOU).
	raw, err := io.ReadAll(io.LimitReader(f, maxTextBytes+1))
	if err != nil {
		return models.OpenedDocument{}, &OpenError{Reason: models.ReasonParseError, Detail: err.Error(), Err: err}
	}
	if len(raw) > maxTextBytes {
		err := fmt.Errorf("text file too large (%d bytes)", len(raw))
		return models.OpenedDocument{}, &OpenError{Reason: models.ReasonTooLarge, Detail: err.Error(), Err: err}
	}
	// Blank-line separated paragraphs; single newlines stay inside a block.
	paras := strings.Split(strings.ReplaceAll(string(raw), "\r\n", "\n"), "\n\n")
	blocks := make([]textBlock, 0, len(paras))
	seen := map[string]int{} // content-address occurrence among identical siblings
	for _, p := range paras {
		if len(blocks) >= maxTextBlocks {
			break
		}
		if strings.TrimSpace(p) == "" {
			continue
		}
		// Markdown-ish headings at the start of a paragraph (# / ## / ###).
		line := strings.TrimSpace(p)
		typ, level, body := "paragraph", 0, p
		if strings.HasPrefix(line, "### ") {
			typ, level, body = "heading", 3, strings.TrimPrefix(line, "### ")
		} else if strings.HasPrefix(line, "## ") {
			typ, level, body = "heading", 2, strings.TrimPrefix(line, "## ")
		} else if strings.HasPrefix(line, "# ") {
			typ, level, body = "heading", 1, strings.TrimPrefix(line, "# ")
		}
		occKey := typ + "\x00" + normalizeBlockText(body)
		occ := seen[occKey]
		seen[occKey] = occ + 1
		blk := textBlock{
			ID:       stableBlockID(relKey, typ, level, body, occ),
			Type:     typ,
			Segments: []textSegment{{Text: body}},
		}
		if level > 0 {
			blk.Level = level
		}
		blocks = append(blocks, blk)
	}
	if len(blocks) == 0 {
		blocks = append(blocks, textBlock{ID: stableBlockID(relKey, "paragraph", 0, "", 0), Type: "paragraph", Segments: []textSegment{{Text: ""}}})
	}
	wire, err := json.Marshal(map[string]any{"blocks": blocks})
	if err != nil {
		return models.OpenedDocument{}, err
	}
	base := filepath.Base(absPath)
	return models.OpenedDocument{
		Title:      strings.TrimSuffix(base, filepath.Ext(base)),
		BlocksJSON: string(wire),
		PageCount:  1,
		Kind:       "text",
	}, nil
}

// WriteSummaryDOCX serializes blocksJSON to a .docx under the workspace root.
// relPath is workspace-relative (e.g. summaries/My-Title.docx). Returns the
// absolute path written. Phase 8.
func (d *Documents) WriteSummaryDOCX(absRoot, relPath, blocksJSON string) (string, error) {
	return WriteSummaryDOCXPath(absRoot, relPath, blocksJSON)
}
