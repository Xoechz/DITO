package dito

import (
	"crypto/rand"

	"go.opentelemetry.io/collector/pdata/pcommon"
)

func generateTraceID() pcommon.TraceID {
	var tid [16]byte
	_, err := rand.Read(tid[:])
	if err != nil {
		// handle error appropriately
	}

	return pcommon.TraceID(tid)
}

func generateSpanID() pcommon.SpanID {
	var sid [8]byte
	_, err := rand.Read(sid[:])
	if err != nil {
		// handle error appropriately
	}
	return pcommon.SpanID(sid)
}
