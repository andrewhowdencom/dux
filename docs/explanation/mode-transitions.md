# Mode Transitions & Workflow Orchestration

Most agent frameworks treat an agent as a single system prompt with a fixed set of tools. This works for simple Q&A, but collapses when a task requires qualitatively different kinds of reasoning at different stages. A coding task, for example, needs an **architect** to plan, an **executor** to write tests and code, and a **reviewer** to audit the result. Each role requires a different system prompt, a different risk posture toward tools, and often a different LLM provider. Mode transitions are Dux's answer to this problem.

## The Problem with Monolithic Agents

If you give a single agent both planning and execution privileges, several failure modes emerge:

- **Role confusion.** The LLM may skip planning and jump straight to coding, or over-plan and never execute.
- **Prompt dilution.** A single system prompt that tries to cover planning, coding, and reviewing becomes so long that it competes with the conversation history for context-window space.
- **Unsafe defaults.** A planning agent should not have `bash` access; an execution agent should. If the same prompt governs both, you must either over-privilege planning or under-privilege execution.

Separating these into independent agents run by an external orchestrator solves some of these problems, but introduces a new one: **session fragmentation**. Each agent starts with a blank slate, losing the conversational context, file modifications, and memory accumulated by the previous agent.

Dux's mode system keeps the session intact while swapping the *persona*.

## Modes as Personas

A **mode** is a complete execution configuration: a system prompt, a provider, a tool set, and a list of legal transitions. Dux ships with four built-in modes that form a canonical software-development workflow, but the system is open-ended and user-defined modes are supported via YAML or pure Go.

| Mode | Role | Typical Tools | Transitions |
|------|------|---------------|-------------|
| `aide` | Conversational interface, requirement gatherer | `stdlib`, `librarian`, `workspace_plans` | Delegates to `planning`, `execution`, or `review` |
| `planning` | Architect and task decomposer | `workspace_plans`, `librarian`, read-only `filesystem` | Returns to caller with a plan document |
| `execution` | Code writer and test runner | Full `filesystem`, `bash`, `workspace_plans` | Returns to caller with results |
| `review` | Quality assurance auditor | `librarian`, read-only `filesystem`, read-only `bash` | Returns with PASS/FAIL verdict |

Each mode is a self-contained agent definition. The `aide` mode does not do the work itself; it delegates to specialists, waits for their return, and synthesizes the result back to the user.

## Three Transition Semantics

Not all hand-offs are the same. Dux distinguishes three transition types because the lifecycle of a sub-task differs depending on whether the caller needs to survive, wait, or terminate.

### `handover` — Context Switch

The caller **terminates** and the target mode **takes over** the user-facing session. This is useful when the conversation itself has shifted purpose. For example, a general-purpose chatbot (`aide`) might hand over to a dedicated customer-support mode when the user asks for a refund. The original mode is gone; the new mode owns the stream.

```text
User: "I need a refund."
Aide → [handover] → SupportMode
SupportMode: "I can help with that. What is your order ID?"
```

### `delegation` — Synchronous Sub-Agent Call

The caller **retains** the user-facing context but **spawns** a sub-agent to complete a specific task. The caller blocks until the sub-agent returns. This is the most common pattern in the built-in workflow: `aide` delegates planning to `planning`, then delegates execution to `execution`, then delegates review to `review`.

```text
User: "Add a health-check endpoint."
Aide → [delegation] → Planning
Planning: (creates plan, returns)
Aide → [delegation] → Execution
Execution: (implements code, returns)
Aide → [delegation] → Review
Review: (audits code, returns)
Aide: "Done. I added /health, a test, and a README note."
```

The user never speaks directly to `planning`, `execution`, or `review`. The `aide` mediates everything.

### `return` — Completion Signal

A sub-agent that was invoked via `delegation` **finishes** and yields its payload back to the caller. The `return` transition does not name a target; the target is implicitly the caller. This is how `execution` and `review` conclude their work and hand control back to `aide`.

## Engine Hot-Swapping Without Session Loss

The mechanism that makes this possible is **engine hot-swapping**. When the `WorkflowEngine` (a wrapper around the core adapter) detects a `TransitionSignalPart` in the stream, it does not mutate the existing engine in place. Instead, it:

1. **Instantiates a brand-new inner engine** with the target mode's configuration (provider, system prompt, tool resolvers).
2. **Preserves the shared working memory** (history) so the new engine sees the full conversation, including the tool results from prior turns.
3. **Injects a bridge message** — a hidden system message that tells the new mode why it was activated and what context it inherits.
4. **Restarts the convergence loop** with the new engine, using the bridge message as the initial prompt.

This hermetic swap is critical. If the engine were mutated in place, residual tool resolvers, hooks, or provider state from the previous mode could leak into the new one. By constructing a fresh engine, Dux guarantees isolation.

## Bridge Messages and History Continuity

