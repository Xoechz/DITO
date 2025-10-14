# DITO - Work in Progress

![DITO](./logo.svg)

**D** ata and

**I** nformation

**T** racing using

**O** penTelemetry

My bachelors thesis about capturing data transfer interfaces validation data via OpenTelemetry.

## Usage

To use the connector, add `dito` to the `collector-config.yaml`.

Then add it to one trace pipeline as exporter and to a trace or metric pipeline as receiver.

### Config

The following configs are available:

- `entity_key`
  - The key of the attribute used to identify entity spans.
  - Default: dito.key
- `job_key`
  - The key of the attribute used to identify job spans.
  - Default: dito.job_id
- `baggage_job_key`
  - The key of the attribute used as fallback, to find the job span, to the ParentSpanId of the entity span.
  - Default: dito.job_span_id
- `max_cache_duration`
  - The maximum caching duration.
  - Default: 1h
- `cache_shard_count`
  - The number of shards to use for the cache. A shard is a subset of the cache to prevent excessive locking.
  - Default: 1
- `queue_size`
  - The maximum number of messages to queue.
  - Default: 32
- `worker_count`
  - The number of worker goroutines to process messages.
  - Default: 10000
- `sampling_fraction`
  - The fraction of messages to sample for processing.
  - Example: 2 => 1 in 2 spans are kept, 7 => 1 in 7 spans are kept.
  - Default: 4
- `batch_size`
  - The number of spans to process in each batch.
  - Default: 256
- `batch_timeout`
  - The maximum duration to wait before processing a batch.
  - Default: 2s

### Example

```yaml
connectors:
  dito:
    entity_key: dito.key
    job_key: dito.job_id
    baggage_job_key: dito.job_span_id
    max_cache_duration: 1h
    cache_shard_count: 1
    queue_size: 32
    worker_count: 10000
    sampling_fraction: 4
    batch_size: 256
    batch_timeout: 2s
service:
  pipelines:
    traces/default:
      receivers: [otlp]
      exporters: [dito, otlp]
    traces/dito:
      receivers: [dito]
      exporters: [otlp]
    metrics/dito:
      receivers: [dito]
      exporters: [otlp]
```
