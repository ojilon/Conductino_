package mirror

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"Conductino/backend/extract"
)

// OpLog is one AI edit applied to the mirror (audit, surfaced in read_summary
// and, after publish, as in-file diffs). Text carries the new content so the
// frontend can locate and decorate the span.
type OpLog struct {
	Op     string `json:"op"` // insert | modify | delete
	Target string `json:"target,omitempty"`
	Text   string `json:"text,omitempty"`
	// OldText is the replaced/removed span (modify/delete) — the pre-image
	// reject-restore needs. Empty for inserts.
	OldText string `json:"oldText,omitempty"`
	Note   string `json:"note,omitempty"`
	At     int64  `json:"at"`
}

type meta struct {
	SourceHash string  `json:"sourceHash"`
	Log        []OpLog `json:"log,omitempty"`
	// Published counts log entries already written through to the .docx
	// (publish_summary). Unpublished = Log[Published:].
	Published int `json:"published,omitempty"`
}

// Store is a directory of summary mirrors. Zero value is unusable;
// construct with New (best-effort: New never fails, IO errors surface per call).
type Store struct {
	dir string
}

// New creates a store rooted at dir (created on demand per call).
func New(dir string) *Store { return &Store{dir: dir} }

// DefaultDir resolves backend/.work/summaries: repo-root cwd first (wails
// dev, go test), then beside the executable (installed binary).
func DefaultDir() string {
	if _, err := os.Stat("backend"); err == nil {
		return filepath.Join("backend", ".work", "summaries")
	}
	if exe, err := os.Executable(); err == nil {
		return filepath.Join(filepath.Dir(exe), ".work", "summaries")
	}
	return filepath.Join(".work", "summaries")
}

func hash(s string) string {
	sum := sha256.Sum256([]byte(s))
	return fmt.Sprintf("%x", sum[:8])
}

// sanitize keeps the summary id filesystem-safe (ids are uid() hex, but
// never trust a parameter that becomes a path).
func sanitize(id string) string {
	var b strings.Builder
	for _, r := range id {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
		}
	}
	if b.Len() == 0 {
		return "summary"
	}
	return b.String()
}

func (s *Store) paths(summaryID string) (mdPath, metaPath string) {
	base := filepath.Join(s.dir, sanitize(summaryID))
	return base + ".md", base + ".json"
}

func (s *Store) readMeta(metaPath string) meta {
	var m meta
	raw, err := os.ReadFile(metaPath)
	if err != nil {
		return m
	}
	_ = json.Unmarshal(raw, &m)
	return m
}

func (s *Store) writeMeta(metaPath string, m meta) error {
	if err := os.MkdirAll(filepath.Dir(metaPath), 0o755); err != nil {
		return err
	}
	raw, err := json.Marshal(m)
	if err != nil {
		return err
	}
	return os.WriteFile(metaPath, raw, 0o644)
}

// Read returns the mirror text for a summary. snapshot is the current
// frontend text: a hash mismatch (user saved/edited) re-syncs the mirror
// from the snapshot; an empty snapshot keeps the working copy (it means
// "nothing loaded", not "nothing exists"). Log rides along for the envelope.
func (s *Store) Read(summaryID, snapshot string) (md string, log []OpLog, err error) {
	if s == nil {
		return "", nil, fmt.Errorf("mirror: no store")
	}
	mdPath, metaPath := s.paths(summaryID)
	m := s.readMeta(metaPath)
	snap := strings.TrimSpace(snapshot)
	if snap == "" {
		// No snapshot: serve the working copy if any, else empty (valid —
		// the summary genuinely has no content yet).
		raw, rerr := os.ReadFile(mdPath)
		if rerr != nil {
			return "", m.Log, nil
		}
		return string(raw), m.Log, nil
	}
	if m.SourceHash == hash(snap) {
		raw, rerr := os.ReadFile(mdPath)
		if rerr != nil {
			return snap, m.Log, nil
		}
		return string(raw), m.Log, nil
	}
	// Snapshot moved under the mirror (user edit/save) — re-sync, keep log.
	md = extract.PlainToMarkdown(snap)
	if err := os.MkdirAll(filepath.Dir(mdPath), 0o755); err != nil {
		return "", nil, err
	}
	if err := os.WriteFile(mdPath, []byte(md), 0o644); err != nil {
		return "", nil, err
	}
	m.SourceHash = hash(snap)
	if err := s.writeMeta(metaPath, m); err != nil {
		return "", nil, err
	}
	return md, m.Log, nil
}

