package dito

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetSpanIDFromHexString(t *testing.T) {
	t.Run("valid hex string", func(t *testing.T) {
		hexStr := "0123456789abcdef"
		spanID, err := getSpanIDFromHexString(hexStr)
		require.NoError(t, err)
		assert.False(t, spanID.IsEmpty())
	})

	t.Run("invalid hex string", func(t *testing.T) {
		hexStr := "invalid_hex"
		spanID, err := getSpanIDFromHexString(hexStr)
		require.Error(t, err)
		assert.True(t, spanID.IsEmpty())
	})
}
