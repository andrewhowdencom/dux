package mode

// Builtins provides a static registry mapping string IDs to their robust mode Definitions.
var Builtins = map[string]Definition{
	"conversation": Conversation,
	"planning":     Planning,
	"execution":    Execution,
	"review":       Review,
	"orchestrator": Orchestrator,
}
