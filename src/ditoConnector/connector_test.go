package dito

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vibeus/opentelemetry-collector/confmap/xconfmap"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"
)

const (
	ENTITY_KEY_VALUE   = "dito.key"
	JOB_KEY_VALUE      = "dito.job_id"
	BATCH_TIMEOUT      = 100 * time.Millisecond
	MAX_CACHE_DURATION = 300 * time.Millisecond
	TEST_WAIT          = 500 * time.Millisecond
)

func TestTracesConnector(t *testing.T) {
	// Create a test consumer that captures traces
	tracesConsumer := &consumertest.TracesSink{}

	cfg := createDefaultConfig().(*Config)
	cfg.BatchTimeout = BATCH_TIMEOUT
	cfg.MaxCacheDuration = MAX_CACHE_DURATION

	connector, err := newTraceConnector(zap.NewNop(), cfg, tracesConsumer)
	require.NoError(t, err)
	ctx := context.Background()

	err = connector.Start(ctx, nil)
	require.NoError(t, err)

	t.Run("no entity spans", func(t *testing.T) {
		// arrange
		tracesConsumer.Reset()
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
		outputTraces := tracesConsumer.AllTraces()
		assert.Equal(t, 0, len(outputTraces))
	})

	t.Run("one entity span one job span", func(t *testing.T) {
		// arrange
		tracesConsumer.Reset()
		connector.sharedCache.reset()

		traces := ptrace.NewTraces()
		inputResourceSpan := traces.ResourceSpans().AppendEmpty()
		inputScopeSpan := inputResourceSpan.ScopeSpans().AppendEmpty()
		inputJobSpan := inputScopeSpan.Spans().AppendEmpty()
		inputSpan := inputScopeSpan.Spans().AppendEmpty()

		jobSpanId := generateSpanID()
		inputJobSpan.SetSpanID(jobSpanId)
		inputJobSpan.Attributes().PutInt(JOB_KEY_VALUE, 1)

		inputSpan.SetSpanID(generateSpanID())
		inputSpan.SetParentSpanID(jobSpanId)
		inputSpan.Attributes().PutInt(ENTITY_KEY_VALUE, 1)
		inputSpan.Attributes().PutStr("test.key", "test.value")

		// act
		err := connector.ConsumeTraces(ctx, traces)
		require.NoError(t, err)
		time.Sleep(TEST_WAIT) // allow worker to process

		// assert
		spanTrees := buildSpanTrees(tracesConsumer.AllTraces())

		require.NotNil(t, spanTrees)
		assert.Len(t, spanTrees, 1)
		assert.Len(t, spanTrees[0].children, 1)
		assert.Len(t, spanTrees[0].children[0].children, 1)

		rootSpan := spanTrees[0].span
		jobSpan := spanTrees[0].children[0].span
		entitySpan := spanTrees[0].children[0].children[0].span

		assertAllUnequal(t, []any{rootSpan.ParentSpanID(), jobSpan.ParentSpanID(), entitySpan.ParentSpanID()})
		assert.True(t, rootSpan.ParentSpanID().IsEmpty())
		assert.Equal(t, rootSpan.SpanID(), jobSpan.ParentSpanID())
		assert.Equal(t, jobSpan.SpanID(), entitySpan.ParentSpanID())

		actualJob, actualJobExists := jobSpan.Attributes().Get(JOB_KEY_VALUE)
		actualKey, actualKeyExists := entitySpan.Attributes().Get(ENTITY_KEY_VALUE)
		actualTest, actualTestExists := entitySpan.Attributes().Get("test.key")
		assert.True(t, actualJobExists)
		assert.True(t, actualKeyExists)
		assert.True(t, actualTestExists)
		assert.Equal(t, "1", actualJob.AsString())
		assert.Equal(t, "1", actualKey.AsString())
		assert.Equal(t, "test.value", actualTest.AsString())
	})

	t.Run("two same entity spans one job span", func(t *testing.T) {
		// arrange
		tracesConsumer.Reset()
		connector.sharedCache.reset()

		traces := ptrace.NewTraces()
		inputResourceSpan := traces.ResourceSpans().AppendEmpty()
		inputScopeSpan := inputResourceSpan.ScopeSpans().AppendEmpty()
		inputJobSpan := inputScopeSpan.Spans().AppendEmpty()
		inputSpan1 := inputScopeSpan.Spans().AppendEmpty()
		inputSpan2 := inputScopeSpan.Spans().AppendEmpty()

		jobSpanId := generateSpanID()
		inputJobSpan.SetSpanID(jobSpanId)
		inputJobSpan.Attributes().PutInt(JOB_KEY_VALUE, 1)

		inputSpan1.SetSpanID(generateSpanID())
		inputSpan1.SetParentSpanID(jobSpanId)
		inputSpan1.Attributes().PutInt(ENTITY_KEY_VALUE, 1)
		inputSpan1.Attributes().PutStr("test.key", "test.value")

		inputSpan2.SetSpanID(generateSpanID())
		inputSpan2.SetParentSpanID(jobSpanId)
		inputSpan2.Attributes().PutInt(ENTITY_KEY_VALUE, 1)
		inputSpan2.Attributes().PutStr("test.key", "test.value")

		// act
		err := connector.ConsumeTraces(ctx, traces)
		require.NoError(t, err)
		time.Sleep(TEST_WAIT) // allow worker to process

		// assert
		spanTrees := buildSpanTrees(tracesConsumer.AllTraces())

		require.NotNil(t, spanTrees)
		assert.Len(t, spanTrees, 1)
		assert.Len(t, spanTrees[0].children, 1)
		assert.Len(t, spanTrees[0].children[0].children, 2)

		rootSpan := spanTrees[0].span
		jobSpan := spanTrees[0].children[0].span
		entitySpan1 := spanTrees[0].children[0].children[0].span
		entitySpan2 := spanTrees[0].children[0].children[1].span

		assertAllUnequal(t, []any{rootSpan.SpanID(), jobSpan.SpanID(), entitySpan1.SpanID(), entitySpan2.SpanID()})
		assert.True(t, rootSpan.ParentSpanID().IsEmpty())
		assert.Equal(t, rootSpan.SpanID(), jobSpan.ParentSpanID())
		assert.Equal(t, jobSpan.SpanID(), entitySpan1.ParentSpanID())
		assert.Equal(t, jobSpan.SpanID(), entitySpan2.ParentSpanID())

		actualJob, actualJobExists := jobSpan.Attributes().Get(JOB_KEY_VALUE)
		actualKey1, actualKeyExists1 := entitySpan1.Attributes().Get(ENTITY_KEY_VALUE)
		actualTest1, actualTestExists1 := entitySpan1.Attributes().Get("test.key")
		actualKey2, actualKeyExists2 := entitySpan2.Attributes().Get(ENTITY_KEY_VALUE)
		actualTest2, actualTestExists2 := entitySpan2.Attributes().Get("test.key")
		assert.True(t, actualJobExists)
		assert.True(t, actualKeyExists1)
		assert.True(t, actualTestExists1)
		assert.True(t, actualKeyExists2)
		assert.True(t, actualTestExists2)
		assert.Equal(t, "1", actualJob.AsString())
		assert.Equal(t, "1", actualKey1.AsString())
		assert.Equal(t, "test.value", actualTest1.AsString())
		assert.Equal(t, "1", actualKey2.AsString())
		assert.Equal(t, "test.value", actualTest2.AsString())
	})

	t.Run("two different entity spans one job span", func(t *testing.T) {
		// arrange
		tracesConsumer.Reset()
		connector.sharedCache.reset()

		traces := ptrace.NewTraces()
		inputResourceSpan := traces.ResourceSpans().AppendEmpty()
		inputScopeSpan := inputResourceSpan.ScopeSpans().AppendEmpty()
		inputJobSpan := inputScopeSpan.Spans().AppendEmpty()
		inputSpan1 := inputScopeSpan.Spans().AppendEmpty()
		inputSpan2 := inputScopeSpan.Spans().AppendEmpty()

		jobSpanId := generateSpanID()
		inputJobSpan.SetSpanID(jobSpanId)
		inputJobSpan.Attributes().PutInt(JOB_KEY_VALUE, 1)

		inputSpan1.SetSpanID(generateSpanID())
		inputSpan1.SetParentSpanID(jobSpanId)
		inputSpan1.Attributes().PutInt(ENTITY_KEY_VALUE, 1)
		inputSpan1.Attributes().PutStr("test.key", "test.value")

		inputSpan2.SetSpanID(generateSpanID())
		inputSpan2.SetParentSpanID(jobSpanId)
		inputSpan2.Attributes().PutInt(ENTITY_KEY_VALUE, 2)
		inputSpan2.Attributes().PutStr("test.key", "test.value")

		// act
		err := connector.ConsumeTraces(ctx, traces)
		require.NoError(t, err)
		time.Sleep(TEST_WAIT) // allow worker to process

		// assert
		spanTrees := buildSpanTrees(tracesConsumer.AllTraces())

		assert.Len(t, spanTrees, 2)
		assert.Len(t, spanTrees[0].children, 1)
		assert.Len(t, spanTrees[1].children, 1)
		assert.Len(t, spanTrees[0].children[0].children, 1)
		assert.Len(t, spanTrees[1].children[0].children, 1)

		jobSpan1 := spanTrees[0].children[0].span
		jobSpan2 := spanTrees[1].children[0].span
		entitySpan1 := spanTrees[0].children[0].children[0].span
		entitySpan2 := spanTrees[1].children[0].children[0].span

		actualJob1, actualJobExists1 := jobSpan1.Attributes().Get(JOB_KEY_VALUE)
		actualJob2, actualJobExists2 := jobSpan2.Attributes().Get(JOB_KEY_VALUE)
		actualTest1, actualTestExists1 := entitySpan1.Attributes().Get("test.key")
		actualTest2, actualTestExists2 := entitySpan2.Attributes().Get("test.key")
		assert.True(t, actualJobExists1)
		assert.True(t, actualJobExists2)
		assert.True(t, actualTestExists1)
		assert.True(t, actualTestExists2)
		assert.Equal(t, "1", actualJob1.AsString())
		assert.Equal(t, "1", actualJob2.AsString())
		assert.Equal(t, "test.value", actualTest1.AsString())
		assert.Equal(t, "test.value", actualTest2.AsString())
	})

	t.Run("two same entity spans two job spans", func(t *testing.T) {
		// arrange
		tracesConsumer.Reset()
		connector.sharedCache.reset()

		traces := ptrace.NewTraces()
		inputResourceSpan := traces.ResourceSpans().AppendEmpty()
		inputScopeSpan := inputResourceSpan.ScopeSpans().AppendEmpty()
		inputJobSpan1 := inputScopeSpan.Spans().AppendEmpty()
		inputJobSpan2 := inputScopeSpan.Spans().AppendEmpty()
		inputSpan1 := inputScopeSpan.Spans().AppendEmpty()
		inputSpan2 := inputScopeSpan.Spans().AppendEmpty()

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
		inputSpan2.Attributes().PutInt(ENTITY_KEY_VALUE, 1)
		inputSpan2.Attributes().PutStr("test.key", "test.value")

		// act
		err := connector.ConsumeTraces(ctx, traces)
		require.NoError(t, err)
		time.Sleep(TEST_WAIT) // allow worker to process

		// assert
		spanTrees := buildSpanTrees(tracesConsumer.AllTraces())

		assert.Len(t, spanTrees, 1)
		assert.Len(t, spanTrees[0].children, 2)
		assert.Len(t, spanTrees[0].children[0].children, 1)
		assert.Len(t, spanTrees[0].children[1].children, 1)

		jobSpan1 := spanTrees[0].children[0].span
		jobSpan2 := spanTrees[0].children[1].span
		entitySpan1 := spanTrees[0].children[0].children[0].span
		entitySpan2 := spanTrees[0].children[1].children[0].span

		actualJob1, actualJobExists1 := jobSpan1.Attributes().Get(JOB_KEY_VALUE)
		actualJob2, actualJobExists2 := jobSpan2.Attributes().Get(JOB_KEY_VALUE)
		actualTest1, actualTestExists1 := entitySpan1.Attributes().Get("test.key")
		actualTest2, actualTestExists2 := entitySpan2.Attributes().Get("test.key")
		assert.True(t, actualJobExists1)
		assert.True(t, actualJobExists2)
		assert.True(t, actualTestExists1)
		assert.True(t, actualTestExists2)
		assert.NotEqual(t, actualJob1.AsString(), actualJob2.AsString())
		assert.Equal(t, "test.value", actualTest1.AsString())
		assert.Equal(t, "test.value", actualTest2.AsString())
	})

	t.Run("two different entity spans two job spans", func(t *testing.T) {
		// arrange
		tracesConsumer.Reset()
		connector.sharedCache.reset()

		traces := ptrace.NewTraces()
		inputResourceSpan := traces.ResourceSpans().AppendEmpty()
		inputScopeSpan := inputResourceSpan.ScopeSpans().AppendEmpty()
		inputJobSpan1 := inputScopeSpan.Spans().AppendEmpty()
		inputJobSpan2 := inputScopeSpan.Spans().AppendEmpty()
		inputSpan1 := inputScopeSpan.Spans().AppendEmpty()
		inputSpan2 := inputScopeSpan.Spans().AppendEmpty()

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

		// act
		err := connector.ConsumeTraces(ctx, traces)
		require.NoError(t, err)
		time.Sleep(TEST_WAIT) // allow worker to process

		// assert
		spanTrees := buildSpanTrees(tracesConsumer.AllTraces())

		assert.Len(t, spanTrees, 2)
		assert.Len(t, spanTrees[0].children, 1)
		assert.Len(t, spanTrees[1].children, 1)
		assert.Len(t, spanTrees[0].children[0].children, 1)
		assert.Len(t, spanTrees[1].children[0].children, 1)

		jobSpan1 := spanTrees[0].children[0].span
		jobSpan2 := spanTrees[1].children[0].span
		entitySpan1 := spanTrees[0].children[0].children[0].span
		entitySpan2 := spanTrees[1].children[0].children[0].span

		actualJob1, actualJobExists1 := jobSpan1.Attributes().Get(JOB_KEY_VALUE)
		actualJob2, actualJobExists2 := jobSpan2.Attributes().Get(JOB_KEY_VALUE)
		actualTest1, actualTestExists1 := entitySpan1.Attributes().Get("test.key")
		actualTest2, actualTestExists2 := entitySpan2.Attributes().Get("test.key")
		assert.True(t, actualJobExists1)
		assert.True(t, actualJobExists2)
		assert.True(t, actualTestExists1)
		assert.True(t, actualTestExists2)
		assert.NotEqual(t, actualJob1.AsString(), actualJob2.AsString())
		assert.Equal(t, "test.value", actualTest1.AsString())
		assert.Equal(t, "test.value", actualTest2.AsString())
	})

	t.Run("one entity span one job span split", func(t *testing.T) {
		// arrange
		tracesConsumer.Reset()
		connector.sharedCache.reset()

		traces1 := ptrace.NewTraces()
		inputResourceSpan1 := traces1.ResourceSpans().AppendEmpty()
		inputScopeSpan1 := inputResourceSpan1.ScopeSpans().AppendEmpty()
		inputSpan := inputScopeSpan1.Spans().AppendEmpty()

		jobSpanId := generateSpanID()
		inputSpan.SetSpanID(jobSpanId)
		inputSpan.SetParentSpanID(jobSpanId)
		inputSpan.Attributes().PutInt(ENTITY_KEY_VALUE, 1)
		inputSpan.Attributes().PutStr("test.key", "test.value")

		traces2 := ptrace.NewTraces()
		inputResourceSpan2 := traces2.ResourceSpans().AppendEmpty()
		inputScopeSpan2 := inputResourceSpan2.ScopeSpans().AppendEmpty()
		inputJobSpan := inputScopeSpan2.Spans().AppendEmpty()

		inputJobSpan.SetSpanID(jobSpanId)
		inputJobSpan.Attributes().PutInt(JOB_KEY_VALUE, 1)

		// act
		err := connector.ConsumeTraces(ctx, traces1)
		require.NoError(t, err)
		err = connector.ConsumeTraces(ctx, traces2)
		require.NoError(t, err)
		time.Sleep(TEST_WAIT) // allow worker to process

		// assert
		spanTrees := buildSpanTrees(tracesConsumer.AllTraces())

		require.NotNil(t, spanTrees)
		assert.Len(t, spanTrees, 1)
		assert.Len(t, spanTrees[0].children, 1)
		assert.Len(t, spanTrees[0].children[0].children, 1)

		rootSpan := spanTrees[0].span
		jobSpan := spanTrees[0].children[0].span
		entitySpan := spanTrees[0].children[0].children[0].span

		assertAllUnequal(t, []any{rootSpan.ParentSpanID(), jobSpan.ParentSpanID(), entitySpan.ParentSpanID()})
		assert.True(t, rootSpan.ParentSpanID().IsEmpty())
		assert.Equal(t, rootSpan.SpanID(), jobSpan.ParentSpanID())
		assert.Equal(t, jobSpan.SpanID(), entitySpan.ParentSpanID())

		actualJob, actualJobExists := jobSpan.Attributes().Get(JOB_KEY_VALUE)
		actualKey, actualKeyExists := entitySpan.Attributes().Get(ENTITY_KEY_VALUE)
		actualTest, actualTestExists := entitySpan.Attributes().Get("test.key")
		assert.True(t, actualJobExists)
		assert.True(t, actualKeyExists)
		assert.True(t, actualTestExists)
		assert.Equal(t, "1", actualJob.AsString())
		assert.Equal(t, "1", actualKey.AsString())
		assert.Equal(t, "test.value", actualTest.AsString())
	})

	t.Run("different entities sampling", func(t *testing.T) {
		// arrange
		tracesConsumer.Reset()
		connector.sharedCache.reset()

		traces := ptrace.NewTraces()
		inputResourceSpan := traces.ResourceSpans().AppendEmpty()
		inputScopeSpan := inputResourceSpan.ScopeSpans().AppendEmpty()

		for i := range int64(100) {
			inputSpan := inputScopeSpan.Spans().AppendEmpty()
			inputSpan.SetSpanID(generateSpanID())
			inputSpan.Attributes().PutInt(ENTITY_KEY_VALUE, i)
		}

		// act
		err = connector.ConsumeTraces(ctx, traces)
		require.NoError(t, err)
		time.Sleep(TEST_WAIT) // allow worker to process

		// assert
		spanTrees := buildSpanTrees(tracesConsumer.AllTraces())

		require.NotNil(t, spanTrees)
		assert.Len(t, spanTrees, 100)

		for _, rootSpan := range spanTrees {
			assert.Len(t, rootSpan.children, 1)
		}
	})

	t.Run("same entities sampling with fraction 2", func(t *testing.T) {
		// arrange
		tracesConsumer.Reset()
		connector.sharedCache.reset()
		beforeFraction := connector.config.SamplingFraction
		defer func() { connector.config.SamplingFraction = beforeFraction }()
		connector.config.SamplingFraction = 2

		traces := ptrace.NewTraces()
		inputResourceSpan := traces.ResourceSpans().AppendEmpty()
		inputScopeSpan := inputResourceSpan.ScopeSpans().AppendEmpty()

		for range 100 {
			inputSpan := inputScopeSpan.Spans().AppendEmpty()
			inputSpan.SetSpanID(generateSpanID())
			inputSpan.Attributes().PutInt(ENTITY_KEY_VALUE, 1)
		}

		// act
		err = connector.ConsumeTraces(ctx, traces)
		require.NoError(t, err)
		time.Sleep(TEST_WAIT) // allow worker to process

		// assert
		spanTrees := buildSpanTrees(tracesConsumer.AllTraces())

		require.NotNil(t, spanTrees)
		assert.Len(t, spanTrees, 1)
		assert.Len(t, spanTrees[0].children, 50)
	})

	t.Run("same entities sampling with fraction 7", func(t *testing.T) {
		// arrange
		tracesConsumer.Reset()
		connector.sharedCache.reset()
		beforeFraction := connector.config.SamplingFraction
		defer func() { connector.config.SamplingFraction = beforeFraction }()
		connector.config.SamplingFraction = 7

		traces := ptrace.NewTraces()
		inputResourceSpan := traces.ResourceSpans().AppendEmpty()
		inputScopeSpan := inputResourceSpan.ScopeSpans().AppendEmpty()

		for range 100 {
			inputSpan := inputScopeSpan.Spans().AppendEmpty()
			inputSpan.SetSpanID(generateSpanID())
			inputSpan.Attributes().PutInt(ENTITY_KEY_VALUE, 1)
		}

		// act
		err = connector.ConsumeTraces(ctx, traces)
		require.NoError(t, err)
		time.Sleep(TEST_WAIT) // allow worker to process

		// assert
		spanTrees := buildSpanTrees(tracesConsumer.AllTraces())

		require.NotNil(t, spanTrees)
		assert.Len(t, spanTrees, 1)
		assert.Len(t, spanTrees[0].children, 15)
	})

	connector.Shutdown(ctx)
}