// Apply mutates the mirror text with one proposal op and appends to the log.
// Insert appends; modify replaces the OldText span (or the Target line);
// delete removes it. Missing spans never fail — they append a note so the
// next read_summary shows what did NOT land. Returns applied + note.
func (s *Store) Apply(summaryID, op, target, oldText, newText string) (applied bool, note string, err error) {
	if s == nil {
		return false, "", fmt.Errorf("mirror: no store")
	}
	mdPath, metaPath := s.paths(summaryID)
	m := s.readMeta(metaPath)
	raw, rerr := os.ReadFile(mdPath)
	if rerr != nil {
		return false, "", fmt.Errorf("mirror: no working copy for %s — read it first", sanitize(summaryID))
	}
	md := string(raw)
	op = strings.ToLower(strings.TrimSpace(op))
	target, oldText = strings.TrimSpace(target), strings.TrimSpace(oldText)
	newText = strings.TrimSpace(newText)
	switch op {
	case "insert":
		if newText == "" {
			return false, "empty insert ignored", nil
		}
		md = strings.TrimSpace(md) + "\n\n" + newText
		applied, note = true, "appended"
	case "modify":
		if oldText != "" && strings.Contains(md, oldText) {
			md = strings.Replace(md, oldText, newText, 1)
			applied, note = true, "span replaced"
		} else if target != "" && replaceLine(&md, target, newText) {
			applied, note = true, "target line replaced"
		} else {
			md = strings.TrimSpace(md) + "\n\n" + newText
			applied, note = false, "span not found — appended as insert instead"
		}
	case "delete":
		span := oldText
		if span == "" {
			span = target
		}
		if span != "" && strings.Contains(md, span) {
			md = strings.Replace(md, span, "", 1)
			applied, note = true, "span removed"
		} else if target != "" && deleteLine(&md, target) {
			applied, note = true, "target line removed"
		} else {
			applied, note = false, "span not found — nothing removed"
		}
	default:
		return false, "", fmt.Errorf("mirror: unknown op %q", op)
	}
	md = strings.TrimSpace(md)
	if err := os.WriteFile(mdPath, []byte(md), 0o644); err != nil {
		return false, "", err
	}
	entry := OpLog{Op: op, Target: target, Note: note, At: time.Now().UnixMilli()}
	if op != "delete" {
		entry.Text = truncateLogText(newText, 2000)
		entry.OldText = truncateLogText(oldText, 2000)
	} else if oldText != "" {
		entry.Text = truncateLogText(oldText, 2000)
	}
	m.Log = append(m.Log, entry)
	if len(m.Log) > 50 {
		m.Log = m.Log[len(m.Log)-50:]
	}
	if err := s.writeMeta(metaPath, m); err != nil {
		return applied, note, err
	}
	return applied, note, nil
}

// truncateLogText caps logged span text (decorations need the span, not megabytes).
func truncateLogText(s string, max int) string {
	s = strings.TrimSpace(s)
	if len(s) > max {
		return s[:max]
	}
	return s
}

// replaceLine swaps the first line containing target with newText.
func replaceLine(md *string, target, newText string) bool {
	lines := strings.Split(*md, "\n")
	for i, l := range lines {
		if strings.Contains(l, target) {
			lines[i] = newText
			*md = strings.Join(lines, "\n")
			return true
		}
	}
	return false
}

// deleteLine drops the first line containing target.
func deleteLine(md *string, target string) bool {
	lines := strings.Split(*md, "\n")
	for i, l := range lines {
		if strings.Contains(l, target) {
			*md = strings.Join(append(lines[:i], lines[i+1:]...), "\n")
			return true
		}
	}
	return false
}

// Unpublished returns log entries not yet written through to the .docx
// (the in-file diffs the UI decorates after a publish).
func (s *Store) Unpublished(summaryID string) []OpLog {
	if s == nil {
		return nil
	}
	_, metaPath := s.paths(summaryID)
	m := s.readMeta(metaPath)
	if m.Published < 0 || m.Published > len(m.Log) {
		return append([]OpLog(nil), m.Log...)
	}
	return append([]OpLog(nil), m.Log[m.Published:]...)
}

// MarkPublished advances the published watermark to the current log end.
// Call only after the .docx write succeeds.
func (s *Store) MarkPublished(summaryID string) error {
	if s == nil {
		return fmt.Errorf("mirror: no store")
	}
	_, metaPath := s.paths(summaryID)
	m := s.readMeta(metaPath)
	m.Published = len(m.Log)
	return s.writeMeta(metaPath, m)
}

// Sweep deletes mirrors untouched for olderThan (7-day passes on startup).
// Best-effort: errors are returned, callers log-and-continue.
func (s *Store) Sweep(olderThan time.Duration) error {
	if s == nil {
		return nil
	}
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	cutoff := time.Now().Add(-olderThan)
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			_ = os.Remove(filepath.Join(s.dir, e.Name()))
		}
	}
	return nil
}
