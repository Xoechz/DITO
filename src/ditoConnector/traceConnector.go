package dito

import (
	"context"
	"sync"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"
)

type traceConnector struct {
	logger                   *zap.Logger
	config                   Config
	traceConsumer            consumer.Traces
	sharedCache              *sharedCache
	shutdownChannel          chan struct{}
	workerWG                 sync.WaitGroup
	batchWG                  sync.WaitGroup
	sweeperWG                sync.WaitGroup
	started                  bool
	waitQueue                chan *entityWorkItem
	outputQueue              chan *ptrace.Span
	internalOutputQueue      chan *ptrace.Span
	currentRootSpan          *ptrace.Span
	aggregateProcessDuration time.Duration
	traceMu                  sync.Mutex
}

func (t *traceConnector) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

func newTraceConnector(logger *zap.Logger, config component.Config, nextConsumer consumer.Traces) (*traceConnector, error) {
	logger.Info("Building dito trace connector")
	cfg := config.(*Config)

	return &traceConnector{
		config:              *cfg,
		logger:              logger,
		traceConsumer:       nextConsumer,
		sharedCache:         newSharedCache(cfg, logger),
		shutdownChannel:     make(chan struct{}),
		waitQueue:           make(chan *entityWorkItem, cfg.QueueSize),
		outputQueue:         make(chan *ptrace.Span, cfg.BatchSize*cfg.WorkerCount),
		internalOutputQueue: make(chan *ptrace.Span, cfg.BatchSize),
	}, nil
}

// Start launches worker, batcher and sweeper goroutines.
func (t *traceConnector) Start(ctx context.Context, host component.Host) error {
	if t.started {
		return nil
	}
	t.started = true

	t.logger.Info("Starting dito trace connector")
	t.startWorkers()
	t.startBatcher()
	t.startSweeper()

	return nil
}

func (t *traceConnector) startWorkers() {
	// Workers: process entity items.
	for i := 0; i < t.config.WorkerCount; i++ {
		t.workerWG.Add(1)
		go func(id int) {
			defer t.workerWG.Done()
			for {
				select {
				case <-t.shutdownChannel:
					return
				case msg := <-t.sharedCache.messageQueue:
					if msg == nil {
						continue
					}
					t.processMessage(msg)
				}
			}
		}(i)
	}
}

func (t *traceConnector) startBatcher() {
	// Batcher: flush outputQueue on size or timeout.
	t.batchWG.Add(1)
	go func() {
		defer t.batchWG.Done()
		ticker := time.NewTicker(t.config.BatchTimeout)
		defer ticker.Stop()
		for {
			select {
			case <-t.shutdownChannel:
				t.flushOutput() // final flush
				return
			case <-ticker.C:
				t.flushOutput()
			default:
				// opportunistic flush when buffer big
				if len(t.outputQueue)+len(t.internalOutputQueue) >= t.config.BatchSize {
					t.flushOutput()
				}
				time.Sleep(10 * time.Millisecond) // small sleep to avoid busy loop
			}
		}
	}()
}

func (t *traceConnector) startSweeper() {
	// Sweeper periodic cache eviction and reprocessing.
	t.sweeperWG.Add(1)
	go func() {
		defer t.sweeperWG.Done()
		sweepTicker := time.NewTicker(t.config.MaxCacheDuration / 2)
		requeueTicker := time.NewTicker(t.config.BatchTimeout)
		defer sweepTicker.Stop()
		defer requeueTicker.Stop()
		for {
			select {
			case <-t.shutdownChannel:
				return
			case <-sweepTicker.C:
				t.sharedCache.sweep()
			case <-requeueTicker.C:
				waitQueueBuffer := make([]*entityWorkItem, 0, len(t.waitQueue))

				// Drain waitQueue into buffer
				// This ensures we reprocess all waiting items, but avoid a live-lock in case waitQueue keeps getting new items.
				for len(t.waitQueue) > 0 {
					msg := <-t.waitQueue
					if msg == nil {
						continue
					}
					waitQueueBuffer = append(waitQueueBuffer, msg)
				}

				for _, msg := range waitQueueBuffer {
					t.sharedCache.messageQueue <- msg
				}
			}
		}
	}()
}

