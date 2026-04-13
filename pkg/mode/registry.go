package mode

// Builtins provides a static registry mapping string IDs to their robust mode Definitions.
var Builtins = map[string]Definition{
	"aide":         Aide,         // New primary interface
	"orchestrator": Aide,         // Alias for backwards compatibility
	"planning":     Planning,
	"execution":    Execution,
	"review":       Review,
}
