package dito

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

const (
	ENTITY_KEY_VALUE   = "dito.key"
	JOB_KEY_VALUE      = "dito.job_id"
	MAX_CACHE_DURATION = 20 * time.Millisecond
	TEST_WAIT          = 100 * time.Millisecond
)

func (sc *sharedCache) reset() {
	for _, sh := range sc.entityShards {
		sh.mu.Lock()
		sh.entityInfoCache = make(map[string]*entityInfoCacheItem)
		sh.mu.Unlock()
	}

	for _, sh := range sc.jobShards {
		sh.mu.Lock()
		sh.jobCache = make(map[pcommon.SpanID]*jobCacheItem)
		sh.mu.Unlock()
	}

	// Drain queues
	for len(sc.messageQueue) > 0 {
		<-sc.messageQueue
	}
}

func (t *traceConnector) reset() {
	for len(t.waitQueue) > 0 {
		<-t.waitQueue
	}

	for len(t.outputQueue) > 0 {
		<-t.outputQueue
	}

	for len(t.internalOutputQueue) > 0 {
		<-t.internalOutputQueue
	}

	t.currentRootSpan = nil

	t.sharedCache.reset()
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
