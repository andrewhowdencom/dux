package config

import (
	"testing"

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
