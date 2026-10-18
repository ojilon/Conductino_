package docpipe

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"Conductino/backend/apppaths"
)

// The workspace SUMMARY TARGET: one file per workspace that the user marks
// as "the document all summaries go into" (right-click a .docx in the
// library → "Make this the target summary document").
//
// It is stored as a tiny JSON file under the app-data work dir — not in the
// workspace itself — so it survives restarts and both the Go tools and the
// AI can read it without any extra wiring:
//
//	<app-data>/work/summary-target.json
//	{ "root": "/abs/library", "relPath": "notes/Summary.docx", … }
//
// Root is stored alongside so a target never leaks across workspaces: when
// the library folder changes, the old target simply stops matching.

// SummaryTarget names the workspace document that receives summaries.
type SummaryTarget struct {
	Root    string `json:"root"`    // absolute library root it belongs to
	RelPath string `json:"relPath"` // workspace-relative path of the .docx
	Title   string `json:"title"`   // display name (file base without extension)
	SetAt   int64  `json:"setAt"`   // Unix ms when the user marked it
}

// TargetFilePath is where the summary target marker lives.
func TargetFilePath() string {
	return filepath.Join(apppaths.WorkDir(), "summary-target.json")
}

// SaveTarget marks a document as the workspace summary target.
func SaveTarget(t SummaryTarget) error {
	if t.SetAt == 0 {
		t.SetAt = time.Now().UnixMilli()
	}
	p := TargetFilePath()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, raw, 0o644)
}

// LoadTarget reads the current summary target. ok=false when none was ever
// set (or the file is unreadable/corrupt — a missing target is never fatal).
func LoadTarget() (SummaryTarget, bool) {
	raw, err := os.ReadFile(TargetFilePath())
	if err != nil {
		return SummaryTarget{}, false
	}
	var t SummaryTarget
	if err := json.Unmarshal(raw, &t); err != nil {
		return SummaryTarget{}, false
	}
	if t.RelPath == "" {
		return SummaryTarget{}, false
	}
	return t, true
}

// ClearTarget removes the marker (e.g. the target file was deleted).
func ClearTarget() { _ = os.Remove(TargetFilePath()) }
