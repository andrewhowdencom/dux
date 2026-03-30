package config

import (
	"fmt"

	"github.com/andrewhowdencom/dux/pkg/llm/provider"
	"github.com/spf13/viper"
)

// LoadLLMProvider reads the LLM deployment mappings from the current Viper context.
// It applies a generous suite of fallback configurations if the tree is completely missing,
// and resolves a single instance representation based on target request strings.
func LoadLLMProvider(targetID string) (provider.InstanceConfig, error) {
	var cfg provider.Config
	if err := viper.UnmarshalKey("llm", &cfg); err != nil {
		return provider.InstanceConfig{}, fmt.Errorf("failed to parse llm config: %w", err)
	}

	// Fallback for seamless local testing if user hasn't written their own config yet
	if len(cfg.Providers) == 0 {
		cfg.Providers = []provider.InstanceConfig{
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

	if targetID == "" {
		targetID = cfg.DefaultProvider
	}
	if targetID == "" && len(cfg.Providers) > 0 {
		targetID = cfg.Providers[0].ID
	}

	for _, p := range cfg.Providers {
		if p.ID == targetID {
			return p, nil
		}
	}

	return provider.InstanceConfig{}, fmt.Errorf("provider %q not found in configuration", targetID)
}