A subtle but important detail: LLM providers expect a strict alternation of roles (`user`, `assistant`, `tool`). If a turn ends with an assistant message that requested a tool, but the tool was never executed because a transition fired instead, the history would appear "broken" to the next engine. Dux solves this by injecting a **synthetic tool result** into history before the transition:

```text
assistant: "I will now transition to execution mode."
   ↓ (transition tool fires)
system (hidden): "State Machine Transition successfully hooked to mode 'execution'."
tool (synthetic): "State Machine Transition successfully hooked."
```

This synthetic result satisfies the provider's expectation that every `ToolRequestPart` is followed by a `ToolResultPart`, preventing the next engine from dropping or misinterpreting the history context.

## A Conceptual Illustration

The following Go sketch demonstrates the structural pattern of mode transitions. It is simplified—there is no real LLM provider, no tool execution, and no streaming—but it captures the control flow:

```go
package main

import (
	"context"
	"fmt"
)

// Simplified transition signal
type TransitionSignal struct {
	TargetMode string
	Message    string
}

// A mode definition
type Mode struct {
	Name        string
	System      string
	Transitions []string // allowed target modes
}

// Mock engine that either returns text or a transition
type MockEngine struct {
	mode   Mode
	turns  int
}

func (e *MockEngine) Stream(ctx context.Context, input string) (string, *TransitionSignal) {
	e.turns++
	fmt.Printf("[%s turn %d] System: %s\n", e.mode.Name, e.turns, e.mode.System)

	switch e.mode.Name {
	case "aide":
		if e.turns == 1 {
			return "Delegating to planner...", &TransitionSignal{TargetMode: "planning", Message: "User wants a health-check endpoint."}
		}
		return "All done. Health-check is live.", nil
	case "planning":
		return "Plan created: 1) Add route, 2) Add test, 3) Update README.", &TransitionSignal{TargetMode: "execution", Message: "Plan approved."}
	case "execution":
		return "Code written and tests pass.", &TransitionSignal{TargetMode: "aide", Message: "Execution complete."}
	}
	return "Unknown mode", nil
}

// WorkflowEngine that hot-swaps modes
type WorkflowEngine struct {
	modes   map[string]Mode
	current *MockEngine
}

func NewWorkflowEngine(initial string, modes map[string]Mode) *WorkflowEngine {
	return &WorkflowEngine{
		modes:   modes,
		current: &MockEngine{mode: modes[initial]},
	}
}

func (w *WorkflowEngine) Run(ctx context.Context, userInput string) {
	msg := userInput
	for {
		output, trans := w.current.Stream(ctx, msg)
		fmt.Println("Output:", output)

		if trans == nil {
			fmt.Println("\n=== Converged ===")
			return
		}

		fmt.Printf("\n>>> Transition to '%s' (reason: %s)\n\n", trans.TargetMode, trans.Message)
		// Hot-swap engine, preserving no state except the transition message
		w.current = &MockEngine{mode: w.modes[trans.TargetMode]}
		msg = trans.Message // bridge message becomes next input
	}
}

func main() {
	modes := map[string]Mode{
		"aide":     {Name: "aide", System: "You are Aide. Gather requirements, then delegate."},
		"planning": {Name: "planning", System: "You are a planner. Break tasks into atomic steps."},
		"execution": {Name: "execution", System: "You are an executor. Write code and run tests."},
	}

	engine := NewWorkflowEngine("aide", modes)
	engine.Run(context.Background(), "Add a health-check endpoint.")
}
```

Running this produces:

```text
[aide turn 1] System: You are Aide. Gather requirements, then delegate.
Output: Delegating to planner...

>>> Transition to 'planning' (reason: User wants a health-check endpoint.)

[planning turn 1] System: You are a planner. Break tasks into atomic steps.
Output: Plan created: 1) Add route, 2) Add test, 3) Update README.

>>> Transition to 'execution' (reason: Plan approved.)

[execution turn 1] System: You are an executor. Write code and run tests.
Output: Code written and tests pass.

>>> Transition to 'aide' (reason: Execution complete.)

[aide turn 2] System: You are Aide. Gather requirements, then delegate.
Output: All done. Health-check is live.

=== Converged ===
```

In the real Dux implementation, the `WorkflowEngine` wraps a real `adapter.Engine`, the modes are loaded from `pkg/mode.Definition` structs, and the transition signal is emitted by a special `transition` tool registered in each mode's tool set.

## When to Define Custom Modes

The built-in `aide/planning/execution/review` quartet is tuned for software development, but the primitive is general. Consider defining custom modes when:

- A task has clearly separable phases with different safety requirements.
- You want to use a cheaper model for planning and an expensive model for execution (each mode can override the provider).
- Different UIs should expose different capabilities (a Slack bot might only expose the `aide` mode, while a background cron job runs `execution` directly).
- You need an approval gate: a `planning` mode can create a plan, but the plan must be approved by a human before `execution` is allowed to start.

The mode system is declarative in YAML and fully programmable in Go. It is the architectural primitive that elevates Dux from a chatbot wrapper to a structured workflow engine.
