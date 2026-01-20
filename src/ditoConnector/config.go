package dito

import (
	"fmt"
	"time"
)

type Config struct {
	BaggageJobKey       string        `mapstructure:"baggage_job_key"`
	BatchSize           int           `mapstructure:"batch_size"`
	BatchTimeout        time.Duration `mapstructure:"batch_timeout"`
	CacheShardCount     int           `mapstructure:"cache_shard_count"`
	EntityCacheDuration time.Duration `mapstructure:"entity_cache_duration"`
	EntityKey           string        `mapstructure:"entity_key"`
	EntityTypeKey       string        `mapstructure:"entity_type_key"`
	JobCacheDuration    time.Duration `mapstructure:"job_cache_duration"`
	JobKey              string        `mapstructure:"job_key"`
	MaxEntityWaitTime   time.Duration `mapstructure:"max_entity_wait_time"`
	QueueSize           int           `mapstructure:"queue_size"`
	SamplingFraction    int           `mapstructure:"sampling_fraction"`
	UseLinks            bool          `mapstructure:"use_links"`
	WorkerCount         int           `mapstructure:"worker_count"`
}

func (cfg *Config) Validate() error {

	if cfg.BaggageJobKey == "" {
		return fmt.Errorf("baggage_job_key must be set")
	}

	if cfg.BatchSize < 1 {
		return fmt.Errorf("batch_size must be greater than 0")
	}

	if cfg.BatchTimeout <= 0 {
		return fmt.Errorf("batch_timeout must be positive")
	}

	if cfg.CacheShardCount < 1 {
		return fmt.Errorf("cache_shard_count must be greater than 0")
	}

	if cfg.EntityCacheDuration <= 0 {
		return fmt.Errorf("entity_cache_duration must be positive")
	}

	if cfg.EntityKey == "" {
		return fmt.Errorf("entity_key must be set")
	}

	if cfg.EntityTypeKey == "" {
		return fmt.Errorf("entity_type_key must be set")
	}

	if cfg.JobCacheDuration <= 0 {
		return fmt.Errorf("job_cache_duration must be positive")
	}

	if cfg.JobKey == "" {
		return fmt.Errorf("job_key must be set")
	}

	if cfg.MaxEntityWaitTime <= 0 {
		return fmt.Errorf("max_entity_wait_time must be positive")
	}

	if cfg.QueueSize < 1 {
		return fmt.Errorf("queue_size must be greater than 0")
	}

	if cfg.SamplingFraction < 1 {
		return fmt.Errorf("sampling_fraction must be greater than 0")
	}

	if cfg.WorkerCount < 1 {
		return fmt.Errorf("worker_count must be greater than 0")
	}

	return nil
}
