package services

import (
	"context"

	"github.com/lumen/desktop/backend/models"
)

// DocumentService owns format extraction: turning raw files (PDF/DOCX/
// HTML/TXT) into the structured DocumentModel the frontend renders.
//
// FUTURE INTEGRATION (per format — see docs/document-rendering.md):
//   PDF   → pdfium / unipdf to extract text+layout, or ship pdf.js in the
//          webview and keep extraction client-side
//   DOCX  → baliance/gooxml or unidoc/unioffice to parse .docx into blocks
//   HTML  → golang.org/x/net/html → block/segment tree
//   TXT   → line split
//
// Until then, Extract returns a mock single-block document.
type DocumentService interface {
	Extract(ctx context.Context, source models.Source) (blocksJSON string, pageCount int, err error)
}

type Documents struct{}

func NewDocuments() *Documents { return &Documents{} }

func (d *Documents) Extract(_ context.Context, source models.Source) (string, int, error) {
	// Mock: one paragraph block containing the source abstract.
	blocks := `{"blocks":[{"id":"mock-1","type":"paragraph","segments":[{"text":"` + source.Abstract + `"}]}]}`
	return blocks, 1, nil
}
