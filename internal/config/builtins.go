package config

// BuiltinModes acts as the central registry for standard mode definitions.
// Users can inherit from these by specifying `use: "mode_name"` in their agent.yaml.
var BuiltinModes = map[string]Mode{
	"conversation": {
		Name: "conversation",
		Context: &AgentContext{
			System: "You are a helpful, conversational AI assistant. Engage the user directly and concisely.",
		},
	},
	"planning": {
		Name: "planning",
		Context: &AgentContext{
			System: "You are an expert technical planner. Before executing commands, you must produce a step-by-step checklist.",
		},
	},
	"execution": {
		Name: "execution",
		Context: &AgentContext{
			System: "You are a precise, autonomous executor. Write code, solve problems, and verify results without engaging in chatty conversation.",
		},
	},
}

// GetBuiltinMode safely returns a deep-copy of a builtin mode if it exists.
// Deep copying prevents parallel sessions from accidentally mutating the global registry 
// if they override properties via implicit inheritance.
func GetBuiltinMode(name string) (*Mode, bool) {
	base, exists := BuiltinModes[name]
	if !exists {
		return nil, false
	}
	
	cpy := base
	if base.Context != nil {
		ctxCpy := *base.Context
		cpy.Context = &ctxCpy
		// Deep copy slices
		if len(base.Context.Enrichers) > 0 {
			cpy.Context.Enrichers = append([]Enricher{}, base.Context.Enrichers...)
		}
		if len(base.Context.Tools) > 0 {
			cpy.Context.Tools = append([]ToolConfig{}, base.Context.Tools...)
		}
	}
	if len(base.Transitions) > 0 {
		cpy.Transitions = append([]ModeTransition{}, base.Transitions...)
	}
	return &cpy, true
}
