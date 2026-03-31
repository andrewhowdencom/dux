package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Agent defines a distinct interactive role combining a provider and a prompt.
type Agent struct {
	Name         string `yaml:"name"`
	Provider     string `yaml:"provider"`
	SystemPrompt string `yaml:"system_prompt"`
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
