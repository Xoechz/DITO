package dito

import (
	"crypto/rand"
	"encoding/hex"
	"hash/fnv"

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

func GetHashCode(r *pcommon.Resource) uint32 {
	hasher := fnv.New32a()

	r.Attributes().Range(func(k string, v pcommon.Value) bool {
		hasher.Write([]byte(k))
		hasher.Write([]byte(v.AsString()))
		return true
	})

	var droppedCountBytes [4]byte
	droppedCount := r.DroppedAttributesCount()
	droppedCountBytes[0] = byte(droppedCount >> 24)
	droppedCountBytes[1] = byte(droppedCount >> 16)
	droppedCountBytes[2] = byte(droppedCount >> 8)
	droppedCountBytes[3] = byte(droppedCount)
	hasher.Write(droppedCountBytes[:])

	return hasher.Sum32()
}
