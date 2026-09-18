package usage

import (
	"context"
	"testing"
	"time"

	"Conductino/backend/models"
)

func TestCostClass(t *testing.T) {
	if got := CostClass(models.OpRewrite, 100); got != "S" {
		t.Fatalf("rewrite = %s, want S", got)
	}
	if got := CostClass(models.OpChat, 100); got != "L" {
		t.Fatalf("chat = %s, want L", got)
	}
	if got := CostClass(models.OpExplain, 100); got != "M" {
		t.Fatalf("explain = %s, want M", got)
	}
	if got := CostClass(models.OpExplain, 9000); got != "L" {
		t.Fatalf("big explain = %s, want L", got)
	}
}

func TestExplainCacheRoundTrip(t *testing.T) {
	req := models.AIRequest{DocumentID: "d", BlockID: "b", SelectionText: "hello cache"}
	if _, ok := ExplainCacheGet(models.OpExplain, req); ok {
		t.Fatal("unexpected hit on empty cache")
	}
	ExplainCachePut(models.OpExplain, req, "cached answer")
	if got, ok := ExplainCacheGet(models.OpExplain, req); !ok || got != "cached answer" {
		t.Fatalf("miss after put: %q %v", got, ok)
	}
	// Chat and merges never cache.
	ExplainCachePut(models.OpChat, req, "nope")
	if _, ok := ExplainCacheGet(models.OpChat, req); ok {
		t.Fatal("chat must never hit cache")
	}
}

func TestSemaphoreAcquireRelease(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := Acquire(ctx, nil); err != nil {
		t.Fatal(err)
	}
	if err := Acquire(ctx, nil); err != nil {
		t.Fatal(err)
	}
	Release()
	Release()
	// Over-Release must not panic or wedge the semaphore.
	Release()
	if err := Acquire(ctx, nil); err != nil {
		t.Fatal(err)
	}
	Release()
}

func TestRecordUsage(t *testing.T) {
	before, _ := UsageStats()
	RecordUsage("gemini", "AI_EXPLAIN", "M", "hello world, this is a prompt", time.Now(), nil)
	after, byProv := UsageStats()
	if after != before+1 || byProv["gemini"] < 1 {
		t.Fatalf("usage not recorded: %d -> %d %v", before, after, byProv)
	}
	if s := UsageSnapshot(); s == "" {
		t.Fatal("empty snapshot after record")
	}
}
