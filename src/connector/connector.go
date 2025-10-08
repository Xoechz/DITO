package dito

import (
	"context"
	"time"

	"crypto/rand"
	"sync"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"
)

const dito_SERVICE_NAME = "dito"

type ditoEntitySpan struct {
	entityKey string
	span      ptrace.Span
}

type ditoJobSpan struct {
	jobKey string
	span   ptrace.Span
}

type ditoCacheEntry struct {
	traceId              pcommon.TraceID
	rootSpanId           pcommon.SpanID
	jobSpanIds           map[pcommon.SpanID]pcommon.SpanID
	countSinceLastSample int
}

type metricGroup struct {
	statusCode   ptrace.StatusCode
	parentSpanId pcommon.SpanID
}

type ditoTracesConnector struct {
	logger        *zap.Logger
	config        Config
	traceConsumer consumer.Traces
	component.StartFunc
	component.ShutdownFunc
	// cache protected by cacheMu
	cache   map[string]*ditoCacheEntry
	cacheMu sync.RWMutex
}

type ditoMetricsConnector struct {
	logger          *zap.Logger
	config          Config
	metricsConsumer consumer.Metrics
	component.StartFunc
	component.ShutdownFunc
}

func newTracesConnector(logger *zap.Logger, config component.Config, nextConsumer consumer.Traces) (*ditoTracesConnector, error) {
	logger.Info("Building dito traces connector")
	cfg := config.(*Config)

	return &ditoTracesConnector{
		config:        *cfg,
		logger:        logger,
		traceConsumer: nextConsumer,
		cache:         make(map[string]*ditoCacheEntry),
	}, nil
}

func newMetricsConnector(logger *zap.Logger, config component.Config, nextConsumer consumer.Metrics) (*ditoMetricsConnector, error) {
	logger.Info("Building dito metrics connector")
	cfg := config.(*Config)

	return &ditoMetricsConnector{
		config:          *cfg,
		logger:          logger,
		metricsConsumer: nextConsumer,
	}, nil
}

func (s *ditoTracesConnector) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

func (s *ditoMetricsConnector) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

func (s *ditoTracesConnector) ConsumeTraces(ctx context.Context, td ptrace.Traces) error {
	entitySpans, jobSpans := gatherSpans(td.ResourceSpans(), &s.config)

	if len(entitySpans) == 0 {
		s.logger.Debug("No entity spans collected")
		return nil
	}

	// Process the collected spans
	entityTrace := ptrace.NewTraces()
	rs := entityTrace.ResourceSpans().AppendEmpty()
	rs.Resource().Attributes().PutStr("service.name", dito_SERVICE_NAME)
	scopeSpan := rs.ScopeSpans().AppendEmpty()
	scopeSpan.Scope().SetName(dito_SERVICE_NAME)
	scopeSpan.Spans().EnsureCapacity(len(entitySpans))

	for _, entitySpan := range entitySpans {
		s.ConsumeSpan(jobSpans, scopeSpan, entitySpan)
	}

	// perform size check under lock
	s.cacheMu.RLock()
	needClear := len(s.cache) > s.config.MaxCachedEntities
	s.cacheMu.RUnlock()

	if needClear {
		s.cacheMu.Lock()

		// re-check if it wasn't cleared in the meantime
		if len(s.cache) > s.config.MaxCachedEntities {
			s.logger.Debug("Max cached entities reached. Cache will be cleared")
			s.cache = make(map[string]*ditoCacheEntry)
		}

		s.cacheMu.Unlock()
	}

	return s.traceConsumer.ConsumeTraces(ctx, entityTrace)
}

