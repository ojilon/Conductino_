package ai

import (
	"context"
	"testing"
	"time"

	"Conductino/backend/models"
)

func TestCostClass(t *testing.T) {
	if got := costClass(models.OpRewrite, 100); got != "S" {
		t.Fatalf("rewrite = %s, want S", got)
	}
	if got := costClass(models.OpChat, 100); got != "L" {
		t.Fatalf("chat = %s, want L", got)
	}
	if got := costClass(models.OpExplain, 100); got != "M" {
		t.Fatalf("explain = %s, want M", got)
	}
	if got := costClass(models.OpExplain, 9000); got != "L" {
		t.Fatalf("big explain = %s, want L", got)
	}
}

func TestExplainCacheRoundTrip(t *testing.T) {
	req := models.AIRequest{DocumentID: "d", BlockID: "b", SelectionText: "hello cache"}
	if _, ok := explainCacheGet(models.OpExplain, req); ok {
		t.Fatal("unexpected hit on empty cache")
	}
	explainCachePut(models.OpExplain, req, "cached answer")
	if got, ok := explainCacheGet(models.OpExplain, req); !ok || got != "cached answer" {
		t.Fatalf("miss after put: %q %v", got, ok)
	}
	// Chat and merges never cache.
	explainCachePut(models.OpChat, req, "nope")
	if _, ok := explainCacheGet(models.OpChat, req); ok {
		t.Fatal("chat must never hit cache")
	}
}

func TestSemaphoreAcquireRelease(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := acquire(ctx, nil); err != nil {
		t.Fatal(err)
	}
	if err := acquire(ctx, nil); err != nil {
		t.Fatal(err)
	}
	release()
	release()
	// Over-release must not panic or wedge the semaphore.
	release()
	if err := acquire(ctx, nil); err != nil {
		t.Fatal(err)
	}
	release()
}

func TestRecordUsage(t *testing.T) {
	before, _ := usageStats()
	recordUsage("gemini", "AI_EXPLAIN", "M", "hello world, this is a prompt", time.Now(), nil)
	after, byProv := usageStats()
	if after != before+1 || byProv["gemini"] < 1 {
		t.Fatalf("usage not recorded: %d -> %d %v", before, after, byProv)
	}
	if s := usageSnapshot(); s == "" {
		t.Fatal("empty snapshot after record")
	}
}
