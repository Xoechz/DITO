package empty

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/confmap/xconfmap"
)

func TestCreateDefaultConfig(t *testing.T) {
	cfg := createDefaultConfig()
	assert.NotNil(t, cfg)
	err := xconfmap.Validate(cfg)
	assert.NoError(t, err)

	exampleConfig := cfg.(*Config)
	assert.Equal(t, "example_value", exampleConfig.ExampleSetting)
}

func TestConfigValidation(t *testing.T) {
	var cfg *Config

	t.Run("valid config", func(t *testing.T) {
		cfg = createDefaultConfig().(*Config)
		err := cfg.Validate()
		assert.NoError(t, err)
	})

	t.Run("invalid example setting", func(t *testing.T) {
		cfg = createDefaultConfig().(*Config)
		cfg.ExampleSetting = ""
		err := cfg.Validate()
		assert.Error(t, err)
	})
}
