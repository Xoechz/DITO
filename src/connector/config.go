package dito

import (
	"fmt"
)

type Config struct {
	EntityKey                string `mapstructure:"entity_key"`
	JobKey                   string `mapstructure:"job_key"`
	MaxCachedEntities        int    `mapstructure:"max_cached_entities"`
	NonErrorSamplingFraction int    `mapstructure:"non_error_sampling_fraction"`
}

func (cfg *Config) Validate() error {
	if cfg.EntityKey == "" {
		return fmt.Errorf("entity_key must be set")
	}

	if cfg.JobKey == "" {
		return fmt.Errorf("job_key must be set")
	}

	if cfg.MaxCachedEntities <= 0 {
		return fmt.Errorf("max_cached_entities must be positive")
	}

	if cfg.NonErrorSamplingFraction < 1 {
		return fmt.Errorf("non_error_sampling_fraction must be greater than 0")
	}

	return nil
}
