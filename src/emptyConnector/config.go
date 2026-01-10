package empty

import (
	"fmt"
)

type Config struct {
	ExampleSetting string `mapstructure:"example_setting"`
}

func (cfg *Config) Validate() error {
	if cfg.ExampleSetting == "" {
		return fmt.Errorf("example_setting cannot be empty")
	}

	return nil
}
