package dito

import (
	"context"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/consumer"
)

var (
	typeStr = component.MustNewType("dito")
)

func createDefaultConfig() component.Config {
	return &Config{
		EntityKey:        "dito.key",
		JobKey:           "dito.job_id",
		BaggageJobKey:    "dito.job_span_id",
		MaxCacheDuration: time.Hour,
		SamplingFraction: 1,
		CacheShardCount:  32,
		QueueSize:        10000,
		WorkerCount:      4,
		BatchSize:        256,
		BatchTimeout:     2 * time.Second,
	}
}

func createTracesToTracesConnector(
	ctx context.Context,
	params connector.Settings,
	cfg component.Config,
	nextConsumer consumer.Traces,
) (connector.Traces, error) {
	return newTraceConnector(params.Logger, cfg, nextConsumer)
}

func createTracesToMetricsConnector(
	ctx context.Context,
	params connector.Settings,
	cfg component.Config,
	nextConsumer consumer.Metrics,
) (connector.Traces, error) {
	return newMetricConnector(params.Logger, cfg, nextConsumer)
}

func NewFactory() connector.Factory {
	return connector.NewFactory(
		typeStr,
		createDefaultConfig,
		connector.WithTracesToTraces(createTracesToTracesConnector, component.StabilityLevelAlpha),
		connector.WithTracesToMetrics(createTracesToMetricsConnector, component.StabilityLevelAlpha),
	)
}
