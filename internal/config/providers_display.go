package config

import (
	"fmt"

	"github.com/spf13/viper"
)

// ProviderInfo holds display-ready provider data.
type ProviderInfo struct {
	ID        string
	Type      string
	Details   string
	IsDefault bool
}

// ListProviders returns all configured LLM providers with display metadata.
func ListProviders() ([]ProviderInfo, error) {
	providers, err := LoadLLMProviders()
	if err != nil {
		return nil, fmt.Errorf("failed to load providers: %w", err)
	}

	defaultID := GetDefaultProviderID()

	var infos []ProviderInfo
	for _, p := range providers {
		infos = append(infos, ProviderInfo{
			ID:        p.ID,
			Type:      p.Type,
			Details:   formatProviderDetails(p),
			IsDefault: p.ID == defaultID,
		})
	}
	return infos, nil
}

// GetDefaultProviderID returns the configured default provider ID.
func GetDefaultProviderID() string {
	var cfg Config
	_ = viper.UnmarshalKey("llm", &cfg)
	return cfg.DefaultProvider
}

func formatProviderDetails(p InstanceConfig) string {
	switch p.Type {
	case "ollama":
		model := getString(p.Config, "model", "unknown")
		address := getString(p.Config, "address", "localhost")
		return fmt.Sprintf("%s @ %s", model, address)
	case "openai", "litellm":
		model := getString(p.Config, "model", "unknown")
		baseURL := getString(p.Config, "base_url", "api.openai.com")
		return fmt.Sprintf("%s @ %s", model, baseURL)
	case "gemini":
		model := getString(p.Config, "model", "unknown")
		return model
	case "static":
		return "static-model"
	default:
		return "unknown"
	}
}

func getString(m map[string]interface{}, key, defaultVal string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return defaultVal
}
