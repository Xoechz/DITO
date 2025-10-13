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
	logger          *zap.Logger
	config          Config
	traceConsumer   consumer.Traces
	sharedCache     *sharedCache
	shutdownChannel chan struct{}
	workerWG        sync.WaitGroup
	batchWG         sync.WaitGroup
	sweeperWG       sync.WaitGroup
	started         bool
}

func (s *traceConnector) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

func newTraceConnector(logger *zap.Logger, config component.Config, nextConsumer consumer.Traces) (*traceConnector, error) {
	logger.Info("Building dito trace connector")
	cfg := config.(*Config)

	return &traceConnector{
		config:          *cfg,
		logger:          logger,
		traceConsumer:   nextConsumer,
		sharedCache:     newSharedCache(cfg, logger),
		shutdownChannel: make(chan struct{}),
	}, nil
}

// Start launches worker, batcher and sweeper goroutines.
func (s *traceConnector) Start(ctx context.Context, host component.Host) error {
	if s.started {
		return nil
	}
	s.started = true

	s.logger.Info("Starting dito trace connector")

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
				if len(s.sharedCache.outputQueue) >= s.config.BatchSize {
					s.flushOutput()
				}
				time.Sleep(10 * time.Millisecond) // small sleep to avoid busy loop
			}
		}
	}()

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

				waitQueueBuffer := make([]*entityWorkItem, 0, len(s.sharedCache.waitQueue))

				// Drain waitQueue into buffer
				// This ensures we reprocess all waiting items, but avoid a live-lock in case waitQueue keeps getting new items.
				for len(s.sharedCache.waitQueue) > 0 {
					msg := <-s.sharedCache.waitQueue
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

	return nil
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
	s.sharedCache.ingestTraces(td, &s.config)

	s.logger.Debug("Traces ingested", zap.Int("messageQueueLength", len(s.sharedCache.messageQueue)), zap.Int("outputQueueLength", len(s.sharedCache.outputQueue)))

	return nil
}

func (s *traceConnector) processMessage(msg *entityWorkItem) {
	// check if job span exists, if not wait for the job span(for a max duration)
	jobSpan, newJobSpanId, jobState := s.sharedCache.getJobSpan(&msg.span, msg.entityKey)

	currentTime := time.Now()
	waitingTimeNotExceeded := msg.receivedAt.Add(s.config.MaxCacheDuration).After(currentTime)
	if jobState == JobStateNotFound && waitingTimeNotExceeded {
		// Requeue non-blocking; if queue full, drop (avoid shutdown hang / deadlock)
		select {
		case s.sharedCache.waitQueue <- msg:
		default:
			s.logger.Debug("Dropping entity span due to full requeue buffer", zap.String("entityKey", msg.entityKey))
		}
		return
	}
	if !s.started {
		// During shutdown we don't requeue; just drop
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

		s.sharedCache.outputQueue <- &rootSpan
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

		s.sharedCache.outputQueue <- jobSpan
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

	s.sharedCache.outputQueue <- &es
}

func (s *traceConnector) flushOutput() {
	if len(s.sharedCache.outputQueue) == 0 {
		return
	}

	batch := ptrace.NewTraces()
	rs := batch.ResourceSpans().AppendEmpty()
	rs.Resource().Attributes().EnsureCapacity(3)
	rs.Resource().Attributes().PutStr("service.name", SERVICE_NAME)
	rs.Resource().Attributes().PutInt("dito.entity.pending", int64(len(s.sharedCache.messageQueue)))
	scopeSpan := rs.ScopeSpans().AppendEmpty()
	scopeSpan.Scope().SetName(SERVICE_NAME)
	scopeSpan.Spans().EnsureCapacity(len(s.sharedCache.outputQueue))

	for len(s.sharedCache.outputQueue) > 0 {
		span := <-s.sharedCache.outputQueue
		span.CopyTo(scopeSpan.Spans().AppendEmpty())
	}

	s.logger.Debug("Flushed output", zap.Int("messageQueueLength", len(s.sharedCache.messageQueue)))
	err := s.traceConsumer.ConsumeTraces(context.Background(), batch)
	if err != nil {
		s.logger.Error("Error sending traces to next consumer", zap.Error(err))
	}
}
