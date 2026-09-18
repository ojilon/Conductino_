package ai

import (
	"context"
	"time"

	"Conductino/backend/models"
	"Conductino/backend/usage"
)

// Request policy lives in backend/usage (cost classes, semaphore, explain
// cache — plan 05 §3). This file keeps only generateMetered: the single
// funnel for model calls — classify, acquire a slot (free-tier RPM
// protection), run failover, record usage. Callers pass the operation for
// cost class + telemetry.
func (g *GeminiService) generateMetered(
	ctx context.Context,
	op models.AIOperation,
	prompt string,
	maxTokens int,
	emitPhase func(label string),
) (string, error) {
	class := usage.CostClass(op, len(prompt))
	if err := usage.Acquire(ctx, emitPhase); err != nil {
		return "", err
	}
	defer usage.Release()
	start := time.Now()
	text, backend, err := g.generateWithFailover(ctx, prompt, maxTokens, emitPhase)
	if backend == "" {
		backend = "none"
	}
	usage.RecordUsage(backend, string(op), class, prompt, start, err)
	return text, err
}
