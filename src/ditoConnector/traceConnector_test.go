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

func TestTraceConnectorCapabilities(t *testing.T) {
	t.Run("dito traces connector capabilities", func(t *testing.T) {
		connector := &traceConnector{}
		capabilities := connector.Capabilities()
		assert.False(t, capabilities.MutatesData)
	})
}
func TestTracesConnector(t *testing.T) {
	// Create a test consumer that captures traces
	tracesConsumer := &consumertest.TracesSink{}

	cfg := createDefaultConfig().(*Config)
	cfg.MaxCacheDuration = MAX_CACHE_DURATION

	connector, err := newTraceConnector(zap.NewNop(), cfg, tracesConsumer)
	require.NoError(t, err)
	ctx := context.Background()

	err = connector.Start(ctx, nil)
	require.NoError(t, err)

	t.Run("no entity spans", func(t *testing.T) {
		// arrange
		tracesConsumer.Reset()
		connector.reset()

		traces := ptrace.NewTraces()
		inputResourceSpan := traces.ResourceSpans().AppendEmpty()
		inputScopeSpan := inputResourceSpan.ScopeSpans().AppendEmpty()
		inputSpan := inputScopeSpan.Spans().AppendEmpty()

		inputSpan.SetSpanID(generateSpanID())
		inputSpan.Attributes().PutInt("other", 1)

		// act
		err := connector.ConsumeTraces(ctx, traces)
		require.NoError(t, err)
		time.Sleep(TEST_WAIT)   // allow worker to process
		connector.flushOutput() // ensure all spans are flushed

		// assert
		outputTraces := tracesConsumer.AllTraces()
		assert.Equal(t, 0, len(outputTraces))
	})

	t.Run("one entity span one job span", func(t *testing.T) {
		// arrange
		tracesConsumer.Reset()
		connector.reset()

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
		time.Sleep(TEST_WAIT)   // allow worker to process
		connector.flushOutput() // ensure all spans are flushed

		// assert
		spanTrees := buildSpanTrees(tracesConsumer.AllTraces())

		require.NotNil(t, spanTrees)
		assert.Len(t, spanTrees, 2)
		assert.Len(t, spanTrees[0].children, 1)
		assert.Len(t, spanTrees[0].children[0].children, 1)
		assert.Len(t, spanTrees[1].children, 2)

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
		connector.reset()

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
		time.Sleep(TEST_WAIT)   // allow worker to process
		connector.flushOutput() // ensure all spans are flushed

		// assert
		spanTrees := buildSpanTrees(tracesConsumer.AllTraces())

		require.NotNil(t, spanTrees)
		assert.Len(t, spanTrees, 2)
		assert.Len(t, spanTrees[0].children, 1)
		assert.Len(t, spanTrees[0].children[0].children, 2)
		assert.Len(t, spanTrees[1].children, 2)

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
		connector.reset()

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
		time.Sleep(TEST_WAIT)   // allow worker to process
		connector.flushOutput() // ensure all spans are flushed

		// assert
		spanTrees := buildSpanTrees(tracesConsumer.AllTraces())

		assert.Len(t, spanTrees, 3)
		assert.Len(t, spanTrees[0].children, 1)
		assert.Len(t, spanTrees[1].children, 1)
		assert.Len(t, spanTrees[0].children[0].children, 1)
		assert.Len(t, spanTrees[1].children[0].children, 1)
		assert.Len(t, spanTrees[2].children, 2)

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
		connector.reset()

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
		time.Sleep(TEST_WAIT)   // allow worker to process
		connector.flushOutput() // ensure all spans are flushed

		// assert
		spanTrees := buildSpanTrees(tracesConsumer.AllTraces())

		assert.Len(t, spanTrees, 2)
		assert.Len(t, spanTrees[0].children, 2)
		assert.Len(t, spanTrees[0].children[0].children, 1)
		assert.Len(t, spanTrees[0].children[1].children, 1)
		assert.Len(t, spanTrees[1].children, 2)

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
		connector.reset()

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
		time.Sleep(TEST_WAIT)   // allow worker to process
		connector.flushOutput() // ensure all spans are flushed

		// assert
		spanTrees := buildSpanTrees(tracesConsumer.AllTraces())

		assert.Len(t, spanTrees, 3)
		assert.Len(t, spanTrees[0].children, 1)
		assert.Len(t, spanTrees[1].children, 1)
		assert.Len(t, spanTrees[0].children[0].children, 1)
		assert.Len(t, spanTrees[1].children[0].children, 1)
		assert.Len(t, spanTrees[2].children, 2)

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
		connector.reset()

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
		time.Sleep(TEST_WAIT)   // allow worker to process
		connector.flushOutput() // ensure all spans are flushed

		// assert
		spanTrees := buildSpanTrees(tracesConsumer.AllTraces())

		require.NotNil(t, spanTrees)
		assert.Len(t, spanTrees, 2)
		assert.Len(t, spanTrees[0].children, 1)
		assert.Len(t, spanTrees[0].children[0].children, 1)
		assert.Len(t, spanTrees[1].children, 3)

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
		connector.reset()

		traces := ptrace.NewTraces()
		inputResourceSpan := traces.ResourceSpans().AppendEmpty()
		inputScopeSpan := inputResourceSpan.ScopeSpans().AppendEmpty()
		jobSpan := inputScopeSpan.Spans().AppendEmpty()
		jobSpan.SetSpanID(generateSpanID())
		jobSpan.Attributes().PutInt(JOB_KEY_VALUE, 1)

		for i := range int64(100) {
			inputSpan := inputScopeSpan.Spans().AppendEmpty()
			inputSpan.SetSpanID(generateSpanID())
			inputSpan.SetParentSpanID(jobSpan.SpanID())
			inputSpan.Attributes().PutInt(ENTITY_KEY_VALUE, i)
		}

		// act
		err = connector.ConsumeTraces(ctx, traces)
		require.NoError(t, err)
		time.Sleep(TEST_WAIT)   // allow worker to process
		connector.flushOutput() // ensure all spans are flushed

		// assert
		spanTrees := buildSpanTrees(tracesConsumer.AllTraces())

		require.NotNil(t, spanTrees)
		require.Len(t, spanTrees, 101)

		assert.Len(t, spanTrees[100].children, 2)

		for i := range 100 {
			assert.Len(t, spanTrees[i].children, 1)
			assert.Len(t, spanTrees[i].children[0].children, 1)
		}
	})

	t.Run("same entities sampling with fraction 2", func(t *testing.T) {
		// arrange
		tracesConsumer.Reset()
		connector.reset()
		beforeFraction := connector.config.SamplingFraction
		defer func() { connector.config.SamplingFraction = beforeFraction }()
		connector.config.SamplingFraction = 2

		traces := ptrace.NewTraces()
		inputResourceSpan := traces.ResourceSpans().AppendEmpty()
		inputScopeSpan := inputResourceSpan.ScopeSpans().AppendEmpty()
		jobSpan := inputScopeSpan.Spans().AppendEmpty()
		jobSpan.SetSpanID(generateSpanID())
		jobSpan.Attributes().PutInt(JOB_KEY_VALUE, 1)

		for range 100 {
			inputSpan := inputScopeSpan.Spans().AppendEmpty()
			inputSpan.SetSpanID(generateSpanID())
			inputSpan.SetParentSpanID(jobSpan.SpanID())
			inputSpan.Attributes().PutInt(ENTITY_KEY_VALUE, 1)
		}

		// act
		err = connector.ConsumeTraces(ctx, traces)
		require.NoError(t, err)
		time.Sleep(TEST_WAIT)   // allow worker to process
		connector.flushOutput() // ensure all spans are flushed

		// assert
		spanTrees := buildSpanTrees(tracesConsumer.AllTraces())

		require.NotNil(t, spanTrees)
		assert.Len(t, spanTrees, 2)
		assert.Len(t, spanTrees[0].children, 1)
		assert.Len(t, spanTrees[0].children[0].children, 50)
		assert.Len(t, spanTrees[1].children, 2)
	})

	t.Run("same entities sampling with fraction 7", func(t *testing.T) {
		// arrange
		tracesConsumer.Reset()
		connector.reset()
		beforeFraction := connector.config.SamplingFraction
		defer func() { connector.config.SamplingFraction = beforeFraction }()
		connector.config.SamplingFraction = 7

		traces := ptrace.NewTraces()
		inputResourceSpan := traces.ResourceSpans().AppendEmpty()
		inputScopeSpan := inputResourceSpan.ScopeSpans().AppendEmpty()
		jobSpan := inputScopeSpan.Spans().AppendEmpty()
		jobSpan.SetSpanID(generateSpanID())
		jobSpan.Attributes().PutInt(JOB_KEY_VALUE, 1)

		for range 100 {
			inputSpan := inputScopeSpan.Spans().AppendEmpty()
			inputSpan.SetSpanID(generateSpanID())
			inputSpan.SetParentSpanID(jobSpan.SpanID())
			inputSpan.Attributes().PutInt(ENTITY_KEY_VALUE, 1)
		}

		// act
		err = connector.ConsumeTraces(ctx, traces)
		require.NoError(t, err)
		time.Sleep(TEST_WAIT)   // allow worker to process
		connector.flushOutput() // ensure all spans are flushed

		// assert
		spanTrees := buildSpanTrees(tracesConsumer.AllTraces())

		require.NotNil(t, spanTrees)
		assert.Len(t, spanTrees, 2)
		assert.Len(t, spanTrees[0].children, 1)
		assert.Len(t, spanTrees[0].children[0].children, 15)
		assert.Len(t, spanTrees[1].children, 2)
	})

	t.Run("one entity span one distant job span", func(t *testing.T) {
		// arrange
		tracesConsumer.Reset()
		connector.reset()

		traces := ptrace.NewTraces()
		inputResourceSpan := traces.ResourceSpans().AppendEmpty()
		inputScopeSpan := inputResourceSpan.ScopeSpans().AppendEmpty()
		inputJobSpan := inputScopeSpan.Spans().AppendEmpty()
		inputSpan := inputScopeSpan.Spans().AppendEmpty()

		parentSpanID := generateSpanID()
		jobSpanId := parentSpanID

		inputJobSpan.SetSpanID(parentSpanID)
		inputJobSpan.Attributes().PutInt(JOB_KEY_VALUE, 1)

		for range 10 {
			intermediateSpan := inputScopeSpan.Spans().AppendEmpty()

			intermediateSpan.SetSpanID(generateSpanID())
			intermediateSpan.SetParentSpanID(parentSpanID)

			parentSpanID = intermediateSpan.SpanID()
		}

		inputSpan.SetSpanID(generateSpanID())
		inputSpan.SetParentSpanID(parentSpanID)
		inputSpan.Attributes().PutInt(ENTITY_KEY_VALUE, 1)
		inputSpan.Attributes().PutStr("dito.job_span_id", jobSpanId.String())
		inputSpan.Attributes().PutStr("test.key", "test.value")

		// act
		err := connector.ConsumeTraces(ctx, traces)
		require.NoError(t, err)
		time.Sleep(TEST_WAIT)   // allow worker to process
		connector.flushOutput() // ensure all spans are flushed

		// assert
		spanTrees := buildSpanTrees(tracesConsumer.AllTraces())

		require.NotNil(t, spanTrees)
		assert.Len(t, spanTrees, 2)
		assert.Len(t, spanTrees[0].children, 1)
		assert.Len(t, spanTrees[0].children[0].children, 1)
		assert.Len(t, spanTrees[1].children, 2)

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

	connector.Shutdown(ctx)
}
