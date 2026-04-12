package mode

// Transition represents a recommended state boundary transition that an LLM can invoke.
type Transition struct {
	Target      string
	Description string
}

// Definition outlines the core structural constraints and personas for an agentic state.
type Definition struct {
	Name        string
	System      string
	Transitions []Transition
}
