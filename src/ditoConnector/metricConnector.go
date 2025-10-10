package dito

import (
	"context"
	"sync"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"
)

type metricGroup struct {
	statusCode ptrace.StatusCode
	jobSpanId  pcommon.SpanID
}

type metricValue struct {
	count   int64
	jobSpan *ptrace.Span
}

type metricConnector struct {
	logger          *zap.Logger
	config          Config
	metricConsumer  consumer.Metrics
	sharedCache     *sharedCache
	shutdownChannel chan struct{}
	workerWG        sync.WaitGroup
	sweeperWG       sync.WaitGroup
	started         bool
}

func (s *metricConnector) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

func newMetricConnector(logger *zap.Logger, config component.Config, nextConsumer consumer.Metrics) (*metricConnector, error) {
	logger.Info("Building dito metric connector")
	cfg := config.(*Config)

	return &metricConnector{
		config:          *cfg,
		logger:          logger,
		metricConsumer:  nextConsumer,
		sharedCache:     newSharedCache(cfg),
		shutdownChannel: make(chan struct{}),
	}, nil
}

// Start launches worker and sweeper goroutines.
func (s *metricConnector) Start(ctx context.Context, host component.Host) error {
	if s.started {
		return nil
	}
	s.started = true

	s.logger.Info("Starting dito metric connector")

	s.workerWG.Add(1)
	go func() {
		defer s.workerWG.Done()
		ticker := time.NewTicker(s.config.BatchTimeout)
		defer ticker.Stop()
		for {
			select {
			case <-s.shutdownChannel:
				s.processMessages() // final flush
				return
			case <-ticker.C:
				s.processMessages()
			default:
				// opportunistic flush when buffer big
				if len(s.sharedCache.messageQueue) >= s.config.BatchSize {
					s.processMessages()
				}
				time.Sleep(10 * time.Millisecond) // small sleep to avoid busy loop
			}
		}
	}()

	// Sweeper periodic eviction.
	s.sweeperWG.Add(1)
	go func() {
		defer s.sweeperWG.Done()
		sweepTicker := time.NewTicker(s.config.MaxCacheDuration / 2)
		defer sweepTicker.Stop()
		for {
			select {
			case <-s.shutdownChannel:
				return
			case <-sweepTicker.C:
				s.sharedCache.sweep()
			}
		}
	}()

	return nil
}

// Shutdown signals goroutines, waits for completion, drains remaining queues.
func (s *metricConnector) Shutdown(ctx context.Context) error {
	if !s.started {
		return nil
	}

	s.logger.Info("Shutting down dito metric connector")

	close(s.shutdownChannel)
	s.started = false
	s.workerWG.Wait()
	s.sweeperWG.Wait()
	// Drain leftover output (best-effort)
	s.processMessages()
	return nil
}

func (s *metricConnector) ConsumeTraces(ctx context.Context, td ptrace.Traces) error {
	s.sharedCache.ingestTraces(td, &s.config)

	s.logger.Debug("Metrics ingested", zap.Int("messageQueueLength", len(s.sharedCache.messageQueue)))

	return nil
}

func (s *metricConnector) processMessages() {
	if len(s.sharedCache.messageQueue) == 0 {
		return
	}

	// drain the message queue to prevent infinite loop due to re-queuing
	currentBatch := make([]*entityWorkItem, 0, len(s.sharedCache.messageQueue))
	for len(s.sharedCache.messageQueue) > 0 {
		msg := <-s.sharedCache.messageQueue
		currentBatch = append(currentBatch, msg)
	}

	metricGroups := make(map[metricGroup]metricValue)
	minTime := currentBatch[0].span.StartTimestamp()
	maxTime := currentBatch[0].span.EndTimestamp()

	for _, msg := range currentBatch {
		// check if job span exists, if not wait for the job span(for a max duration)
		jobSpan, _, jobState := s.sharedCache.getJobSpan(msg.span.ParentSpanID(), msg.entityKey)

		currentTime := time.Now()
		waitingTimeNotExceeded := msg.receivedAt.Add(s.config.MaxCacheDuration).After(currentTime)
		if jobState == JobStateNotFound && waitingTimeNotExceeded {
			// Requeue non-blocking; if queue full, drop (avoid shutdown hang / deadlock)
			select {
			case s.sharedCache.messageQueue <- msg:
			default:
				s.logger.Debug("Dropping entity span due to full requeue buffer", zap.String("entityKey", msg.entityKey))
			}
			return
		}
		if !s.started {
			// During shutdown we don't requeue; just drop
			return
		}

		// cache and sampling are not needed for the metrics

		statusCode := msg.span.Status().Code()

		metricGroup := metricGroup{
			statusCode: statusCode,
		}

		if jobState != JobStateNotFound {
			metricGroup.jobSpanId = jobSpan.SpanID()
		}

		mv, exists := metricGroups[metricGroup]
		if !exists {
			mv = metricValue{
				count:   0,
				jobSpan: jobSpan,
			}
		}

		mv.count++
		metricGroups[metricGroup] = mv

		if msg.span.StartTimestamp() < minTime {
			minTime = msg.span.StartTimestamp()
		}

		if msg.span.EndTimestamp() > maxTime {
			maxTime = msg.span.EndTimestamp()
		}
	}

	metrics := pmetric.NewMetrics()
	rm := metrics.ResourceMetrics().AppendEmpty()
	rm.Resource().Attributes().EnsureCapacity(3)
	rm.Resource().Attributes().PutStr("service.name", SERVICE_NAME)
	rm.Resource().Attributes().PutInt("dito.entity.pending", int64(len(s.sharedCache.messageQueue)))

	sm := rm.ScopeMetrics().AppendEmpty()
	sm.Scope().SetName(SERVICE_NAME)

	countMetric := sm.Metrics().AppendEmpty()
	countMetric.SetName("dito.entity.count")

	sum := countMetric.SetEmptySum()
	sum.SetIsMonotonic(true)
	sum.SetAggregationTemporality(pmetric.AggregationTemporalityDelta)

	for group, value := range metricGroups {
		dp := sum.DataPoints().AppendEmpty()
		dp.SetStartTimestamp(minTime)
		dp.SetTimestamp(maxTime)

		value.jobSpan.Attributes().CopyTo(dp.Attributes())
		dp.Attributes().PutStr("dito.entity.status_code", group.statusCode.String())
		dp.SetIntValue(value.count)

		exemplar := dp.Exemplars().AppendEmpty()
		exemplar.SetTraceID(value.jobSpan.TraceID())
		exemplar.SetSpanID(value.jobSpan.SpanID())
	}

	s.logger.Debug("Flushed output", zap.Int("messageQueueLength", len(s.sharedCache.messageQueue)))
	err := s.metricConsumer.ConsumeMetrics(context.Background(), metrics)
	if err != nil {
		s.logger.Error("Error sending metrics to next consumer", zap.Error(err))
	}
}