// Shutdown signals goroutines, waits for completion, drains remaining queues.
func (t *traceConnector) Shutdown(ctx context.Context) error {
	if !t.started {
		return nil
	}

	t.logger.Info("Shutting down dito trace connector")

	close(t.shutdownChannel)
	t.started = false
	t.workerWG.Wait()
	t.batchWG.Wait()
	t.sweeperWG.Wait()
	// Drain leftover output (best-effort)
	t.flushOutput()
	return nil
}

func (t *traceConnector) ConsumeTraces(ctx context.Context, td ptrace.Traces) error {
	t.traceMu.Lock()
	defer t.traceMu.Unlock()
	t.startInternalTraceIfNotExists()

	internalSpan := ptrace.NewSpan()
	internalSpan.SetName("dito.traceConnector.ConsumeTraces")
	internalSpan.SetTraceID(t.currentRootSpan.TraceID())
	internalSpan.SetSpanID(generateSpanID())
	internalSpan.SetParentSpanID(t.currentRootSpan.SpanID())
	internalSpan.SetStartTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	internalSpan.SetKind(ptrace.SpanKindServer)

	t.sharedCache.ingestTraces(td, &t.config)

	t.logger.Debug("Traces ingested", zap.Int("messageQueueLength", len(t.sharedCache.messageQueue)), zap.Int("outputQueueLength", len(t.outputQueue)))
	internalSpan.Attributes().PutInt("dito.entity.pending", int64(len(t.sharedCache.messageQueue)))
	internalSpan.Attributes().PutInt("dito.entity.ready", int64(len(t.outputQueue)))
	internalSpan.SetEndTimestamp(pcommon.NewTimestampFromTime(time.Now()))

	t.internalOutputQueue <- &internalSpan

	return nil
}

func (t *traceConnector) processMessage(msg *entityWorkItem) {
	// check if job span exists, if not wait for the job span(for a max duration)
	jobSpan, newJobSpanId, jobState := t.sharedCache.getJobSpan(&msg.span, msg.entityKey)

	currentTime := time.Now()
	waitingTimeNotExceeded := msg.receivedAt.Add(t.config.MaxCacheDuration).After(currentTime)
	if jobState == JobStateNotFound && waitingTimeNotExceeded && t.started {
		// Requeue non-blocking; if queue full, drop (avoid shutdown hang / deadlock)
		// If the connector is shutdown we just process everything we have left
		select {
		case t.waitQueue <- msg:
		default:
			t.logger.Debug("Dropping entity span due to full requeue buffer", zap.String("entityKey", msg.entityKey))
		}
		return
	}

	cache, entityWasCreated := t.sharedCache.getOrCreateEntityEntry(msg.entityKey)
	if entityWasCreated {
		rootSpan := ptrace.NewSpan()
		rootSpan.SetName(msg.entityKey)
		rootSpan.SetTraceID(cache.traceId)
		rootSpan.SetSpanID(cache.rootSpanId)
		rootSpan.SetStartTimestamp(pcommon.NewTimestampFromTime(currentTime))
		rootSpan.SetEndTimestamp(pcommon.NewTimestampFromTime(currentTime))

		t.outputQueue <- &rootSpan
	}

	// drop the span if the status is not an error and sampling says so
	// error spans are also increasing the sampling counter, so theoretically
	// if the samping fraction is 2 and every second span is an error, we could run into a situation
	// where every span is kept, but this is an edge case and acceptable
	if msg.span.Status().Code() != ptrace.StatusCodeError && cache.samplingCounter%t.config.SamplingFraction != 0 {
		t.logger.Debug("Non-error span skipped due to sampling", zap.String("entityKey", msg.entityKey))
		return
	}

	parentSpanId := cache.rootSpanId
	if jobState != JobStateNotFound {
		parentSpanId = newJobSpanId
	}

	if jobState == JobStateCreated {
		// add link to the original job span
		jLink := jobSpan.Links().AppendEmpty()
		jLink.SetTraceID(jobSpan.TraceID())
		jLink.SetSpanID(jobSpan.SpanID())

		jobSpan.SetTraceID(cache.traceId)
		jobSpan.SetSpanID(newJobSpanId)
		jobSpan.SetParentSpanID(cache.rootSpanId)

		t.outputQueue <- jobSpan
	}

	newSpanId := generateSpanID()
	es := ptrace.NewSpan()
	msg.span.CopyTo(es)

	es.SetTraceID(cache.traceId)
	es.SetSpanID(newSpanId)
	es.SetParentSpanID(parentSpanId)

	eLink := es.Links().AppendEmpty()
	eLink.SetTraceID(msg.span.TraceID())
	eLink.SetSpanID(msg.span.SpanID())

	t.aggregateProcessDuration += time.Since(currentTime)

	t.outputQueue <- &es
}

