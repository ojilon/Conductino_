package services

import (
	"Conductino/backend/extract"
	"Conductino/backend/models"
)

// Extraction compatibility aliases (plan 06 step 1).
//
// Canonical implementations live in backend/extract (pure Go: no network,
// no Wails). These aliases keep backend/main.go, the Wails shell, and
// existing callers compiling untouched during the migration. New code must
// import backend/extract directly. Delete this file once all callers point
// at extract (plan 06 step 6).

// Documents is the extraction service (canonical: extract.Documents).
type Documents = extract.Documents

// DocumentService is the extraction boundary (canonical: extract.DocumentService).
type DocumentService = extract.DocumentService

// NewDocuments builds the extraction service.
func NewDocuments() *Documents { return extract.NewDocuments() }

// OpenError is a classified open failure (typed-failure contract).
type OpenError = extract.OpenError

// ErrUnsupportedType signals "no extractor for this extension yet".
var ErrUnsupportedType = extract.ErrUnsupportedType

// ReasonOf classifies any open error into a wire reason + detail.
func ReasonOf(err error) (models.OpenFailureReason, string) { return extract.ReasonOf(err) }

// DocxBlock is one DOCX wire block (canonical: extract.DocxBlock).
type DocxBlock = extract.DocxBlock

// DocxSegment is one styled text span (canonical: extract.DocxSegment).
type DocxSegment = extract.DocxSegment

// BlocksToJSON marshals blocks into the {blocks:[...]} wire shape.
func BlocksToJSON(blocks []DocxBlock) (string, error) { return extract.BlocksToJSON(blocks) }

// ParseBlocksJSON decodes the frontend/AI wire format.
func ParseBlocksJSON(s string) ([]DocxBlock, error) { return extract.ParseBlocksJSON(s) }

// DefaultSummaryDOCXName returns a stable relative path for a summary file.
func DefaultSummaryDOCXName(title string) string { return extract.DefaultSummaryDOCXName(title) }

// WriteSummaryDOCXPath resolves a workspace-relative path and writes DOCX.
func WriteSummaryDOCXPath(absRoot, relPath string, blocksJSON string) (string, error) {
	return extract.WriteSummaryDOCXPath(absRoot, relPath, blocksJSON)
}
