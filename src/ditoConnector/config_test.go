package dito

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/confmap/xconfmap"
)

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
	assert.Equal(t, time.Hour*24*7, exampleConfig.EntityCacheDuration)
	assert.Equal(t, 32, exampleConfig.CacheShardCount)
	assert.Equal(t, 10000, exampleConfig.QueueSize)
	assert.Equal(t, 4, exampleConfig.WorkerCount)
	assert.Equal(t, 256, exampleConfig.BatchSize)
	assert.Equal(t, time.Minute, exampleConfig.BatchTimeout)
}

func TestConfigValidation(t *testing.T) {
	var cfg *Config

	t.Run("valid config", func(t *testing.T) {
		cfg = createDefaultConfig().(*Config)
		err := cfg.Validate()
		assert.NoError(t, err)
	})

	t.Run("invalid entity key", func(t *testing.T) {
		cfg = createDefaultConfig().(*Config)
		cfg.EntityKey = ""
		err := cfg.Validate()
		assert.Error(t, err)
	})

	t.Run("invalid job key", func(t *testing.T) {
		cfg = createDefaultConfig().(*Config)
		cfg.JobKey = ""
		err := cfg.Validate()
		assert.Error(t, err)
	})

	t.Run("invalid max cache duration", func(t *testing.T) {
		cfg = createDefaultConfig().(*Config)
		cfg.MaxCacheDuration = 0
		err := cfg.Validate()
		assert.Error(t, err)
	})

	t.Run("invalid entity cache duration", func(t *testing.T) {
		cfg = createDefaultConfig().(*Config)
		cfg.EntityCacheDuration = 0
		err := cfg.Validate()
		assert.Error(t, err)
	})

	t.Run("invalid baggage job key", func(t *testing.T) {
		cfg = createDefaultConfig().(*Config)
		cfg.BaggageJobKey = ""
		err := cfg.Validate()
		assert.Error(t, err)
	})

	t.Run("invalid sampling fraction", func(t *testing.T) {
		cfg = createDefaultConfig().(*Config)
		cfg.SamplingFraction = 0
		err := cfg.Validate()
		assert.Error(t, err)
	})

	t.Run("invalid cache shard count", func(t *testing.T) {
		cfg = createDefaultConfig().(*Config)
		cfg.CacheShardCount = 0
		err := cfg.Validate()
		assert.Error(t, err)
	})

	t.Run("invalid queue size", func(t *testing.T) {
		cfg = createDefaultConfig().(*Config)
		cfg.QueueSize = 0
		err := cfg.Validate()
		assert.Error(t, err)
	})

	t.Run("invalid worker count", func(t *testing.T) {
		cfg = createDefaultConfig().(*Config)
		cfg.WorkerCount = 0
		err := cfg.Validate()
		assert.Error(t, err)
	})

	t.Run("invalid batch size", func(t *testing.T) {
		cfg = createDefaultConfig().(*Config)
		cfg.BatchSize = 0
		err := cfg.Validate()
		assert.Error(t, err)
	})

	t.Run("invalid batch timeout", func(t *testing.T) {
		cfg = createDefaultConfig().(*Config)
		cfg.BatchTimeout = 0
		err := cfg.Validate()
		assert.Error(t, err)
	})
}
