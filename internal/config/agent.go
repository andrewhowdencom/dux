package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/adrg/xdg"
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
	Supervision any   `yaml:"supervision,omitempty"` // Boolean or CEL string
	Sandbox     *bool `yaml:"sandbox,omitempty"`
}

// ToolInput defines single input parameter expected for a binary tool.
type ToolInput struct {
	Type        string `yaml:"type"`
	Description string `yaml:"description"`
	Required    bool   `yaml:"required,omitempty"`
}

// BinaryTool defines execution requirements for a custom command line binary.
type BinaryTool struct {
	Description string               `yaml:"description"`
	Executable  string               `yaml:"executable"`
	Args        []string             `yaml:"args"`
	Inputs      map[string]ToolInput `yaml:"inputs,omitempty"`
}

// ToolConfig maps a specific tool and its deployment requirements.
// A tool can be either a locally built-in tool, an external MCP server, or a declarative binary tool.
// It can also act as a logical "toolset" grouping by nesting additional configurations inside 'Tools'.
type ToolConfig struct {
	Name           string           `yaml:"name"`
	Enabled        bool             `yaml:"enabled"`
	TimeoutSeconds *int             `yaml:"timeout_seconds,omitempty"`
	Requirements   ToolRequirements `yaml:"requirements,omitempty"`
	MCP            *MCPServer       `yaml:"mcp,omitempty"`
	Binary         *BinaryTool      `yaml:"binary,omitempty"`
	Tools          []ToolConfig     `yaml:"tools,omitempty"`
}

// AgentContext defines dynamic and static context to configure an agent's memory before interaction.
type AgentContext struct {
	Enrichers []Enricher   `yaml:"enrichers,omitempty"`
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

// ModeTransition defines an automated tool injection to transition between workflow modes.
type ModeTransition struct {
	To          string `yaml:"to"`
	Description string `yaml:"description"`
}

// Mode represents a specific state in an agent's workflow graph.
type Mode struct {
	Name        string           `yaml:"name"`
	Provider    string           `yaml:"provider,omitempty"`
	Context     *AgentContext    `yaml:"context,omitempty"`
	Transitions []ModeTransition `yaml:"transitions,omitempty"`
}

// Workflow defines the graph of modes a context router traverses.
type Workflow struct {
	DefaultMode string `yaml:"default_mode"`
	Modes       []Mode `yaml:"modes"`
}

// Agent defines a distinct interactive role combining a provider and dynamic context.
type Agent struct {
	Name     string        `yaml:"name"`
	Provider string        `yaml:"provider"`
	Context  *AgentContext `yaml:"context,omitempty"`
	Workflow *Workflow     `yaml:"workflow,omitempty"`
	Triggers []Trigger     `yaml:"triggers,omitempty"`
}

// Trigger represents a configuration for defining how an agent starts.
type Trigger struct {
	Type   string            `yaml:"type"`
	Config map[string]string `yaml:"config,omitempty"`
}

// ResolveAgentsDir determines the base directory for agents configuration.
// If path is empty, it returns $XDG_CONFIG_HOME/dux/agents strictly.
func ResolveAgentsDir(path string) string {
	if path != "" {
		return path
	}
	p, _ := xdg.ConfigFile("dux/agents")
	return p
}

// LoadAgents enumerates all folders within agentsDir, expecting an agent.yaml inside.
func LoadAgents(agentsDir string) ([]Agent, error) {
	dir := ResolveAgentsDir(agentsDir)

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // Return empty if the agents directory doesn't exist
		}
		return nil, fmt.Errorf("failed to read agents directory %q: %w", dir, err)
	}

	var agents []Agent
	for _, entry := range entries {
		if !entry.IsDir() {
			continue // Only process subdirectories
		}
		
		agentFilePath := filepath.Join(dir, entry.Name(), "agent.yaml")
		b, err := os.ReadFile(agentFilePath)
		if err != nil {
			if os.IsNotExist(err) {
				continue // Skip if there's no agent.yaml in this folder
			}
			return nil, fmt.Errorf("failed to read agent file %q: %w", agentFilePath, err)
		}

		var agent Agent
		if err := yaml.Unmarshal(b, &agent); err != nil {
			return nil, fmt.Errorf("failed to unmarshal agent spec file %q: %w", agentFilePath, err)
		}
		
		// Optional: If the internal Name is missing, we could default to entry.Name() here
		// if agent.Name == "" { agent.Name = entry.Name() }

		agents = append(agents, agent)
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

	configured := make(map[string]bool)
	for _, t := range globalTools {
		configured[t.Name] = true
	}

	if !configured["stdlib"] {
		globalTools = append(globalTools, ToolConfig{
			Name:    "stdlib",
			Enabled: true,
			Requirements: ToolRequirements{
				Supervision: false,
			},
		})
	}

	return globalTools
}
