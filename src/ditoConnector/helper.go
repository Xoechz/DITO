package dito

import (
	"crypto/rand"
	"encoding/hex"

	"go.opentelemetry.io/collector/pdata/pcommon"
)

const (
	SERVICE_NAME = "dito"
)

func generateTraceID() pcommon.TraceID {
	var tid [16]byte
	rand.Read(tid[:])
	return pcommon.TraceID(tid)
}

func generateSpanID() pcommon.SpanID {
	var sid [8]byte
	rand.Read(sid[:])
	return pcommon.SpanID(sid)
}

func getSpanIDFromHexString(hexStr string) (pcommon.SpanID, error) {
	spanID := pcommon.SpanID{}

	byteArray, err := hex.DecodeString(hexStr)

	if err != nil {
		return spanID, err
	}

	spanID = pcommon.SpanID(byteArray)
	return spanID, nil
}
