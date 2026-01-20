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
	ditoCache       *ditoCache
	shutdownChannel chan struct{}
	workerWG        sync.WaitGroup
	sweeperWG       sync.WaitGroup
	started         bool
}

func (m *metricConnector) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

func newMetricConnector(logger *zap.Logger, config component.Config, nextConsumer consumer.Metrics) (*metricConnector, error) {
	logger.Info("Building dito metric connector")
	cfg := config.(*Config)

	return &metricConnector{
		config:          *cfg,
		logger:          logger,
		metricConsumer:  nextConsumer,
		ditoCache:       newditoCache(cfg, logger),
		shutdownChannel: make(chan struct{}),
	}, nil
}

// Start launches worker and sweeper goroutines.
func (m *metricConnector) Start(_ context.Context, _ component.Host) error {
	if m.started {
		return nil
	}
	m.started = true

	m.logger.Info("Starting dito metric connector")

	m.workerWG.Add(1)
	go func() {
		defer m.workerWG.Done()
		ticker := time.NewTicker(m.config.BatchTimeout)
		defer ticker.Stop()
		for {
			select {
			case <-m.shutdownChannel:
				m.processMessages() // final flush
				return
			case <-ticker.C:
				m.processMessages()
			default:
				// opportunistic flush when buffer big
				if len(m.ditoCache.messageQueue) >= m.config.BatchSize {
					m.processMessages()
				}
				time.Sleep(10 * time.Millisecond) // small sleep to avoid busy loop
			}
		}
	}()

	// Sweeper periodic eviction.
	m.sweeperWG.Add(1)
	go func() {
		defer m.sweeperWG.Done()
		sweepTicker := time.NewTicker(m.config.MaxCacheDuration / 2)
		defer sweepTicker.Stop()
		for {
			select {
			case <-m.shutdownChannel:
				return
			case <-sweepTicker.C:
				m.ditoCache.sweep()
			}
		}
	}()

	return nil
}

// Shutdown signals goroutines, waits for completion, drains remaining queues.
func (m *metricConnector) Shutdown(_ context.Context) error {
	if !m.started {
		return nil
	}

	m.logger.Info("Shutting down dito metric connector")

	close(m.shutdownChannel)
	m.started = false
	m.workerWG.Wait()
	m.sweeperWG.Wait()
	// Drain leftover output (best-effort)
	m.processMessages()
	return nil
}

func (m *metricConnector) ConsumeTraces(_ context.Context, td ptrace.Traces) error {
	m.ditoCache.ingestTraces(td, &m.config)

	m.logger.Debug("Metrics ingested", zap.Int("messageQueueLength", len(m.ditoCache.messageQueue)))

	return nil
}

