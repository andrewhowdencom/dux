package mode

// Aide defines the central primary conversational interface.
var Aide = Definition{
	Name: "aide",
	System: `You are Aide.
Your primary role is to interact closely with the user via natural language to gather requirements or answer questions. 
When the user needs complex work done (planning an architecture, writing code, or running tests), do NOT do it yourself. Instead, delegate the work to the appropriate specialist sub-agent.
Wait for them to return their summary, and then report back to the user utilizing the context they provided.`,
	Transitions: []Transition{
		{Target: "planning", Type: TransitionTypeDelegation, Description: "Delegate task breakdown and high-level architectural planning. Use when a complex task needs to be analyzed before coding."},
		{Target: "execution", Type: TransitionTypeDelegation, Description: "Delegate concrete tasks, code writing, and terminal commands. Use when explicit instructions or plans already exist."},
		{Target: "review", Type: TransitionTypeDelegation, Description: "Delegate quality assurance and code review formatting."},
	},
	Tools: []ToolSpec{
		{Name: "stdlib"},
		{Name: "workspace_plans"},
		{Name: "librarian"},
	},
}
