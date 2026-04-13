package ui

import (
	"context"
	"fmt"

	"github.com/andrewhowdencom/dux/internal/config"
	"github.com/andrewhowdencom/dux/pkg/llm"
	"github.com/andrewhowdencom/dux/pkg/llm/enrich"
	"github.com/andrewhowdencom/dux/pkg/llm/provider"
	"github.com/andrewhowdencom/dux/pkg/llm/provider/gemini"
	"github.com/andrewhowdencom/dux/pkg/llm/provider/ollama"
	"github.com/andrewhowdencom/dux/pkg/llm/provider/openai"
	"github.com/andrewhowdencom/dux/pkg/llm/provider/static"
	"github.com/andrewhowdencom/dux/pkg/llm/tool"
	bashtool "github.com/andrewhowdencom/dux/pkg/llm/tool/bash"
	filetool "github.com/andrewhowdencom/dux/pkg/llm/tool/file"
	static_resolver "github.com/andrewhowdencom/dux/pkg/llm/tool/static"
	stdlibtool "github.com/andrewhowdencom/dux/pkg/llm/tool/stdlib"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
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
			NumCtx  int    `mapstructure:"num_ctx"`
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
		if ollamaCfg.NumCtx > 0 {
			opts = append(opts, ollama.WithNumCtx(ollamaCfg.NumCtx))
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

	case "gemini":
		var geminiCfg struct {
			APIKey string `mapstructure:"api_key"`
			Model  string `mapstructure:"model"`
		}
		if err := mapstructure.Decode(cfg.Config, &geminiCfg); err != nil {
			return nil, fmt.Errorf("failed to decode gemini config: %w", err)
		}
		var opts []gemini.Option
		if geminiCfg.Model != "" {
			opts = append(opts, gemini.WithModel(geminiCfg.Model))
		}
		return gemini.New(geminiCfg.APIKey, opts...)

	default:
		return nil, fmt.Errorf("unknown or unsupported provider type: %q (id: %q)", cfg.Type, cfg.ID)
	}
}

// NewEnrichersFromConfig builds an array of enrichers from raw agent configuration.
func NewEnrichersFromConfig(cfgs []config.Enricher) ([]llm.Injector, error) {
	var results []llm.Injector

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
func NewResolversFromConfig(cfgs []string) ([]llm.ToolProvider, error) {
	var results []llm.ToolProvider

	for _, c := range cfgs {
		switch c {
		case "stdlib":
			results = append(results, static_resolver.New(
				"stdlib",
				stdlibtool.New(), stdlibtool.NewDate(), stdlibtool.NewTimer(), stdlibtool.NewStopwatch(),
				stdlibtool.NewMath(), stdlibtool.NewUUID(), stdlibtool.NewRandom(),
				stdlibtool.NewBase64Encode(), stdlibtool.NewBase64Decode(),
				stdlibtool.NewURLEncode(), stdlibtool.NewURLDecode(), stdlibtool.NewSleep(),
			))
		case "filesystem":
			results = append(results, static_resolver.New("filesystem", filetool.NewRead(), filetool.NewWrite(), filetool.NewPatch(), filetool.NewList()))
		case "file_read":
			results = append(results, static_resolver.New("file_read", filetool.NewRead()))
		case "file_list":
			results = append(results, static_resolver.New("file_list", filetool.NewList()))
		case "bash":
			results = append(results, static_resolver.New("bash", bashtool.New()))
		default:
			return nil, fmt.Errorf("unknown tool bundle/name: %s", c)
		}
	}

	return results, nil
}

// NewMCPResolversFromConfig builds an array of tool resolvers from an array of MCP tool configurations.
// It connects to the MCP servers, initializes them, and binds resolving wrappers to them.
func NewMCPResolversFromConfig(ctx context.Context, agentName string, mcpConfigs []config.ToolConfig) ([]llm.ToolProvider, func(), error) {
	var mcpClients []*client.Client
	var resolvers []llm.ToolProvider

	cleanup := func() {
		for _, c := range mcpClients {
			_ = c.Close()
		}
	}

	for _, t := range mcpConfigs {
		var mcpClient *client.Client
		var clientErr error
		s := t.MCP

		transportType := s.Transport
		if transportType == "" {
			if s.Command != "" {
				transportType = "stdio"
			} else if s.URL != "" {
				transportType = "streamable_http"
			}
		}

		switch transportType {
		case "stdio":
			env := make([]string, 0, len(s.Env))
			for k, v := range s.Env {
				env = append(env, fmt.Sprintf("%s=%s", k, v))
			}
			mcpClient, clientErr = client.NewStdioMCPClient(s.Command, env, s.Args...)
		case "sse":
			var opts []transport.ClientOption
			if s.Headers != nil {
				opts = append(opts, transport.WithHeaders(s.Headers))
			}
			mcpClient, clientErr = client.NewSSEMCPClient(s.URL, opts...)
			if clientErr == nil {
				clientErr = mcpClient.Start(ctx)
			}
		case "streamable_http":
			var opts []transport.StreamableHTTPCOption
			if s.Headers != nil {
				opts = append(opts, transport.WithHTTPHeaders(s.Headers))
			}
			var tport *transport.StreamableHTTP
			tport, clientErr = transport.NewStreamableHTTP(s.URL, opts...)
			if clientErr == nil {
				mcpClient = client.NewClient(tport)
				clientErr = mcpClient.Start(ctx)
			}
		default:
			clientErr = fmt.Errorf("unsupported or missing MCP transport type: %q", transportType)
		}

		if clientErr != nil {
			cleanup()
			return nil, nil, fmt.Errorf("failed to create MCP client for %q: %w", t.Name, clientErr)
		}

		// Initialize
		initReq := mcp.InitializeRequest{}
		initReq.Params.ProtocolVersion = "2024-11-05"

		mcpName := agentName
		if mcpName == "" {
			mcpName = "dux"
		}
		initReq.Params.ClientInfo = mcp.Implementation{Name: mcpName, Version: "1.0.0"}
		if _, err := mcpClient.Initialize(ctx, initReq); err != nil {
			cleanup()
			return nil, nil, fmt.Errorf("failed to initialize MCP client %q: %w", t.Name, err)
		}
		mcpClients = append(mcpClients, mcpClient)

		r, err := tool.NewMCPResolver(ctx, t.Name, mcpClient)
		if err != nil {
			cleanup()
			return nil, nil, fmt.Errorf("failed to bind MCP resolver %q: %w", t.Name, err)
		}
		resolvers = append(resolvers, r)
	}

	return resolvers, cleanup, nil
}
