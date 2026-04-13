package config

import (
	"testing"
	"github.com/andrewhowdencom/dux/pkg/mode"
	"gopkg.in/yaml.v3"
)

func TestParseAgentWithWorkflow(t *testing.T) {
	yamlContent := `
name: code-reviewer
provider: anthropic/claude-3-haiku
workflow:
  default_mode: "qa"
  modes:
    - name: "qa"
      context:
        system: "You are a friendly QA bot handling questions."
        tools:
          - name: "read_file"
          - name: "stdlib"
      transitions:
        - to: "review"
          description: "Use this tool to switch to the review mode"
        
    - name: "review"
      provider: anthropic/claude-3-opus
      context:
        system: "You are a harsh code critic."
        tools:
          - name: "run_linter"
      transitions:
        - to: "qa"
          description: "Use this tool to return your bug report"
`

	var agent Agent
	err := yaml.Unmarshal([]byte(yamlContent), &agent)
	if err != nil {
		t.Fatalf("Failed to unmarshal YAML: %v", err)
	}

	if agent.Name != "code-reviewer" {
		t.Errorf("Expected agent name 'code-reviewer', got '%s'", agent.Name)
	}

	if agent.Provider != "anthropic/claude-3-haiku" {
		t.Errorf("Expected agent provider 'anthropic/claude-3-haiku', got '%s'", agent.Provider)
	}

	if agent.Workflow == nil {
		t.Fatalf("Expected Workflow to be populated, was nil")
	}

	if agent.Workflow.DefaultMode != "qa" {
		t.Errorf("Expected DefaultMode 'qa', got '%s'", agent.Workflow.DefaultMode)
	}

	if len(agent.Workflow.Modes) != 2 {
		t.Fatalf("Expected 2 modes, got %d", len(agent.Workflow.Modes))
	}

	// Verify Mode 1: QA
	qaMode := agent.Workflow.Modes[0]
	if qaMode.Name != "qa" {
		t.Errorf("Expected QA mode name 'qa', got '%s'", qaMode.Name)
	}
	if qaMode.Context == nil {
		t.Fatalf("QA context is nil")
	}
	if len(qaMode.Context.Tools) != 2 {
		t.Errorf("Expected 2 tools in QA mode, got %d", len(qaMode.Context.Tools))
	}
	if len(qaMode.Transitions) != 1 {
		t.Fatalf("Expected 1 transition in QA mode, got %d", len(qaMode.Transitions))
	}
	if qaMode.Transitions[0].To != "review" {
		t.Errorf("Expected transition to 'review', got '%s'", qaMode.Transitions[0].To)
	}

	// Verify Mode 2: Review
	reviewMode := agent.Workflow.Modes[1]
	if reviewMode.Name != "review" {
		t.Errorf("Expected Review mode name 'review', got '%s'", reviewMode.Name)
	}
	if reviewMode.Provider != "anthropic/claude-3-opus" {
		t.Errorf("Expected Review provider 'anthropic/claude-3-opus', got '%s'", reviewMode.Provider)
	}
}

func TestTransitiveModeInjection(t *testing.T) {
// If an agent defines orchestrator, which transitions to conversation,
// conversation should be auto-injected from Builtins.
yamlContent := `
name: auto-inject-test
provider: test
workflow:
  default_mode: "orchestrator"
  modes:
    - name: "orchestrator"
`
var agent Agent
err := yaml.Unmarshal([]byte(yamlContent), &agent)
if err != nil {
t.Fatalf("Failed to unmarshal YAML: %v", err)
}

// This duplicates the logic from LoadAgents to test it in isolation
if agent.Workflow != nil {
for i, m := range agent.Workflow.Modes {
useKey := m.Use
if useKey == "" {
useKey = m.Name
}

if def, ok := mode.Builtins[useKey]; ok {
base := mapDefinitionToMode(def)
m.Merge(base)
agent.Workflow.Modes[i] = m
}
}

for {
added := false
existingModes := make(map[string]bool)
for _, m := range agent.Workflow.Modes {
existingModes[m.Name] = true
}

var newModes []Mode
for _, m := range agent.Workflow.Modes {
for _, t := range m.Transitions {
if !existingModes[t.To] {
if def, ok := mode.Builtins[t.To]; ok {
base := mapDefinitionToMode(def)
newModes = append(newModes, *base)
existingModes[t.To] = true
added = true
}
}
}
}
if added {
agent.Workflow.Modes = append(agent.Workflow.Modes, newModes...)
} else {
break
}
}
}

foundConversation := false
for _, m := range agent.Workflow.Modes {
if m.Name == "conversation" {
foundConversation = true
break
}
}

if !foundConversation {
t.Errorf("Expected 'conversation' mode to be transitively injected")
}
}
