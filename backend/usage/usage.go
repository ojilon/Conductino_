package ai

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// Usage tracking (issues 10+11, plan 05 §5): per-call records in a bounded
// ring plus sliding RPM windows per provider. Estimates use chars/4 until a
// provider returns exact usage. Everything here is best-effort telemetry —
// it never fails a model call.

// UsageEvent is one completed (or failed) model call.
type UsageEvent struct {
	At         int64  // Unix millis
	Provider   string // gemini | groq | openrouter
	Operation  string // AI_EXPLAIN, AI_CHAT, ...
	CostClass  string // S | M | L
	InputEst   int    // input tokens, estimated
	Status     string // ok | error | rate_limited
	LatencyMs  int64
}

const usageRingCap = 200

var usageMu sync.Mutex
var usageRing []UsageEvent
var usageCalls []usageTick // timestamps per provider for RPM

type usageTick struct {
	at       int64
	provider string
}

// estimateTokens is the chars/4 heuristic (plan 05 §3.2).
func estimateTokens(s string) int {
	return len(s) / 4
}

func recordUsage(provider, operation, class, prompt string, start time.Time, err error) {
	status := "ok"
	if err != nil {
		status = "error"
		if isRateLimitOrUnavailable(err) {
			status = "rate_limited"
		}
	}
	ev := UsageEvent{
		At:        time.Now().UnixMilli(),
		Provider:  provider,
		Operation: operation,
		CostClass: class,
		InputEst:  estimateTokens(prompt),
		Status:    status,
		LatencyMs: time.Since(start).Milliseconds(),
	}
	usageMu.Lock()
	defer usageMu.Unlock()
	if len(usageRing) >= usageRingCap {
		usageRing = usageRing[1:]
	}
	usageRing = append(usageRing, ev)
	usageCalls = append(usageCalls, usageTick{at: ev.At, provider: provider})
	// Prune ticks older than 2 minutes (keeps the slice bounded; RPM reads 60s).
	cutoff := ev.At - 120_000
	i := 0
	for i < len(usageCalls) && usageCalls[i].at < cutoff {
		i++
	}
	usageCalls = usageCalls[i:]
}

// usageSnapshot renders one line per provider for Settings / debugging:
// "gemini · 12 calls (1 err) · ~8k in-est · 2 rpm". Empty when no calls yet.
func usageSnapshot() string {
	usageMu.Lock()
	defer usageMu.Unlock()
	if len(usageRing) == 0 {
		return ""
	}
	type agg struct {
		calls, errs, in int
		rpm            int
		order          int
	}
	byProv := map[string]*agg{}
	var order []string
	now := time.Now().UnixMilli()
	for _, ev := range usageRing {
		a, ok := byProv[ev.Provider]
		if !ok {
			a = &agg{order: len(order)}
			byProv[ev.Provider] = a
			order = append(order, ev.Provider)
		}
		a.calls++
		a.in += ev.InputEst
		if ev.Status != "ok" {
			a.errs++
		}
	}
	for _, t := range usageCalls {
		if now-t.at <= 60_000 {
			byProv[t.provider].rpm++
		}
	}
	parts := make([]string, 0, len(order))
	for _, p := range order {
		a := byProv[p]
		inK := fmt.Sprintf("~%dk", a.in/1000)
		if a.in < 1000 {
			inK = fmt.Sprintf("~%d", a.in)
		}
		errS := ""
		if a.errs > 0 {
			errS = fmt.Sprintf(" · %d err", a.errs)
		}
		parts = append(parts, fmt.Sprintf("%s · %d calls%s · %s in-est · %d rpm", p, a.calls, errS, inK, a.rpm))
	}
	return strings.Join(parts, "\n")
}

// usageStats is the test hook (counts only, no formatting).
func usageStats() (calls int, byProvider map[string]int) {
	usageMu.Lock()
	defer usageMu.Unlock()
	byProvider = map[string]int{}
	for _, ev := range usageRing {
		calls++
		byProvider[ev.Provider]++
	}
	return calls, byProvider
}
