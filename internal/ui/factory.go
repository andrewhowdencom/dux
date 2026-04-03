package ui

import (
	"fmt"

	"github.com/andrewhowdencom/dux/internal/config"
	"github.com/andrewhowdencom/dux/pkg/llm"
	"github.com/andrewhowdencom/dux/pkg/llm/enrich"
	"github.com/andrewhowdencom/dux/pkg/llm/provider"
	"github.com/andrewhowdencom/dux/pkg/llm/provider/ollama"
	"github.com/andrewhowdencom/dux/pkg/llm/provider/openai"
	"github.com/andrewhowdencom/dux/pkg/llm/provider/static"
	bashtool "github.com/andrewhowdencom/dux/pkg/llm/tool/bash"
	filetool "github.com/andrewhowdencom/dux/pkg/llm/tool/file"
	static_resolver "github.com/andrewhowdencom/dux/pkg/llm/tool/static"
	timetool "github.com/andrewhowdencom/dux/pkg/llm/tool/time"
	"github.com/mitchellh/mapstructure"
)

// NewProviderFromConfig maps a generic config definition to a concrete LLM Provider constructor.
func NewProviderFromConfig(cfg config.InstanceConfig) (provider.Provider, error) {
	switch cfg.Type {
	case "static":
		var staticOpts []static.Option
		if text, ok := cfg.Config["text"].(string); ok {
			staticOpts = append(staticOpts, static.WithText(text))
		}
		return static.New(staticOpts...)

	case "ollama":
		var ollamaCfg struct {
			Address string `mapstructure:"address"`
			Model   string `mapstructure:"model"`
		}
		if err := mapstructure.Decode(cfg.Config, &ollamaCfg); err != nil {
			return nil, fmt.Errorf("failed to decode ollama config: %w", err)
		}
		var opts []ollama.Option
		if ollamaCfg.Address != "" {
			opts = append(opts, ollama.WithAddress(ollamaCfg.Address))
		}
		if ollamaCfg.Model != "" {
			opts = append(opts, ollama.WithModel(ollamaCfg.Model))
		}
		return ollama.New(opts...)

	case "openai", "litellm":
		var openAICfg struct {
			BaseURL string `mapstructure:"base_url"`
			APIKey  string `mapstructure:"api_key"`
			Model   string `mapstructure:"model"`
		}
		if err := mapstructure.Decode(cfg.Config, &openAICfg); err != nil {
			return nil, fmt.Errorf("failed to decode openai config: %w", err)
		}
		var opts []openai.Option
		if openAICfg.BaseURL != "" {
			opts = append(opts, openai.WithBaseURL(openAICfg.BaseURL))
		}
		if openAICfg.Model != "" {
			opts = append(opts, openai.WithModel(openAICfg.Model))
		}
		return openai.New(openAICfg.APIKey, opts...)

	default:
		return nil, fmt.Errorf("unknown or unsupported provider type: %q (id: %q)", cfg.Type, cfg.ID)
	}
}

// NewEnrichersFromConfig builds an array of enrichers from raw agent configuration.
func NewEnrichersFromConfig(cfgs []config.Enricher) ([]enrich.Enricher, error) {
	var results []enrich.Enricher

	for _, c := range cfgs {
		switch c.Type {
		case "time":
			results = append(results, enrich.NewTime())
		case "os":
			results = append(results, enrich.NewOS())
		case "prompt":
			results = append(results, enrich.NewPrompt(c.Text))
		case "guard_rail":
			results = append(results, enrich.NewGuardRail(c.Text))
		default:
			// Returning an error ensures configuration typos are caught.
			return nil, fmt.Errorf("unknown enricher type: %s", c.Type)
		}
	}

	return results, nil
}

// NewResolversFromConfig builds an array of tool resolvers from string representations.
func NewResolversFromConfig(cfgs []string) ([]llm.ToolResolver, error) {
	var results []llm.ToolResolver

	// For standard configuration strings, wrap the tool inside a static resolver
	var staticTools []llm.Tool

	for _, c := range cfgs {
		switch c {
		case "time":
			staticTools = append(staticTools, timetool.New())
		case "bash":
			staticTools = append(staticTools, bashtool.New())
		case "file_read":
			staticTools = append(staticTools, filetool.NewRead())
		case "file_write":
			staticTools = append(staticTools, filetool.NewWrite())
		case "file_patch":
			staticTools = append(staticTools, filetool.NewPatch())
		case "file_list":
			staticTools = append(staticTools, filetool.NewList())
		default:
			return nil, fmt.Errorf("unknown tool name: %s", c)
		}
	}

	if len(staticTools) > 0 {
		results = append(results, static_resolver.New(staticTools...))
	}

	return results, nil
}
