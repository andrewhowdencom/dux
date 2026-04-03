package config

import (
	"fmt"

	"github.com/spf13/viper"
)

// Config represents the top-level configuration for the LLM subsystem.
type Config struct {
	DefaultProvider string           `mapstructure:"default_provider"`
	Providers       []InstanceConfig `mapstructure:"providers"`
}

// InstanceConfig holds the generic mapping for any provider instance.
type InstanceConfig struct {
	ID     string                 `mapstructure:"id"`
	Type   string                 `mapstructure:"type"`
	Config map[string]interface{} `mapstructure:"config"` // Polymorphic config
}

// LoadLLMProviders returns all configured LLM providers.
func LoadLLMProviders() ([]InstanceConfig, error) {
	var cfg Config
	if err := viper.UnmarshalKey("llm", &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse llm config: %w", err)
	}

	// Fallback for seamless local testing if user hasn't written their own config yet
	if len(cfg.Providers) == 0 {
		cfg.Providers = []InstanceConfig{
			{
				ID:   "static",
				Type: "static",
			},
			{
				ID:   "ollama",
				Type: "ollama",
				Config: map[string]interface{}{
					"address": "http://localhost:11434",
					"model":   "llama3",
				},
			},
		}
	}
	return cfg.Providers, nil
}

// LoadLLMProvider reads the LLM deployment mappings from the current Viper context.
// It applies a generous suite of fallback configurations if the tree is completely missing,
// and resolves a single instance representation based on target request strings.
func LoadLLMProvider(targetID string) (InstanceConfig, error) {
	providers, err := LoadLLMProviders()
	if err != nil {
		return InstanceConfig{}, err
	}

	if targetID == "" {
		var cfg Config
		_ = viper.UnmarshalKey("llm", &cfg)
		targetID = cfg.DefaultProvider
	}
	if targetID == "" && len(providers) > 0 {
		targetID = providers[0].ID
	}

	for _, p := range providers {
		if p.ID == targetID {
			return p, nil
		}
	}

	return InstanceConfig{}, fmt.Errorf("provider %q not found in configuration", targetID)
}