func (s *ditoMetricsConnector) ConsumeTraces(ctx context.Context, td ptrace.Traces) error {
	entitySpans, jobSpans := gatherSpans(td.ResourceSpans(), &s.config)

	if len(entitySpans) == 0 {
		s.logger.Debug("No entity spans collected")
		return nil
	}

	metricGroups := make(map[metricGroup]int64)
	statusExemplars := make(map[ptrace.StatusCode]ptrace.Span)
	minTime := entitySpans[0].span.StartTimestamp()
	maxTime := entitySpans[0].span.EndTimestamp()

	for _, entitySpan := range entitySpans {
		statusCode := entitySpan.span.Status().Code()

		metricGroup := metricGroup{
			statusCode:   statusCode,
			parentSpanId: entitySpan.span.ParentSpanID(),
		}

		metricGroups[metricGroup]++
		statusExemplars[statusCode] = entitySpan.span

		if entitySpan.span.StartTimestamp() < minTime {
			minTime = entitySpan.span.StartTimestamp()
		}

		if entitySpan.span.EndTimestamp() > maxTime {
			maxTime = entitySpan.span.EndTimestamp()
		}
	}

	metrics := pmetric.NewMetrics()
	rm := metrics.ResourceMetrics().AppendEmpty()
	rm.Resource().Attributes().PutStr("service.name", dito_SERVICE_NAME)

	sm := rm.ScopeMetrics().AppendEmpty()
	sm.Scope().SetName(dito_SERVICE_NAME)

	countMetric := sm.Metrics().AppendEmpty()
	countMetric.SetName("dito.entity.count")

	sum := countMetric.SetEmptySum()
	sum.SetIsMonotonic(true)
	sum.SetAggregationTemporality(pmetric.AggregationTemporalityDelta)

	for group, count := range metricGroups {
		dp := sum.DataPoints().AppendEmpty()
		dp.SetStartTimestamp(minTime)
		dp.SetTimestamp(maxTime)

		jobSpan, exists := jobSpans[group.parentSpanId]

		if exists {
			jobSpan.span.Attributes().CopyTo(dp.Attributes())
		}

		dp.Attributes().PutStr("dito.entity.status_code", group.statusCode.String())
		dp.SetIntValue(count)

		exemplar := dp.Exemplars().AppendEmpty()
		exemplar.SetTraceID(statusExemplars[group.statusCode].TraceID())
		exemplar.SetSpanID(statusExemplars[group.statusCode].SpanID())
	}

	return s.metricsConsumer.ConsumeMetrics(ctx, metrics)
}

