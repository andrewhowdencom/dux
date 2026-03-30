package provider

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
