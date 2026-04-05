package config

import (
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
)

type UIConfig struct {
	Type          string                 `mapstructure:"type"`
	Agent         string                 `mapstructure:"agent"`
	Provider      string                 `mapstructure:"provider"`
	Configuration map[string]interface{} `mapstructure:"configuration"`
}

type TelegramUIConfig struct {
	Token          string  `mapstructure:"token"`
	WebhookURL     string  `mapstructure:"webhook_url"`
	WebhookAddress string  `mapstructure:"webhook_address"`
	AllowedUsers   []int64 `mapstructure:"allowed_users"`
}

type WebUIConfig struct {
	Address string `mapstructure:"address"`
}

func (c *UIConfig) ParseTelegramConfig() (*TelegramUIConfig, error) {
	var cfg TelegramUIConfig
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result:  &cfg,
		TagName: "mapstructure",
	})
	if err != nil {
		return nil, err
	}
	if err := decoder.Decode(c.Configuration); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (c *UIConfig) ParseWebConfig() (*WebUIConfig, error) {
	var cfg WebUIConfig
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result:  &cfg,
		TagName: "mapstructure",
	})
	if err != nil {
		return nil, err
	}
	if err := decoder.Decode(c.Configuration); err != nil {
		return nil, err
	}
	// Fallback to :8080 if not set
	if cfg.Address == "" {
		cfg.Address = ":8080"
	}
	return &cfg, nil
}

func LoadUIs() ([]UIConfig, error) {
	var uis []UIConfig
	if err := viper.UnmarshalKey("ui", &uis); err != nil {
		return nil, err
	}
	return uis, nil
}
