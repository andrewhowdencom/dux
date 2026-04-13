package mode

// TransitionType defines the behavioral semantics of the transition.
type TransitionType string

const (
	// TransitionTypeHandover represents a strict context-switch. The calling agent terminates its loop
	// and transfers the active conversation directly over to the Target agent.
	TransitionTypeHandover TransitionType = "handover"

	// TransitionTypeDelegation represents a synchronous tool call to a sub-agent. The calling agent
	// retains the user-facing chat context while the sub-agent spins up, completes its task, and returns.
	TransitionTypeDelegation TransitionType = "delegation"

	// TransitionTypeReturn represents the completion of a delegation. The sub-agent terminates
	// and returns its payload back to the calling parent. The 'Target' field is ignored.
	TransitionTypeReturn TransitionType = "return"
)

// Transition represents a recommended state boundary transition that an LLM can invoke.
type Transition struct {
	Target      string         `yaml:"target,omitempty"`
	Description string         `yaml:"description"`
	Type        TransitionType `yaml:"type"`
}

// ToolSpec declares a tool that a mode requires, along with optional supervision policy.
type ToolSpec struct {
	Name        string
	Supervision any // nil = default, bool, or CEL expression string
}

// Definition outlines the core structural constraints and personas for an agentic state.
type Definition struct {
	Name        string
	System      string
	Tools       []ToolSpec
	Transitions []Transition
}
