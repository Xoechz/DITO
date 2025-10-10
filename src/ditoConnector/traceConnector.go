package dito

import (
	"context"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"
)

const (
	SERVICE_NAME = "dito"
)

type entityWorkItem struct {
	entityKey  string
	span       ptrace.Span
	receivedAt time.Time
}

type traceConnector struct {
	logger          *zap.Logger
	config          Config
	traceConsumer   consumer.Traces
	messageQueue    chan *entityWorkItem
	shutdownChannel chan struct{}
	sharedCache     *sharedCache
}

func newTraceConnector(logger *zap.Logger, config component.Config, nextConsumer consumer.Traces) (*traceConnector, error) {
	logger.Info("Building dito traces connector")
	cfg := config.(*Config)

	return &traceConnector{
		config:        *cfg,
		logger:        logger,
		traceConsumer: nextConsumer,
		messageQueue:  make(chan *entityWorkItem, cfg.QueueSize),
		sharedCache:   newSharedCache(cfg.CacheShardCount, cfg.CacheShardCount, cfg.MaxCacheDuration),
	}, nil
}

func (s *traceConnector) ConsumeTraces(ctx context.Context, td ptrace.Traces) error {
	rss := td.ResourceSpans()
	now := time.Now()

	for i := 0; i < rss.Len(); i++ {
		rs := rss.At(i)
		ilss := rs.ScopeSpans()
		for j := 0; j < ilss.Len(); j++ {
			ils := ilss.At(j)
			for k := 0; k < ils.Spans().Len(); k++ {
				span := ils.Spans().At(k)

				entityKey, isEntity := span.Attributes().Get(s.config.EntityKey)
				if isEntity {
					s.messageQueue <- &entityWorkItem{
						entityKey:  entityKey.AsString(),
						span:       span,
						receivedAt: now,
					}
				}

				_, isJob := span.Attributes().Get(s.config.JobKey)
				if isJob {
					s.sharedCache.addJobSpan(&span, now)
				}
			}
		}
	}

	return nil
}
