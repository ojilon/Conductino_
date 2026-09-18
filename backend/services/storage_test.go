package services

import (
	"context"
	"path/filepath"
	"testing"
)

func TestInMemoryRoundTrip(t *testing.T) {
	s := NewStorage()
	_ = s.Init(context.Background())
	ws := WorkspaceRecord{ID: "ws1", RootPath: "/tmp/ws", PrimarySummaryID: "doc-s", Label: "demo"}
	if err := s.UpsertWorkspace(ws); err != nil {
		t.Fatal(err)
	}
	got, err := s.GetWorkspace("ws1")
	if err != nil || got == nil || got.PrimarySummaryID != "doc-s" {
		t.Fatalf("workspace: %+v err=%v", got, err)
	}
	doc := DocumentRecord{
		ID: "doc-s", WorkspaceID: "ws1", Kind: "summary", Title: "Summary",
		BlocksJSON: `{"blocks":[]}`,
	}
	if err := s.UpsertDocument(doc); err != nil {
		t.Fatal(err)
	}
	sum, err := s.GetPrimarySummary("ws1")
	if err != nil || sum == nil || sum.Title != "Summary" {
		t.Fatalf("primary summary: %+v", sum)
	}
	_ = s.UpsertThread(ChatThreadRecord{ID: "t1", WorkspaceID: "ws1", Title: "chat"})
	_ = s.AppendMessage(ChatMessageRecord{ID: "m1", ThreadID: "t1", Role: "user", Content: "hi"})
	msgs, err := s.ListMessages("t1")
	if err != nil || len(msgs) != 1 || msgs[0].Content != "hi" {
		t.Fatalf("messages: %+v", msgs)
	}
}

func TestSQLiteRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")
	s, err := NewSQLiteStorage(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err := s.Init(context.Background()); err != nil {
		t.Fatal(err)
	}
	if s.Engine() != "sqlite" {
		t.Fatalf("engine %s", s.Engine())
	}
	ws := WorkspaceRecord{ID: "ws1", RootPath: dir, PrimarySummaryID: "doc-s", Label: "lab"}
	if err := s.UpsertWorkspace(ws); err != nil {
		t.Fatal(err)
	}
	byRoot, err := s.GetWorkspaceByRoot(dir)
	if err != nil || byRoot == nil || byRoot.ID != "ws1" {
		t.Fatalf("by root: %+v", byRoot)
	}
	doc := DocumentRecord{
		ID: "doc-s", WorkspaceID: "ws1", Kind: "summary", Title: "Research Summary",
		BlocksJSON: `{"blocks":[{"id":"b1","type":"paragraph","segments":[{"text":"hello"}]}]}`,
	}
	if err := s.UpsertDocument(doc); err != nil {
		t.Fatal(err)
	}
	sum, err := s.GetPrimarySummary("ws1")
	if err != nil || sum == nil || sum.Title != "Research Summary" {
		t.Fatalf("summary: %+v", sum)
	}
	ch := ChangeRecord{
		ID: "c1", DocumentID: "doc-s", Type: "insert", BlockID: "b2",
		NewContent: "insert me", Status: "pending",
	}
	if err := s.UpsertChange(ch); err != nil {
		t.Fatal(err)
	}
	pending, err := s.ListPendingChanges("doc-s")
	if err != nil || len(pending) != 1 {
		t.Fatalf("pending: %+v", pending)
	}
	_ = s.SetSetting("last_library_root", dir)
	if v, ok := s.GetSetting("last_library_root"); !ok || v != dir {
		t.Fatalf("setting %q %v", v, ok)
	}
}

func TestExtractCacheRoundTrip(t *testing.T) {
	s := NewStorage()
	e := CachedExtract{Path: "root\x00a.txt", Mtime: 1, Size: 5, Title: "a", BlocksJSON: `{"blocks":[]}`}
	if err := s.PutCachedExtract(e); err != nil {
		t.Fatal(err)
	}
	got, err := s.GetCachedExtract(e.Path, 1, 5)
	if err != nil || got == nil || got.Title != "a" {
		t.Fatalf("cache hit: %+v err=%v", got, err)
	}
	// mtime change invalidates.
	if got, _ := s.GetCachedExtract(e.Path, 2, 5); got != nil {
		t.Fatalf("stale entry returned: %+v", got)
	}
}