func TestMetricsConnector(t *testing.T) {
	// Create a test consumer that captures metrics
	metricsConsumer := &consumertest.MetricsSink{}
	cfg := createDefaultConfig().(*Config)
	cfg.BatchTimeout = BATCH_TIMEOUT
	cfg.MaxCacheDuration = MAX_CACHE_DURATION

	connector, err := newMetricConnector(zap.NewNop(), cfg, metricsConsumer)
	require.NoError(t, err)

	ctx := context.Background()

	err = connector.Start(ctx, nil)
	require.NoError(t, err)

	t.Run("no entity spans", func(t *testing.T) {
		// arrange
		metricsConsumer.Reset()

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
		assert.Equal(t, 1, outputMetrics.Len())
		assert.Equal(t, "dito.entity.count", outputMetrics.At(0).Name())

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

func TestConnectorCapabilities(t *testing.T) {
	t.Run("dito traces connector capabilities", func(t *testing.T) {
		connector := &traceConnector{}
		capabilities := connector.Capabilities()
		assert.False(t, capabilities.MutatesData)
	})

	t.Run("dito metrics connector capabilities", func(t *testing.T) {
		connector := &metricConnector{}
		capabilities := connector.Capabilities()
		assert.False(t, capabilities.MutatesData)
	})
}

func TestCreateDefaultConfig(t *testing.T) {
	cfg := createDefaultConfig()
	assert.NotNil(t, cfg)
	err := xconfmap.Validate(cfg)
	assert.NoError(t, err)

	exampleConfig := cfg.(*Config)
	assert.Equal(t, 1, exampleConfig.SamplingFraction)
	assert.Equal(t, ENTITY_KEY_VALUE, exampleConfig.EntityKey)
	assert.Equal(t, JOB_KEY_VALUE, exampleConfig.JobKey)
	assert.Equal(t, time.Hour, exampleConfig.MaxCacheDuration)
}

func TestConfigValidation(t *testing.T) {
	t.Run("SamplingFraction 0", func(t *testing.T) {
		cfg := &Config{
			EntityKey:        ENTITY_KEY_VALUE,
			JobKey:           JOB_KEY_VALUE,
			SamplingFraction: 0,
			MaxCacheDuration: time.Hour,
		}
		err := xconfmap.Validate(cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "sampling_fraction must be greater than 0")
	})

	t.Run("no JobKey", func(t *testing.T) {
		cfg := &Config{
			EntityKey:        ENTITY_KEY_VALUE,
			JobKey:           "",
			SamplingFraction: 1,
			MaxCacheDuration: time.Hour,
		}
		err := xconfmap.Validate(cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "job_key must be set")
	})

	t.Run("no EntityKey", func(t *testing.T) {
		cfg := &Config{
			EntityKey:        "",
			JobKey:           JOB_KEY_VALUE,
			SamplingFraction: 1,
			MaxCacheDuration: time.Hour,
		}
		err := xconfmap.Validate(cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "entity_key must be set")
	})

	t.Run("negative MaxCacheDuration", func(t *testing.T) {
		cfg := &Config{
			EntityKey:        ENTITY_KEY_VALUE,
			JobKey:           JOB_KEY_VALUE,
			SamplingFraction: 1,
			MaxCacheDuration: -1,
		}
		err := xconfmap.Validate(cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "max_cache_duration must be positive")
	})
}

func assertAllUnequal(t *testing.T, items []any) {
	for i := range items {
		for j := i + 1; j < len(items); j++ {
			assert.NotEqual(t, items[i], items[j])
		}
	}
}

type spanTree struct {
	span     ptrace.Span
	children []*spanTree
}

func buildSpanTrees(traces []ptrace.Traces) []*spanTree {
	if len(traces) == 0 {
		return nil
	}

	var spans []*ptrace.Span
	var roots []*spanTree

	for _, trace := range traces {
		if trace.ResourceSpans().Len() == 0 || trace.ResourceSpans().At(0).ScopeSpans().Len() == 0 {
			continue
		}

		traceSpans := trace.ResourceSpans().At(0).ScopeSpans().At(0).Spans()

		for i := 0; i < traceSpans.Len(); i++ {
			span := traceSpans.At(i)
			spans = append(spans, &span)
		}
	}

	for _, span := range spans {
		if span.ParentSpanID().IsEmpty() {
			root := &spanTree{span: *span}
			root.children = getChildSpans(span, spans)
			roots = append(roots, root)
		}
	}

	return roots
}

func getChildSpans(parent *ptrace.Span, allSpans []*ptrace.Span) []*spanTree {
	var children []*spanTree

	for _, span := range allSpans {

		if span.ParentSpanID() == parent.SpanID() {
			child := &spanTree{span: *span}
			child.children = getChildSpans(span, allSpans)
			children = append(children, child)
		}
	}

	return children
}