func (t *traceConnector) flushOutput() {
	if len(t.outputQueue) == 0 && len(t.internalOutputQueue) < t.config.BatchSize {
		return
	}

	t.traceMu.Lock()
	defer t.traceMu.Unlock()
	t.startInternalTraceIfNotExists()

	internalSpan := ptrace.NewSpan()
	internalSpan.SetName("dito.traceConnector.flushOutput")
	internalSpan.SetTraceID(t.currentRootSpan.TraceID())
	internalSpan.SetSpanID(generateSpanID())
	internalSpan.SetParentSpanID(t.currentRootSpan.SpanID())
	internalSpan.SetStartTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	internalSpan.Attributes().PutInt("dito.entity.ready", int64(len(t.outputQueue)))

	batch := ptrace.NewTraces()
	rs := batch.ResourceSpans().AppendEmpty()
	rs.Resource().Attributes().PutStr("service.name", SERVICE_NAME)
	scopeSpan := rs.ScopeSpans().AppendEmpty()
	scopeSpan.Scope().SetName(SERVICE_NAME)
	scopeSpan.Spans().EnsureCapacity(len(t.outputQueue))

	counter := 0
	for len(t.outputQueue) > 0 {
		span := <-t.outputQueue
		span.CopyTo(scopeSpan.Spans().AppendEmpty())
		counter++
	}

	for len(t.internalOutputQueue) > 0 {
		span := <-t.internalOutputQueue
		span.CopyTo(scopeSpan.Spans().AppendEmpty())
	}

	internalSpan.Attributes().PutInt("dito.entity.flushed", int64(counter))
	internalSpan.Attributes().PutInt("dito.entity.pending", int64(len(t.sharedCache.messageQueue)))

	internalRootSpan := scopeSpan.Spans().AppendEmpty()
	t.currentRootSpan.CopyTo(internalRootSpan)
	t.currentRootSpan = nil

	endTime := time.Now()
	internalSpan.SetEndTimestamp(pcommon.NewTimestampFromTime(endTime))
	internalRootSpan.SetEndTimestamp(pcommon.NewTimestampFromTime(endTime))
	internalRootSpan.Attributes().PutInt("dito.aggregate_process_duration_ns", t.aggregateProcessDuration.Nanoseconds())
	internalSpan.CopyTo(scopeSpan.Spans().AppendEmpty())

	t.logger.Debug("Flushed output", zap.Int("messageQueueLength", len(t.sharedCache.messageQueue)))
	err := t.traceConsumer.ConsumeTraces(context.Background(), batch)
	if err != nil {
		t.logger.Error("Error sending traces to next consumer", zap.Error(err))
	}
}

func (t *traceConnector) startInternalTraceIfNotExists() {
	if t.currentRootSpan != nil {
		return
	}

	internalRootSpan := ptrace.NewSpan()
	internalRootSpan.SetName("dito.traceConnector")
	internalRootSpan.SetTraceID(generateTraceID())
	internalRootSpan.SetSpanID(generateSpanID())
	internalRootSpan.SetStartTimestamp(pcommon.NewTimestampFromTime(time.Now()))

	t.currentRootSpan = &internalRootSpan
	t.aggregateProcessDuration = 0
}
