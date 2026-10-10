package services

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"Conductino/backend/internal/heavy"
	"Conductino/backend/models"
)

// openTextViaHeavy is the split-path txt/md reader (journey 10-09, 10-11):
// IO + jail-adjacent caps stay here in services; pure parsing lives in
// internal/heavy (no os, no net). Behavior (caps, headings, stable IDs,
// wire shape) matches openTextFile in documents.go by construction — heavy
// duplicates the same FNV-1a ID scheme so it never imports services.
//
// Not yet the default dispatch: OpenFile still routes to openTextFile.
// Flip after you compare both on your corpus (see documents_heavy_test.go).
func openTextViaHeavy(absPath, relKey string) (models.OpenedDocument, error) {
	if relKey == "" {
		relKey = filepath.Base(absPath)
	}
	info, err := os.Stat(absPath)
	if err != nil {
		reason, _ := ReasonOf(err)
		return models.OpenedDocument{}, &OpenError{Reason: reason, Detail: err.Error(), Err: err}
	}
	if info.IsDir() {
		err := fmt.Errorf("not a file: %s", absPath)
		return models.OpenedDocument{}, &OpenError{Reason: models.ReasonParseError, Detail: err.Error(), Err: err}
	}
	if info.Size() > heavy.MaxTextBytes {
		err := fmt.Errorf("text file too large (%d bytes)", info.Size())
		return models.OpenedDocument{}, &OpenError{Reason: models.ReasonTooLarge, Detail: err.Error(), Err: err}
	}
	f, err := os.Open(absPath)
	if err != nil {
		reason, _ := ReasonOf(err)
		return models.OpenedDocument{}, &OpenError{Reason: reason, Detail: err.Error(), Err: err}
	}
	defer f.Close()
	raw, err := io.ReadAll(io.LimitReader(f, heavy.MaxTextBytes+1))
	if err != nil {
		return models.OpenedDocument{}, &OpenError{Reason: models.ReasonParseError, Detail: err.Error(), Err: err}
	}
	blocksJSON, title, kind, reason, detail := heavy.ParseText(relKey, raw)
	if reason != "" {
		var r models.OpenFailureReason
		switch reason {
		case "too_large":
			r = models.ReasonTooLarge
		default:
			r = models.ReasonParseError
		}
		return models.OpenedDocument{}, &OpenError{Reason: r, Detail: detail, Err: fmt.Errorf("%s", detail)}
	}
	_ = strings.TrimSpace // keep strings import if future trims move here
	return models.OpenedDocument{
		Title:      title,
		BlocksJSON: blocksJSON,
		PageCount:  1,
		Kind:       kind,
	}, nil
}
