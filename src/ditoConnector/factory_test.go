package dito

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.uber.org/zap"
)

func TestNewFactory(t *testing.T) {
	f := NewFactory()
	assert.Equal(t, typeStr, f.Type())

	// Ensure config creation works
	cfg := f.CreateDefaultConfig().(*Config)
	require.NotNil(t, cfg)

	// Create traces->traces connector
	tracesSink := &consumertest.TracesSink{}
	params := connector.Settings{ID: component.MustNewIDWithName(typeStr.String(), "test"), TelemetrySettings: component.TelemetrySettings{Logger: zap.NewNop()}}
	tracesConn, err := f.CreateTracesToTraces(context.Background(), params, cfg, tracesSink)
	require.NoError(t, err)
	assert.NotNil(t, tracesConn)

	// Create traces->metrics connector
	metricsSink := &consumertest.MetricsSink{}
	tracesToMetricsConn, err := f.CreateTracesToMetrics(context.Background(), params, cfg, metricsSink)
	require.NoError(t, err)
	assert.NotNil(t, tracesToMetricsConn)
}
