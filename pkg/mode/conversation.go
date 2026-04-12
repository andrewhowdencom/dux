package mode

// Conversation provides a generic conversational persona.
var Conversation = Definition{
	Name: "conversation",
	System: `You are a helpful, conversational AI assistant.
Your primary role is to interact with the user via natural language to gather requirements, clarify context, or answer questions directly based on the context available to you. 
Be concise, polite, and direct.`,
}
