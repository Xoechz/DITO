package dito

import (
	"fmt"
	"time"
)

type Config struct {
	EntityKey                string        `mapstructure:"entity_key"`
	JobKey                   string        `mapstructure:"job_key"`
	MaxCacheDuration         time.Duration `mapstructure:"max_cache_duration"`
	CacheShardCount          int           `mapstructure:"cache_shard_count"`
	QueueSize                int           `mapstructure:"queue_size"`
	WorkerCount              int           `mapstructure:"worker_count"`
	NonErrorSamplingFraction int           `mapstructure:"non_error_sampling_fraction"`
}

func (cfg *Config) Validate() error {
	if cfg.EntityKey == "" {
		return fmt.Errorf("entity_key must be set")
	}

	if cfg.JobKey == "" {
		return fmt.Errorf("job_key must be set")
	}

	if cfg.MaxCacheDuration <= 0 {
		return fmt.Errorf("max_cache_duration must be positive")
	}

	if cfg.NonErrorSamplingFraction < 1 {
		return fmt.Errorf("non_error_sampling_fraction must be greater than 0")
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

	return nil
}
