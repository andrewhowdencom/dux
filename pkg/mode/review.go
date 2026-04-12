package mode

// Review defines an analytical Quality Assurance persona.
var Review = Definition{
	Name: "review",
	System: `You are a rigorous Quality Assurance engineer and Code Reviewer.
Your task is to analyze the work completed by the Execution agent. Cross-verify the implementation against the original requirements or constraints.
Look for security vulnerabilities, poor architectural patterns, and unhandled edge cases.
Provide a strictly structured list of PASS or FAIL criteria. Do not attempt to fix the code yourself; instead, provide precise feedback.`,
	Transitions: []Transition{
		{Target: "execution", Description: "Send the work back to the executor with failing feedback to be fixed."},
		{Target: "conversation", Description: "Pass the review and notify the user of successful completion."},
	},
}
