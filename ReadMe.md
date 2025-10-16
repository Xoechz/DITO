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
  - Default: 32
- `queue_size`
  - The maximum number of messages to queue.
  - Default: 10000
- `worker_count`
  - The number of worker goroutines to process messages.
  - Default: 4
- `sampling_fraction`
  - The fraction of messages to sample for processing.
  - Example: 2 => 1 in 2 spans are kept, 7 => 1 in 7 spans are kept.
  - Default: 1
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
    cache_shard_count: 32
    queue_size: 10000
    worker_count: 4
    sampling_fraction: 1
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

## Architecture

Sadly a connector can only have one entry point and one output point. So there will always be multiple instances of the connector running, if traces AND metrics are being processed. The shared cache is only shared for parallel processes of one instance.

### Architecture Diagram

This diagram shows the architecture of the connector, as shown in the example configuration.

:::mermaid
flowchart LR
subgraph "default traces pipeline"
dr[otlp receiver]
end

subgraph "dito traces pipeline"
te[otlp exporter]
end

subgraph "dito metrics pipeline"
me[otlp exporter]
end

subgraph "dito connector"
subgraph "traceConnector"
tc[ConsumeTraces]
tp[processMessage]
tf[flushOutput]
tsc[sharedCache]
end
subgraph "metricConnector"
mc[ConsumeTraces]
mp[processMessages]
msc[sharedCache]
end
end

dr --> tc;
dr --> mc;

tc --> tsc;
mc --> msc;

tsc --> tp;
msc --> mp;

tp --> tf;

tf --> te;
mp ---> me;
:::

### Sequence Diagram

This sequence diagram shows the scenario, that multiple traces are processed to traces, with the example configuration. Incoming traces are not always one full trace, but a collection of batched trace data, the naming of OTEL is a bit confusing there.

The following data is incomming(simplified):

#### Traces 1

```json
{
  "spans": [
    {
      "trace_id": "100",
      "span_id": "10",
      "parent_span_id": "1",
      "name": "entity10",
      "attributes": {
        "dito.key": "entity a"
      }
    },
    {
      "trace_id": "100",
      "span_id": "20",
      "parent_span_id": "1",
      "name": "entity20",
      "attributes": {
        "dito.key": "entity b"
      }
    }
  ]
}
```

#### Traces 2

```json
{
  "spans": [
    {
      "trace_id": "100",
      "span_id": "1",
      "parent_span_id": null,
      "name": "job1",
      "attributes": {
        "dito.job_id": "job1"
      }
    },
    {
      "trace_id": "100",
      "span_id": "30",
      "parent_span_id": "1",
      "name": "entity10",
      "attributes": {
        "dito.key": "entity a"
      }
    }
  ]
}
```

:::mermaid
sequenceDiagram
box default traces pipeline
participant dr as otlp receiver
end
box traceConnector
participant tc as traceConnector
participant tm as messageQueue
participant tsc as sharedCache
participant t1 as Worker 1
participant t2 as Worker 2
participant t3 as Worker 3
participant to as outputQueue
participant tf as flushOutput
end
box dito traces pipeline
participant te as otlp exporter
end

dr->>tc: Input Traces 1
tc->>tm: Enqueue entity spans 10 and 20
tm->>t1: Dequeue entity span 10
tm->>t2: Dequeue entity span 20
t1->>tsc: lookup job span 1
t2->>tsc: wait for lock
tsc->>t1: not found
t2->>tsc: lookup job span 2
tsc->>t2: not found
t1->>tm: Requeue entity span 10
t2->>tm: Requeue entity span 10

dr->>tc: Input Traces 2
tc->>tm: Enqueue entity span 30
tc->>tsc: Cache job span 1

tm->>t3: Dequeue entity span 10
tm->>t1: Dequeue entity span 20
tm->>t2: Dequeue entity span 30
t3->>tsc: lookup job span 1
t1->>tsc: wait for lock
t2->>tsc: wait for lock
tsc->>t3: return job span 1
t1->>tsc: lookup job span 1
tsc->>t1: return job span 1
t2->>tsc: lookup job span 1
tsc->>t2: return job span 1

t3->>to: Enqueue root span for entity a
t3->>to: Enqueue result job span 1 in root a
t3->>to: Enqueue result entity span 30 in root a

t1->>to: Enqueue root span for entity b
t1->>to: Enqueue result job span 1 in root b
t1->>to: Enqueue result entity span 20 in root b

t2->>to: Enqueue result entity span 30 in root a
to->>tf: flushOutput
tf->>te: Send flushed spans in a traces object
