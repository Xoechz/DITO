package dito

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"
)

func TestMetricConnectorCapabilities(t *testing.T) {
	t.Run("dito metrics connector capabilities", func(t *testing.T) {
		connector := &metricConnector{}
		capabilities := connector.Capabilities()
		assert.False(t, capabilities.MutatesData)
	})
}

func TestMetricsConnector(t *testing.T) {
	// Create a test consumer that captures metrics
	metricsConsumer := &consumertest.MetricsSink{}
	cfg := createDefaultConfig().(*Config)
	cfg.BatchTimeout = 100 * time.Millisecond
	cfg.MaxCacheDuration = MAX_CACHE_DURATION

	connector, err := newMetricConnector(zap.NewNop(), cfg, metricsConsumer)
	require.NoError(t, err)

	ctx := context.Background()

	err = connector.Start(ctx, nil)
	require.NoError(t, err)

	t.Run("no entity spans", func(t *testing.T) {
		// arrange
		metricsConsumer.Reset()
		connector.sharedCache.reset()

		traces := ptrace.NewTraces()
		inputResourceSpan := traces.ResourceSpans().AppendEmpty()
		inputScopeSpan := inputResourceSpan.ScopeSpans().AppendEmpty()
		inputSpan := inputScopeSpan.Spans().AppendEmpty()

		inputSpan.SetSpanID(generateSpanID())
		inputSpan.Attributes().PutInt("other", 1)

		// act
		err := connector.ConsumeTraces(ctx, traces)
		require.NoError(t, err)
		time.Sleep(TEST_WAIT) // allow worker to process

		// assert
		outputMetrics := metricsConsumer.AllMetrics()
		assert.Equal(t, 0, len(outputMetrics))
	})

	t.Run("grouped metrics", func(t *testing.T) {
		// arrange
		metricsConsumer.Reset()
		connector.sharedCache.reset()

		traces := ptrace.NewTraces()
		inputResourceSpan := traces.ResourceSpans().AppendEmpty()
		inputScopeSpan := inputResourceSpan.ScopeSpans().AppendEmpty()
		inputJobSpan1 := inputScopeSpan.Spans().AppendEmpty()
		inputJobSpan2 := inputScopeSpan.Spans().AppendEmpty()
		inputSpan1 := inputScopeSpan.Spans().AppendEmpty()
		inputSpan2 := inputScopeSpan.Spans().AppendEmpty()
		inputSpan3 := inputScopeSpan.Spans().AppendEmpty()
		inputSpan4 := inputScopeSpan.Spans().AppendEmpty()
		inputSpan5 := inputScopeSpan.Spans().AppendEmpty()
		inputSpan6 := inputScopeSpan.Spans().AppendEmpty()
		inputSpan7 := inputScopeSpan.Spans().AppendEmpty()

		jobSpanId1 := generateSpanID()
		inputJobSpan1.SetSpanID(jobSpanId1)
		inputJobSpan1.Attributes().PutInt(JOB_KEY_VALUE, 1)

		jobSpanId2 := generateSpanID()
		inputJobSpan2.SetSpanID(jobSpanId2)
		inputJobSpan2.Attributes().PutInt(JOB_KEY_VALUE, 2)

		inputSpan1.SetSpanID(generateSpanID())
		inputSpan1.SetParentSpanID(jobSpanId1)
		inputSpan1.Attributes().PutInt(ENTITY_KEY_VALUE, 1)
		inputSpan1.Attributes().PutStr("test.key", "test.value")

		inputSpan2.SetSpanID(generateSpanID())
		inputSpan2.SetParentSpanID(jobSpanId2)
		inputSpan2.Attributes().PutInt(ENTITY_KEY_VALUE, 2)
		inputSpan2.Attributes().PutStr("test.key", "test.value")

		inputSpan3.SetSpanID(generateSpanID())
		inputSpan3.SetParentSpanID(jobSpanId1)
		inputSpan3.Attributes().PutInt(ENTITY_KEY_VALUE, 3)

		inputSpan4.SetSpanID(generateSpanID())
		inputSpan4.SetParentSpanID(jobSpanId1)
		inputSpan4.Attributes().PutInt(ENTITY_KEY_VALUE, 4)

		inputSpan5.SetSpanID(generateSpanID())
		inputSpan5.SetParentSpanID(jobSpanId1)
		inputSpan5.Attributes().PutInt(ENTITY_KEY_VALUE, 5)

		inputSpan6.SetSpanID(generateSpanID())
		inputSpan6.SetParentSpanID(jobSpanId1)
		inputSpan6.Attributes().PutInt(ENTITY_KEY_VALUE, 6)

		inputSpan7.SetSpanID(generateSpanID())
		inputSpan7.SetParentSpanID(jobSpanId1)
		inputSpan7.Attributes().PutInt(ENTITY_KEY_VALUE, 7)
		inputSpan7.Status().SetCode(ptrace.StatusCodeError)

		// act
		err := connector.ConsumeTraces(ctx, traces)
		require.NoError(t, err)
		time.Sleep(TEST_WAIT) // allow worker to process

		// assert
		outputRootMetrics := metricsConsumer.AllMetrics()
		assert.Equal(t, 1, len(outputRootMetrics))

		outputResourceMetrics := outputRootMetrics[0].ResourceMetrics()
		assert.Equal(t, 1, outputResourceMetrics.Len())

		outputScopeMetrics := outputResourceMetrics.At(0).ScopeMetrics()
		assert.Equal(t, 1, outputScopeMetrics.Len())

		outputMetrics := outputScopeMetrics.At(0).Metrics()
		assert.Equal(t, 2, outputMetrics.Len())
		assert.Equal(t, "dito.entity.count", outputMetrics.At(0).Name())
		assert.Equal(t, "dito.entity.process_duration_ns", outputMetrics.At(1).Name())

		dataPoints := outputMetrics.At(0).Sum().DataPoints()
		assert.Equal(t, 3, dataPoints.Len())

		errorJob1MetricFound := false
		okJob1MetricFound := false
		okJob2MetricFound := false

		for _, dp := range dataPoints.All() {
			job, jobExists := dp.Attributes().Get(JOB_KEY_VALUE)
			status, statusExists := dp.Attributes().Get("dito.entity.status_code")

			assert.True(t, jobExists)
			assert.True(t, statusExists)

			if job.AsString() == "1" && status.AsString() == ptrace.StatusCodeError.String() {
				errorJob1MetricFound = true
				assert.Equal(t, int64(1), dp.IntValue())
			} else if job.AsString() == "1" && status.AsString() == ptrace.StatusCodeUnset.String() {
				okJob1MetricFound = true
				assert.Equal(t, int64(5), dp.IntValue())
			} else if job.AsString() == "2" && status.AsString() == ptrace.StatusCodeUnset.String() {
				okJob2MetricFound = true
				assert.Equal(t, int64(1), dp.IntValue())
			}
		}

		assert.True(t, errorJob1MetricFound)
		assert.True(t, okJob1MetricFound)
		assert.True(t, okJob2MetricFound)
	})

	connector.Shutdown(ctx)
}
