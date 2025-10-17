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

func (s *traceConnector) Capabilities() consumer.Capabilities {
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
func (s *traceConnector) Start(ctx context.Context, host component.Host) error {
	if s.started {
		return nil
	}
	s.started = true

	s.logger.Info("Starting dito trace connector")
	s.startWorkers()
	s.startBatcher()
	s.startSweeper()

	return nil
}

func (s *traceConnector) startWorkers() {
	// Workers: process entity items.
	for i := 0; i < s.config.WorkerCount; i++ {
		s.workerWG.Add(1)
		go func(id int) {
			defer s.workerWG.Done()
			for {
				select {
				case <-s.shutdownChannel:
					return
				case msg := <-s.sharedCache.messageQueue:
					if msg == nil {
						continue
					}
					s.processMessage(msg)
				}
			}
		}(i)
	}
}

func (s *traceConnector) startBatcher() {
	// Batcher: flush outputQueue on size or timeout.
	s.batchWG.Add(1)
	go func() {
		defer s.batchWG.Done()
		ticker := time.NewTicker(s.config.BatchTimeout)
		defer ticker.Stop()
		for {
			select {
			case <-s.shutdownChannel:
				s.flushOutput() // final flush
				return
			case <-ticker.C:
				s.flushOutput()
			default:
				// opportunistic flush when buffer big
				if len(s.outputQueue)+len(s.internalOutputQueue) >= s.config.BatchSize {
					s.flushOutput()
				}
				time.Sleep(10 * time.Millisecond) // small sleep to avoid busy loop
			}
		}
	}()
}

func (s *traceConnector) startSweeper() {
	// Sweeper periodic cache eviction and reprocessing.
	s.sweeperWG.Add(1)
	go func() {
		defer s.sweeperWG.Done()
		sweepTicker := time.NewTicker(time.Second * 10)
		defer sweepTicker.Stop()
		for {
			select {
			case <-s.shutdownChannel:
				return
			case <-sweepTicker.C:
				s.sharedCache.sweep()

				waitQueueBuffer := make([]*entityWorkItem, 0, len(s.waitQueue))

				// Drain waitQueue into buffer
				// This ensures we reprocess all waiting items, but avoid a live-lock in case waitQueue keeps getting new items.
				for len(s.waitQueue) > 0 {
					msg := <-s.waitQueue
					if msg == nil {
						continue
					}
					waitQueueBuffer = append(waitQueueBuffer, msg)
				}

				for _, msg := range waitQueueBuffer {
					s.sharedCache.messageQueue <- msg
				}
			}
		}
	}()
}

// Shutdown signals goroutines, waits for completion, drains remaining queues.
func (s *traceConnector) Shutdown(ctx context.Context) error {
	if !s.started {
		return nil
	}

	s.logger.Info("Shutting down dito trace connector")

	close(s.shutdownChannel)
	s.started = false
	s.workerWG.Wait()
	s.batchWG.Wait()
	s.sweeperWG.Wait()
	// Drain leftover output (best-effort)
	s.flushOutput()
	return nil
}

func (s *traceConnector) ConsumeTraces(ctx context.Context, td ptrace.Traces) error {
	s.traceMu.Lock()
	defer s.traceMu.Unlock()
	s.startInternalTraceIfNotExists()

	internalSpan := ptrace.NewSpan()
	internalSpan.SetName("dito.traceConnector.ConsumeTraces")
	internalSpan.SetTraceID(s.currentRootSpan.TraceID())
	internalSpan.SetSpanID(generateSpanID())
	internalSpan.SetParentSpanID(s.currentRootSpan.SpanID())
	internalSpan.SetStartTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	internalSpan.SetKind(ptrace.SpanKindServer)

	s.sharedCache.ingestTraces(td, &s.config)

	s.logger.Debug("Traces ingested", zap.Int("messageQueueLength", len(s.sharedCache.messageQueue)), zap.Int("outputQueueLength", len(s.outputQueue)))
	internalSpan.Attributes().PutInt("dito.entity.pending", int64(len(s.sharedCache.messageQueue)))
	internalSpan.Attributes().PutInt("dito.entity.ready", int64(len(s.outputQueue)))
	internalSpan.SetEndTimestamp(pcommon.NewTimestampFromTime(time.Now()))

	s.internalOutputQueue <- &internalSpan

	return nil
}

func (s *traceConnector) processMessage(msg *entityWorkItem) {
	// check if job span exists, if not wait for the job span(for a max duration)
	jobSpan, newJobSpanId, jobState := s.sharedCache.getJobSpan(&msg.span, msg.entityKey)

	currentTime := time.Now()
	waitingTimeNotExceeded := msg.receivedAt.Add(s.config.MaxCacheDuration).After(currentTime)
	if jobState == JobStateNotFound && waitingTimeNotExceeded && s.started {
		// Requeue non-blocking; if queue full, drop (avoid shutdown hang / deadlock)
		// If the connector is shutdown we just process everything we have left
		select {
		case s.waitQueue <- msg:
		default:
			s.logger.Debug("Dropping entity span due to full requeue buffer", zap.String("entityKey", msg.entityKey))
		}
		return
	}

	cache, entityWasCreated := s.sharedCache.getOrCreateEntityEntry(msg.entityKey)
	if entityWasCreated {
		rootSpan := ptrace.NewSpan()
		rootSpan.SetName(msg.entityKey)
		rootSpan.SetTraceID(cache.traceId)
		rootSpan.SetSpanID(cache.rootSpanId)
		rootSpan.SetStartTimestamp(pcommon.NewTimestampFromTime(currentTime))
		rootSpan.SetEndTimestamp(pcommon.NewTimestampFromTime(currentTime))

		s.outputQueue <- &rootSpan
	}

	// drop the span if the status is not an error and sampling says so
	// error spans are also increasing the sampling counter, so theoretically
	// if the samping fraction is 2 and every second span is an error, we could run into a situation
	// where every span is kept, but this is an edge case and acceptable
	if msg.span.Status().Code() != ptrace.StatusCodeError && cache.samplingCounter%s.config.SamplingFraction != 0 {
		s.logger.Debug("Non-error span skipped due to sampling", zap.String("entityKey", msg.entityKey))
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

		s.outputQueue <- jobSpan
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

	s.aggregateProcessDuration += time.Since(currentTime)

	s.outputQueue <- &es
}

func (s *traceConnector) flushOutput() {
	if len(s.outputQueue) == 0 && len(s.internalOutputQueue) < s.config.BatchSize {
		return
	}

	s.traceMu.Lock()
	defer s.traceMu.Unlock()
	s.startInternalTraceIfNotExists()

	internalSpan := ptrace.NewSpan()
	internalSpan.SetName("dito.traceConnector.flushOutput")
	internalSpan.SetTraceID(s.currentRootSpan.TraceID())
	internalSpan.SetSpanID(generateSpanID())
	internalSpan.SetParentSpanID(s.currentRootSpan.SpanID())
	internalSpan.SetStartTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	internalSpan.Attributes().PutInt("dito.entity.ready", int64(len(s.outputQueue)))

	batch := ptrace.NewTraces()
	rs := batch.ResourceSpans().AppendEmpty()
	rs.Resource().Attributes().PutStr("service.name", SERVICE_NAME)
	scopeSpan := rs.ScopeSpans().AppendEmpty()
	scopeSpan.Scope().SetName(SERVICE_NAME)
	scopeSpan.Spans().EnsureCapacity(len(s.outputQueue))

	counter := 0
	for len(s.outputQueue) > 0 {
		span := <-s.outputQueue
		span.CopyTo(scopeSpan.Spans().AppendEmpty())
		counter++
	}

	for len(s.internalOutputQueue) > 0 {
		span := <-s.internalOutputQueue
		span.CopyTo(scopeSpan.Spans().AppendEmpty())
	}

	internalSpan.Attributes().PutInt("dito.entity.flushed", int64(counter))
	internalSpan.Attributes().PutInt("dito.entity.pending", int64(len(s.sharedCache.messageQueue)))

	internalRootSpan := scopeSpan.Spans().AppendEmpty()
	s.currentRootSpan.CopyTo(internalRootSpan)
	s.currentRootSpan = nil

	endTime := time.Now()
	internalSpan.SetEndTimestamp(pcommon.NewTimestampFromTime(endTime))
	internalRootSpan.SetEndTimestamp(pcommon.NewTimestampFromTime(endTime))
	internalRootSpan.Attributes().PutInt("dito.aggregate_process_duration_ns", s.aggregateProcessDuration.Nanoseconds())
	internalSpan.CopyTo(scopeSpan.Spans().AppendEmpty())

	s.logger.Debug("Flushed output", zap.Int("messageQueueLength", len(s.sharedCache.messageQueue)))
	err := s.traceConsumer.ConsumeTraces(context.Background(), batch)
	if err != nil {
		s.logger.Error("Error sending traces to next consumer", zap.Error(err))
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
