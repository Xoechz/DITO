<!-- Async unified connector refactor plan -->

# Async Unified Connector Refactor Plan

Refactor so traces are ingested once, queued, processed asynchronously, and both transformed traces and derived metrics are emitted efficiently.

---

## 1. Unify Connector

**Goal:** Single component handles trace→trace and trace→metrics.

**Actions:**

1. Replace separate structs with one `ditoMonConnector`.
2. Store both downstream consumers: `nextTraces consumer.Traces`, `nextMetrics consumer.Metrics`.
3. Factory returns the same instance from both constructors (lookup by component ID).

---

## 2. Define Work Item

Create a lightweight `entityWork` struct containing:

- Cloned span (or extracted primitives only).
- SpanID, ParentSpanID.
- entityKey, jobKey (if known at ingest time).
- Status, start/end timestamps.
- Arrival time + deadline (`receivedAt + MaxCacheDuration`).

Index job spans immediately: `map[jobSpanID]*jobInfo`.

---

## 3. Ingestion (`ConsumeTraces`)

Pipeline on receive:

1. Gather spans once.
2. For job spans: update job index (small RWMutex window).
3. For entity spans not already pending:

- Add to `pending` map (dedupe).
- Push pointer onto bounded channel `pendingCh`.
- If channel full: drop (log + metric) or block briefly with timeout.

4. Return quickly (avoid heavy transformation here).

---

## 4. Channel & Worker Pool

**Startup:** Launch `N` workers (configurable).

**Worker loop:**

1. Read work item.
2. Resolve job span (check index).
3. If dependency missing and before deadline:

- Increment attempts; compute capped exponential backoff.
- Requeue via `time.AfterFunc` or light sleep; avoid busy-wait.

4. If expired: remove from pending; increment `expired_total`.
5. If ready: transform → trace accumulation + metrics counters.

---

## 5. Output Buffering & Flushing

Maintain two accumulators:

- Trace accumulator: entity-rooted spans grouped by entityKey.
- Metrics accumulator: counters keyed by `(jobKey, statusCode)` + time range.

**Flush strategies:**

- Option A: Per item (simple, higher overhead).
- Option B (recommended): Periodic ticker (e.g. 1s, configurable) and/or size-based (`spanCount >= batchSize`).

**On flush:**

1. Build `ptrace.Traces` → `nextTraces.ConsumeTraces` if non-empty.
2. Build `pmetric.Metrics` → `nextMetrics.ConsumeMetrics`.
3. Reset accumulators.

---

## 6. Sampling

Perform in worker before adding to trace accumulator.

- Maintain per-entity counters (in a sharded or per-entry lock map).
- Drop non-error spans based on `NonErrorSamplingFraction`.

---

## 7. Caches / Maps

| Purpose | Structure | Notes |
|---------|-----------|-------|
| Job spans | `map[SpanID]*jobInfo` | RWMutex; long-lived until TTL |
| Entity cache (roots) | `map[entityKey]*cacheEntry` | Holds root trace IDs, sampling counters |
| Pending work | `map[SpanID]*entityWork` | Prevent duplicate enqueues; expiry checks |

Consider a sharded map for entity cache under high cardinality.

---

## 8. Concurrency & Safety

- Never hold references to upstream pdata slices; clone or copy needed fields.
- Keep lock scope minimal: read job + entity state, then build spans outside.
- Use `sync.Once` for shared instance shutdown across both pipelines.

---

## 9. Shutdown Sequence

1. Close `stopCh`.
2. Close `pendingCh` (workers drain).
3. `WaitGroup` wait for workers.
4. Final flush.
5. Optionally clear maps.

---

## 10. Backpressure & Instrumentation

Metrics to emit:

- `queue_depth` (gauge)
- `enqueue_dropped_total` (counter)
- `requeue_total` (counter)
- `expired_total` (counter)
- `work_item_age_ms` (histogram)
- `processing_latency_ms` (histogram)
- `flush_spans` / `flush_metric_points` (histogram or summary)
- `delivery_errors_total`

---

## 11. Error Handling

On downstream error:

- Log with context (entityKey, counts).
- Increment `delivery_errors_total`.
- Do not block; drop the batch (avoid retry storms in connector layer).

---

## 12. Configuration Additions

| Field | Type | Purpose |
|-------|------|---------|
| `workers` | int | Number of worker goroutines |
| `queue_capacity` | int | Channel size for pending work |
| `flush_interval` | duration | Periodic flush cadence |
| `batch_size_threshold` | int (optional) | Flush when accumulated spans reach this |
| `max_requeue_backoff` | duration | Cap on backoff delay |
| `max_cache_duration` | duration | Existing: TTL for pending/work & roots |
| `non_error_sampling_fraction` | int | Existing sampling control |

---

## 13. Testing Strategy

- Unit tests: enqueue/processing, late job arrival, sampling correctness, expiration, flush trigger.
- Race detector: `go test -race`.
- Benchmarks: synthetic load measuring queue depth, latency, flush throughput.
- Deterministic IDs in tests for reproducibility.

---

## 14. Migration Steps (Order)

1. Merge connectors → shared instance.
2. Add shared accumulators (still synchronous path) to validate combined emission.
3. Introduce channel + workers (flush still inline for now).
4. Move to periodic / size-based flush.
5. Add sampling inside workers.
6. Implement expiration + requeue.
7. Add instrumentation + config fields.
8. Remove obsolete synchronous logic.

---

## 15. Optional Enhancements

- Adaptive flush interval (shorter under backlog).
- Replace channel with lock-free ring buffer if contention observed.
- Export Prometheus self-metrics for instrumentation counters.
- Add trace exemplars to metrics for sampled error spans.

---

## 16. Ready to Start

Ask: “start refactor with steps 1–3” (or another range) to begin incremental implementation.

---

_End of plan._
