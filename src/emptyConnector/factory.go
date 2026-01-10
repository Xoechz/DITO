package empty

// This connector exists as an example for my theis on how to build an OpenTelemetry Collector connector.

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/consumer"
)

var (
	typeStr = component.MustNewType("empty")
)

func createDefaultConfig() component.Config {
	return &Config{
		ExampleSetting: "example_value",
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

func NewFactory() connector.Factory {
	return connector.NewFactory(
		typeStr,
		createDefaultConfig,
		connector.WithTracesToTraces(createTracesToTracesConnector, component.StabilityLevelAlpha),
	)
}
