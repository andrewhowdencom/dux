package config

import (
	"fmt"
	"strings"
)

// AgentInfo holds display-ready agent data.
type AgentInfo struct {
	Name     string
	Provider string
	Modes    string
	Triggers string
}

// ModeInfo holds display-ready mode data.
type ModeInfo struct {
	AgentName   string
	Name        string
	Provider    string
	Transitions string
}

// ListAgents returns all configured agents with display metadata.
func ListAgents(agentsDir string) ([]AgentInfo, error) {
	agents, err := LoadAgents(agentsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load agents: %w", err)
	}

	var infos []AgentInfo
	for _, a := range agents {
		infos = append(infos, AgentInfo{
			Name:     a.Name,
			Provider: a.Provider,
			Modes:    formatModes(a.Workflow),
			Triggers: formatTriggers(a.Triggers),
		})
	}
	return infos, nil
}

// ListModes returns all workflow modes for all agents with display metadata.
func ListModes(agentsDir string) ([]ModeInfo, error) {
	agents, err := LoadAgents(agentsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load agents: %w", err)
	}

	var infos []ModeInfo
	for _, a := range agents {
		if a.Workflow == nil || len(a.Workflow.Modes) == 0 {
			continue
		}
		for _, m := range a.Workflow.Modes {
			provider := m.Provider
			if provider == "" {
				provider = a.Provider
			}
			infos = append(infos, ModeInfo{
				AgentName:   a.Name,
				Name:        m.Name,
				Provider:    provider,
				Transitions: formatTransitions(m.Transitions),
			})
		}
	}
	return infos, nil
}

func formatModes(w *Workflow) string {
	if w == nil || len(w.Modes) == 0 {
		return "none"
	}
	names := make([]string, len(w.Modes))
	for i, m := range w.Modes {
		names[i] = m.Name
	}
	return fmt.Sprintf("%d (%s)", len(names), strings.Join(names, ", "))
}

func formatTriggers(triggers []Trigger) string {
	if len(triggers) == 0 {
		return "none"
	}
	types := make([]string, len(triggers))
	for i, t := range triggers {
		types[i] = t.Type
	}
	return strings.Join(types, ", ")
}

func formatTransitions(transitions []ModeTransition) string {
	if len(transitions) == 0 {
		return "none"
	}
	targets := make([]string, len(transitions))
	for i, t := range transitions {
		targets[i] = t.To
	}
	return strings.Join(targets, ", ")
}
