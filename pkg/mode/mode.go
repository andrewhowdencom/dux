package mode

// Transition represents a recommended state boundary transition that an LLM can invoke.
type Transition struct {
	Target      string
	Description string
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
