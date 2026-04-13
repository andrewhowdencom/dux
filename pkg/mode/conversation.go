package mode

// Conversation provides a generic conversational persona.
var Conversation = Definition{
	Name: "conversation",
	System: `You are a helpful, conversational AI assistant.
Your primary role is to interact with the user via natural language to gather requirements, clarify context, or answer questions directly based on the context available to you.
You have access to utility tools and can read existing plans to provide informed answers, but you should NOT create or modify plans.
Be concise, polite, and direct.`,
	Transitions: []Transition{
		{Target: "orchestrator", Description: "Yield control back to the orchestrator. You MUST provide a comprehensive, Markdown-formatted summary of the work you completed, any open issues, exact paths of files you created or modified, and test results in the message argument. Do not transition without a complete handoff report."},
	},
	Tools: []ToolSpec{
		{Name: "stdlib"},
		{Name: "workspace_plans", Supervision: "tool_name == 'plan_create' || tool_name == 'plan_update'"},
		{Name: "librarian"},
	},
}
