package dito

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/consumer"
)

var (
	typeStr = component.MustNewType("dito")
)

const (
	defaultEntityKey                = "dito.key"
	defaultJobKey                   = "dito.job_id"
	defaultMaxCachedEntities        = 10000
	defaultNonErrorSamplingFraction = 1
)

func createDefaultConfig() component.Config {
	return &Config{
		EntityKey:                defaultEntityKey,
		JobKey:                   defaultJobKey,
		MaxCachedEntities:        defaultMaxCachedEntities,
		NonErrorSamplingFraction: defaultNonErrorSamplingFraction,
	}
}

func createTracesToTracesConnector(
	ctx context.Context,
	params connector.Settings,
	cfg component.Config,
	nextConsumer consumer.Traces,
) (connector.Traces, error) {
	return newTracesConnector(params.Logger, cfg, nextConsumer)
}

func createTracesToMetricsConnector(
	ctx context.Context,
	params connector.Settings,
	cfg component.Config,
	nextConsumer consumer.Metrics,
) (connector.Traces, error) {
	return newMetricsConnector(params.Logger, cfg, nextConsumer)
}

func NewFactory() connector.Factory {
	return connector.NewFactory(
		typeStr,
		createDefaultConfig,
		connector.WithTracesToTraces(createTracesToTracesConnector, component.StabilityLevelAlpha),
		connector.WithTracesToMetrics(createTracesToMetricsConnector, component.StabilityLevelAlpha),
	)
}
