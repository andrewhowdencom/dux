package config

import (
	"fmt"
	"os"

	"github.com/spf13/viper"

	"gopkg.in/yaml.v3"
)

// Enricher defines the configuration for a context enricher.
type Enricher struct {
	Type string `yaml:"type"`
	Text string `yaml:"text,omitempty"`
}

// ToolRequirements details specific execution protections for a given tool.
type ToolRequirements struct {
	Supervision *bool `yaml:"supervision,omitempty"`
	Sandbox     *bool `yaml:"sandbox,omitempty"`
}

// ToolConfig maps a specific tool and its deployment requirements.
// A tool can be either a locally built-in tool or an external MCP server.
type ToolConfig struct {
	Name         string           `yaml:"name"`
	Enabled      bool             `yaml:"enabled"`
	Requirements ToolRequirements `yaml:"requirements,omitempty"`
	MCP          *MCPServer       `yaml:"mcp,omitempty"`
}

// AgentContext defines dynamic and static context to configure an agent's memory before interaction.
type AgentContext struct {
	Enrichers []Enricher `yaml:"enrichers,omitempty"`
	Tools     []ToolConfig `yaml:"tools,omitempty"`
	System    string       `yaml:"system,omitempty"`
}

// MCPServer defines configuration for an external Model Context Protocol server.
// It supports stdio (Command/Args), streamable_http (URL), or explicitly sse (URL) transports.
type MCPServer struct {
	Transport string            `yaml:"transport,omitempty"`
	Command   string            `yaml:"command,omitempty"`
	Args      []string          `yaml:"args,omitempty"`
	Env       map[string]string `yaml:"env,omitempty"`
	URL       string            `yaml:"url,omitempty"`
	Headers   map[string]string `yaml:"headers,omitempty"`
}

// Agent defines a distinct interactive role combining a provider and dynamic context.
type Agent struct {
	Name     string        `yaml:"name"`
	Provider string        `yaml:"provider"`
	Context  *AgentContext `yaml:"context,omitempty"`
}

// LoadAgents maps an agent specification list at a given filepath to an array of Agent objects.
func LoadAgents(filePath string) ([]Agent, error) {
	if filePath == "" {
		return nil, nil // No file requested
	}

	b, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // Return empty if the default file doesn't exist
		}
		return nil, fmt.Errorf("failed to read agents file %q: %w", filePath, err)
	}

	var agents []Agent
	if err := yaml.Unmarshal(b, &agents); err != nil {
		return nil, fmt.Errorf("failed to unmarshal agents spec file %q: %w", filePath, err)
	}

	return agents, nil
}

// GetAgent extracts a specific agent by name from an array of agents.
func GetAgent(agents []Agent, name string) (Agent, error) {
	for _, a := range agents {
		if a.Name == name {
			return a, nil
		}
	}
	return Agent{}, fmt.Errorf("agent %q not found", name)
}

// LoadGlobalTools retrieves the application-wide defined tools, securely injecting missing mandatory defaults (e.g. 'time').
func LoadGlobalTools() []ToolConfig {
	var globalTools []ToolConfig
	_ = viper.UnmarshalKey("tools", &globalTools)

	hasTime := false
	for _, t := range globalTools {
		if t.Name == "time" {
			hasTime = true
			break
		}
	}
	if !hasTime {
		f := false
		globalTools = append(globalTools, ToolConfig{
			Name:    "time",
			Enabled: true,
			Requirements: ToolRequirements{
				Supervision: &f,
			},
		})
	}

	return globalTools
}
