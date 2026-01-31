package empty

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"
)

type traceConnector struct {
	logger        *zap.Logger
	config        Config
	traceConsumer consumer.Traces
}

func newTraceConnector(logger *zap.Logger, config component.Config, nextConsumer consumer.Traces) (*traceConnector, error) {
	logger.Info("Building empty trace connector")
	cfg := config.(*Config)

	return &traceConnector{
		config:        *cfg,
		logger:        logger,
		traceConsumer: nextConsumer,
	}, nil
}

// Implements connector.Traces and consumer.Traces
func (t *traceConnector) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

// Implements connector.Traces and component.Component
func (t *traceConnector) Start(_ context.Context, _ component.Host) error {
	t.logger.Info("Starting empty trace connector with example setting", zap.String("example_setting", t.config.ExampleSetting))
	return nil
}

// Implements connector.Traces and component.Component
func (t *traceConnector) Shutdown(_ context.Context) error {
	t.logger.Info("Shutting down empty trace connector")
	return nil
}

// Implements connector.Traces and consumer.Traces
func (t *traceConnector) ConsumeTraces(ctx context.Context, td ptrace.Traces) error {
	t.logger.Info("Passing through traces", zap.Int("span_count", td.SpanCount()))
	return t.traceConsumer.ConsumeTraces(ctx, td)
}
