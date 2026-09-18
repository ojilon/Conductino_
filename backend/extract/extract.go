package extract

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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

// ErrUnsupportedType signals "no extractor for this extension yet" (HTML,
// … until their arms land). TXT/MD/PDF/DOCX are supported.
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
	case "pdf":
		// Text-layer extraction (pure Go, per-page blocks + page breaks).
		return openPdfFile(absPath, relKey)
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

// WriteSummaryDOCX serializes blocksJSON to a .docx under the workspace root.
// relPath is workspace-relative (e.g. summaries/My-Title.docx). Returns the
// absolute path written. Phase 8.
func (d *Documents) WriteSummaryDOCX(absRoot, relPath, blocksJSON string) (string, error) {
	return WriteSummaryDOCXPath(absRoot, relPath, blocksJSON)
}

// emptyDocument returns a blank openable document for 0-byte files.
// Brand-new files (e.g. a Summary.docx just created on disk) open as an
// empty page the user/AI can fill — never a parse_error. The renderer shows
// one empty block; the summary edit loop treats it as "no content yet".
func emptyDocument(absPath, kind string) models.OpenedDocument {
	base := filepath.Base(absPath)
	wire, err := marshalTextBlocks([]textBlock{
		{ID: "b-empty", Type: "paragraph", Segments: []textSegment{{Text: ""}}},
	})
	if err != nil {
		// Static shape above cannot fail to marshal; empty string still
		// decodes to zero blocks, which renderers handle as blank.
		wire = `{"blocks":[]}`
	}
	return models.OpenedDocument{
		Title:      strings.TrimSuffix(base, filepath.Ext(base)),
		BlocksJSON: wire,
		PageCount:  1,
		Kind:       kind,
	}
}
