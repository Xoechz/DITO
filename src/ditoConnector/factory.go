package dito

import (
	"context"
	"time"

	_ "github.com/expr-lang/expr" // ensure latest version is included to fix CVE-2025-68156
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/consumer"
)

var (
	typeStr = component.MustNewType("dito")
)

func createDefaultConfig() component.Config {
	return &Config{
		EntityKey:           "dito.key",
		EntityTypeKey:       "dito.entity_type",
		JobKey:              "dito.job_id",
		BaggageJobKey:       "dito.job_span_id",
		JobCacheDuration:    time.Hour,
		MaxEntityWaitTime:   time.Hour,
		EntityCacheDuration: time.Hour * 24 * 7,
		SamplingFraction:    1,
		CacheShardCount:     32,
		QueueSize:           10000,
		WorkerCount:         4,
		BatchSize:           256,
		BatchTimeout:        time.Minute,
		UseLinks:            true,
	}
}

func createTracesToTracesConnector(
	_ context.Context,
	params connector.Settings,
	cfg component.Config,
	nextConsumer consumer.Traces,
) (connector.Traces, error) {
	return newTraceConnector(params.Logger, cfg, nextConsumer)
}

func createTracesToMetricsConnector(
	_ context.Context,
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
