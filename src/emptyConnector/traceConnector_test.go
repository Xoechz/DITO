package empty

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"
)

func TestTraceConnectorCapabilities(t *testing.T) {
	connector := &traceConnector{}
	capabilities := connector.Capabilities()
	assert.False(t, capabilities.MutatesData)
}
func TestTracesConnector(t *testing.T) {
	ctx := context.Background()

	t.Run("passthrough span", func(t *testing.T) {
		// arrange
		tracesConsumer := &consumertest.TracesSink{}
		cfg := createDefaultConfig().(*Config)

		connector, err := newTraceConnector(zap.NewNop(), cfg, tracesConsumer)
		require.NoError(t, err)
		err = connector.Start(ctx, nil)
		require.NoError(t, err)
		defer connector.Shutdown(ctx)

		traces := ptrace.NewTraces()
		inputResourceSpan := traces.ResourceSpans().AppendEmpty()
		inputScopeSpan := inputResourceSpan.ScopeSpans().AppendEmpty()
		inputSpan := inputScopeSpan.Spans().AppendEmpty()

		inputSpan.Attributes().PutInt("attribute", 1)

		// act
		err = connector.ConsumeTraces(ctx, traces)
		require.NoError(t, err)

		// assert
		outputTraces := tracesConsumer.AllTraces()
		assert.Equal(t, 1, len(outputTraces))
		assert.Equal(t, 1, outputTraces[0].SpanCount())
		outputResourceSpan := outputTraces[0].ResourceSpans().At(0)
		outputScopeSpan := outputResourceSpan.ScopeSpans().At(0)
		outputSpan := outputScopeSpan.Spans().At(0)
		outputAttrValue, exists := outputSpan.Attributes().Get("attribute")
		assert.True(t, exists)
		assert.Equal(t, int64(1), outputAttrValue.Int())
	})
}
