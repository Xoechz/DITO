package dito

import (
	"fmt"
	"time"
)

type Config struct {
	EntityKey        string        `mapstructure:"entity_key"`
	JobKey           string        `mapstructure:"job_key"`
	BaggageJobKey    string        `mapstructure:"baggage_job_key"`
	MaxCacheDuration time.Duration `mapstructure:"max_cache_duration"`
	CacheShardCount  int           `mapstructure:"cache_shard_count"`
	QueueSize        int           `mapstructure:"queue_size"`
	WorkerCount      int           `mapstructure:"worker_count"`
	SamplingFraction int           `mapstructure:"sampling_fraction"`
	BatchSize        int           `mapstructure:"batch_size"`
	BatchTimeout     time.Duration `mapstructure:"batch_timeout"`
}

func (cfg *Config) Validate() error {
	if cfg.EntityKey == "" {
		return fmt.Errorf("entity_key must be set")
	}

	if cfg.JobKey == "" {
		return fmt.Errorf("job_key must be set")
	}

	if cfg.BaggageJobKey == "" {
		return fmt.Errorf("baggage_job_key must be set")
	}

	if cfg.MaxCacheDuration <= 0 {
		return fmt.Errorf("max_cache_duration must be positive")
	}

	if cfg.SamplingFraction < 1 {
		return fmt.Errorf("sampling_fraction must be greater than 0")
	}

	if cfg.CacheShardCount < 1 {
		return fmt.Errorf("cache_shard_count must be greater than 0")
	}

	if cfg.QueueSize < 1 {
		return fmt.Errorf("queue_size must be greater than 0")
	}

	if cfg.WorkerCount < 1 {
		return fmt.Errorf("worker_count must be greater than 0")
	}

	if cfg.BatchSize < 1 {
		return fmt.Errorf("batch_size must be greater than 0")
	}

	if cfg.BatchTimeout <= 0 {
		return fmt.Errorf("batch_timeout must be positive")
	}

	return nil
}
