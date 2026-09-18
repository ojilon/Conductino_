// Package usage is backend telemetry: per-call usage rings, sliding RPM
// windows, request policy (cost classes, concurrency semaphore, one-shot
// explain cache), and the rate-limit error classifier.
//
// Everything here is best-effort — it never fails a model call — and
// dependency-free: models + stdlib ONLY. No network, no storage I/O.
// Persisted UsageEvents (plan 08 §5: ai_log table) build on UsageEvent,
// they don't replace this ring.
//
// Layout:
//   usage.go  — UsageEvent ring, RPM ticks, snapshot, token estimate
//   policy.go — CostClass, Acquire/Release, ExplainCacheGet/Put
//
// The ai package (network) calls usage.RecordUsage/CostClass/Acquire;
// nothing depends on ai. Parallel providers (plan 05/08) gain per-model
// quota buckets here without touching model code.
package usage