func (m *metricConnector) processMessages() {
	if len(m.ditoCache.messageQueue) == 0 {
		return
	}

	startTime := time.Now()

	// drain the message queue to prevent infinite loop due to re-queuing
	currentBatch := make([]*entityWorkItem, 0, len(m.ditoCache.messageQueue))
	for len(m.ditoCache.messageQueue) > 0 {
		msg := <-m.ditoCache.messageQueue
		currentBatch = append(currentBatch, msg)
	}

	// guard if message queue was emptied by other goroutine
	if len(currentBatch) == 0 {
		return
	}

	metricGroups, minTime, maxTime := m.getMetricGroups(currentBatch)

	metrics := pmetric.NewMetrics()
	rm := metrics.ResourceMetrics().AppendEmpty()
	rm.Resource().Attributes().EnsureCapacity(3)
	rm.Resource().Attributes().PutStr("service.name", SERVICE_NAME)
	rm.Resource().Attributes().PutInt("dito.entity.pending", int64(len(m.ditoCache.messageQueue)))

	sm := rm.ScopeMetrics().AppendEmpty()
	sm.Scope().SetName(SERVICE_NAME)

	countMetric := sm.Metrics().AppendEmpty()
	countMetric.SetName("dito.entity.count")

	sum := countMetric.SetEmptySum()
	sum.SetIsMonotonic(true)
	sum.SetAggregationTemporality(pmetric.AggregationTemporalityDelta)

	for group, value := range *metricGroups {
		dp := sum.DataPoints().AppendEmpty()

		if value.jobSpan != nil {
			value.jobSpan.Attributes().CopyTo(dp.Attributes())

			exemplar := dp.Exemplars().AppendEmpty()
			exemplar.SetTraceID(value.jobSpan.TraceID())
			exemplar.SetSpanID(value.jobSpan.SpanID())
		}

		dp.SetStartTimestamp(minTime)
		dp.SetTimestamp(maxTime)
		dp.Attributes().PutStr("dito.entity.status_code", group.statusCode.String())
		dp.SetIntValue(value.count)
	}

	durationMetric := sm.Metrics().AppendEmpty()
	durationMetric.SetName("dito.entity.process_duration_ns")
	gauge := durationMetric.SetEmptyGauge()

	dp := gauge.DataPoints().AppendEmpty()
	dp.SetStartTimestamp(pcommon.NewTimestampFromTime(startTime))
	dp.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	dp.SetIntValue(int64(time.Since(startTime).Nanoseconds()))

	m.logger.Debug("Flushed output", zap.Int("messageQueueLength", len(m.ditoCache.messageQueue)))
	err := m.metricConsumer.ConsumeMetrics(context.Background(), metrics)
	if err != nil {
		m.logger.Error("Error sending metrics to next consumer", zap.Error(err))
	}
}

func (m *metricConnector) getMetricGroups(currentBatch []*entityWorkItem) (*map[metricGroup]metricValue, pcommon.Timestamp, pcommon.Timestamp) {
	// Guard against empty batch
	if len(currentBatch) == 0 {
		return &map[metricGroup]metricValue{}, pcommon.Timestamp(0), pcommon.Timestamp(0)
	}

	metricGroups := make(map[metricGroup]metricValue)
	minTime := currentBatch[0].sr.span.StartTimestamp()
	maxTime := currentBatch[0].sr.span.EndTimestamp()

	for _, msg := range currentBatch {
		// check if job span exists, if not wait for the job span(for a max duration)
		jobSpan, _, jobState := m.ditoCache.getJobSpan(msg.sr.span, msg.fullEntityKey)

		currentTime := time.Now()
		waitingTimeNotExceeded := msg.receivedAt.Add(m.config.MaxCacheDuration).After(currentTime)
		if jobState == JobStateNotFound && waitingTimeNotExceeded && m.started {
			// Requeue non-blocking; if queue full, drop (avoid shutdown hang / deadlock)
			// We don't need to use the waitingQueue, because we batch either way
			// If the connector is shutdown we just process everything we have left
			select {
			case m.ditoCache.messageQueue <- msg:
			default:
				m.logger.Debug("Dropping entity span due to full requeue buffer", zap.String("fullEntityKey", msg.fullEntityKey))
			}
			continue
		}
		// cache and sampling are not needed for the metrics

		statusCode := msg.sr.span.Status().Code()

		metricGroup := metricGroup{
			statusCode: statusCode,
		}

		var jobSpanSpan *ptrace.Span = nil
		if jobState != JobStateNotFound {
			metricGroup.jobSpanId = jobSpan.span.SpanID()
			jobSpanSpan = jobSpan.span
		}

		mv, exists := metricGroups[metricGroup]
		if !exists {
			mv = metricValue{
				count:   0,
				jobSpan: jobSpanSpan,
			}
		}

		mv.count++
		metricGroups[metricGroup] = mv

		if msg.sr.span.StartTimestamp() < minTime {
			minTime = msg.sr.span.StartTimestamp()
		}

		if msg.sr.span.EndTimestamp() > maxTime {
			maxTime = msg.sr.span.EndTimestamp()
		}
	}

	return &metricGroups, minTime, maxTime
}
