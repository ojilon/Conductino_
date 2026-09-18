package ai

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"
	"sync"
	"time"

	"Conductino/backend/models"
)

// Request policy (issue 10, plan 05 §3): cost classes, a global in-process
// semaphore for free-tier concurrency, and a short-TTL cache for repeatable
// one-shot explanations. Chat turns and merge proposals are NEVER cached.

// costClass tags a request S/M/L before send (plan 05 §3.1).
func costClass(op models.AIOperation, inputChars int) string {
	if inputChars > 6000 {
		return "L" // large packs queue hardest regardless of op
	}
	switch op {
	case models.OpRewrite:
		return "S"
	case models.OpChat, models.OpMerge, models.OpSummarize:
		return "L" // tool loops / multi-call / full-text work
	default:
		return "M"
	}
}

// aiSem caps concurrent outbound model calls (free-tier RPM protection).
// Depth 2 lets a chat turn overlap one oneshot without stampeding.
var aiSem = make(chan struct{}, 2)

// acquire blocks until a slot frees or ctx cancels. The caller reports the
// wait via emitPhase so the UI shows "Waiting for AI capacity…" honestly.
func acquire(ctx context.Context, emitPhase func(label string)) error {
	select {
	case aiSem <- struct{}{}:
		return nil
	default:
	}
	if emitPhase != nil {
		emitPhase("Waiting for AI capacity…")
	}
	select {
	case aiSem <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func release() {
	select {
	case <-aiSem:
	default:
	}
}

// generateMetered is the single funnel for model calls: classify, acquire a
// semaphore slot (free-tier RPM protection), run failover, record usage.
// Callers pass the operation for cost class + telemetry.
func (g *GeminiService) generateMetered(
	ctx context.Context,
	op models.AIOperation,
	prompt string,
	maxTokens int,
	emitPhase func(label string),
) (string, error) {
	class := costClass(op, len(prompt))
	if err := acquire(ctx, emitPhase); err != nil {
		return "", err
	}
	defer release()
	start := time.Now()
	text, backend, err := g.generateWithFailover(ctx, prompt, maxTokens, emitPhase)
	if backend == "" {
		backend = "none"
	}
	recordUsage(backend, string(op), class, prompt, start, err)
	return text, err
}

// explainCache is a short-TTL cache for identical AI_EXPLAIN / AI_VERIFY
// calls (plan 05 §3.4). Key covers op + document + block + text hash so the
// same passage in another document never hits.
type explainCacheEntry struct {
	text string
	at   int64
}

const (
	explainCacheTTL = 5 * 60 * 1000 // 5 minutes
	explainCacheCap = 100
)

var explainMu sync.Mutex
var explainCache = map[string]explainCacheEntry{}

func explainCacheKey(op models.AIOperation, req models.AIRequest) string {
	sel := strings.TrimSpace(req.SelectionText)
	if sel == "" {
		sel = strings.TrimSpace(req.Selection)
	}
	sum := sha256.Sum256([]byte(sel + "\x00" + strings.TrimSpace(req.ContextPack)))
	return fmt.Sprintf("%s|%s|%s|%x", op, req.DocumentID, req.BlockID, sum[:8])
}

func explainCacheGet(op models.AIOperation, req models.AIRequest) (string, bool) {
	if op != models.OpExplain && op != models.OpVerify {
		return "", false
	}
	key := explainCacheKey(op, req)
	explainMu.Lock()
	defer explainMu.Unlock()
	e, ok := explainCache[key]
	if !ok || time.Now().UnixMilli()-e.at > explainCacheTTL {
		delete(explainCache, key)
		return "", false
	}
	return e.text, true
}

func explainCachePut(op models.AIOperation, req models.AIRequest, text string) {
	if op != models.OpExplain && op != models.OpVerify {
		return
	}
	if strings.TrimSpace(text) == "" {
		return
	}
	explainMu.Lock()
	defer explainMu.Unlock()
	if len(explainCache) >= explainCacheCap {
		// Evict an arbitrary expired-or-oldest entry; exact LRU is overkill
		// for a 5-minute cache.
		for k, e := range explainCache {
			if time.Now().UnixMilli()-e.at > explainCacheTTL {
				delete(explainCache, k)
				break
			}
		}
		if len(explainCache) >= explainCacheCap {
			for k := range explainCache {
				delete(explainCache, k)
				break
			}
		}
	}
	explainCache[explainCacheKey(op, req)] = explainCacheEntry{text: text, at: time.Now().UnixMilli()}
}
