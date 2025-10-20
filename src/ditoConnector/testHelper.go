package dito

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

const (
	ENTITY_KEY_VALUE   = "dito.key"
	JOB_KEY_VALUE      = "dito.job_id"
	MAX_CACHE_DURATION = 20 * time.Millisecond
	TEST_WAIT          = 150 * time.Millisecond
)

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