func (s *ditoTracesConnector) ConsumeSpan(jobSpans map[pcommon.SpanID]ditoJobSpan, scopeSpan ptrace.ScopeSpans, entitySpan ditoEntitySpan) {
	// All cache interactions + potential writes are protected
	// If the duration of waiting for the log is too long, we may need to optimize this
	beforeLock := time.Now()
	s.cacheMu.Lock()
	afterLock := time.Now()

	s.logger.Debug("Acquired cache lock",
		zap.String("entityKey", entitySpan.entityKey),
		zap.Duration("wait_duration", afterLock.Sub(beforeLock)),
	)

	cache, cacheHit := s.cache[entitySpan.entityKey]
	if !cacheHit {
		cache = &ditoCacheEntry{
			traceId:              generateTraceID(),
			rootSpanId:           generateSpanID(),
			jobSpanIds:           make(map[pcommon.SpanID]pcommon.SpanID),
			countSinceLastSample: s.config.NonErrorSamplingFraction,
		}
		s.cache[entitySpan.entityKey] = cache

		rootSpan := scopeSpan.Spans().AppendEmpty()
		rootSpan.SetName(entitySpan.entityKey)
		rootSpan.SetTraceID(cache.traceId)
		rootSpan.SetSpanID(cache.rootSpanId)
		rootSpan.SetStartTimestamp(pcommon.NewTimestampFromTime(time.Now()))
		rootSpan.SetEndTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	}

	// If the span is not an error, we apply sampling, every 1 in nonErrorSamplingFraction entitySpan of a specific key will be kept
	if entitySpan.span.Status().Code() != ptrace.StatusCodeError {
		cache.countSinceLastSample++

		if cache.countSinceLastSample < s.config.NonErrorSamplingFraction {
			s.logger.Debug("Non-error span skipped due to sampling", zap.String("entityKey", entitySpan.entityKey))
			s.cacheMu.Unlock()
			return
		} else {
			cache.countSinceLastSample = 0
		}
	}

	parentSpanId := cache.rootSpanId
	jobSpan, jobSpanExists := jobSpans[entitySpan.span.ParentSpanID()]

	if jobSpanExists {
		jobSpanId, exists := cache.jobSpanIds[jobSpan.span.SpanID()]

		if !exists {
			jobSpanId = generateSpanID()
			cache.jobSpanIds[jobSpan.span.SpanID()] = jobSpanId

			js := scopeSpan.Spans().AppendEmpty()
			jobSpan.span.CopyTo(js)
			js.SetTraceID(cache.traceId)
			js.SetSpanID(jobSpanId)
			js.SetParentSpanID(cache.rootSpanId)

			jLink := js.Links().AppendEmpty()
			jLink.SetTraceID(jobSpan.span.TraceID())
			jLink.SetSpanID(jobSpan.span.SpanID())
		}

		parentSpanId = jobSpanId
	}

	newSpanId := generateSpanID()
	es := scopeSpan.Spans().AppendEmpty()
	entitySpan.span.CopyTo(es)

	es.SetTraceID(cache.traceId)
	es.SetSpanID(newSpanId)
	es.SetParentSpanID(parentSpanId)
	es.Attributes().PutInt("dito.lock_wait_duration", afterLock.Sub(beforeLock).Milliseconds())

	eLink := es.Links().AppendEmpty()
	eLink.SetTraceID(entitySpan.span.TraceID())
	eLink.SetSpanID(entitySpan.span.SpanID())

	s.cacheMu.Unlock()
}

func gatherSpans(rss ptrace.ResourceSpansSlice, config *Config) (entitySpans []ditoEntitySpan, jobSpans map[pcommon.SpanID]ditoJobSpan) {
	entitySpans = make([]ditoEntitySpan, 0)
	jobSpans = make(map[pcommon.SpanID]ditoJobSpan)

	for i := 0; i < rss.Len(); i++ {
		rs := rss.At(i)
		ilss := rs.ScopeSpans()
		for j := 0; j < ilss.Len(); j++ {
			ils := ilss.At(j)
			for k := 0; k < ils.Spans().Len(); k++ {
				span := ils.Spans().At(k)

				isEntity, entitySpan := checkEntitySpan(span, config)
				if isEntity {
					entitySpans = append(entitySpans, *entitySpan)
				}

				isJob, jobSpan := checkJobSpan(span, config)
				if isJob {
					jobSpans[jobSpan.span.SpanID()] = *jobSpan
				}
			}
		}
	}

	return entitySpans, jobSpans
}

func checkEntitySpan(span ptrace.Span, config *Config) (bool, *ditoEntitySpan) {
	entityKey, isEntity := span.Attributes().Get(config.EntityKey)

	if !isEntity {
		return false, nil
	}

	return true, &ditoEntitySpan{
		entityKey: entityKey.AsString(),
		span:      span,
	}
}

func checkJobSpan(span ptrace.Span, config *Config) (bool, *ditoJobSpan) {
	jobKey, isJob := span.Attributes().Get(config.JobKey)

	if !isJob {
		return false, nil
	}

	return true, &ditoJobSpan{
		jobKey: jobKey.AsString(),
		span:   span,
	}
}

func generateTraceID() pcommon.TraceID {
	var tid [16]byte
	_, err := rand.Read(tid[:])
	if err != nil {
		// handle error appropriately
	}

	return pcommon.TraceID(tid)
}

func generateSpanID() pcommon.SpanID {
	var sid [8]byte
	_, err := rand.Read(sid[:])
	if err != nil {
		// handle error appropriately
	}
	return pcommon.SpanID(sid)
}
