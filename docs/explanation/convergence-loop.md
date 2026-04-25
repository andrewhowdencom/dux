# The Recursive Convergence Loop

A Large Language Model, on its own, is a stateless text generator. It receives a prompt and emits a completion. To turn this into an *agent*—a system that can read files, run commands, query databases, and reason about the results—the model must be able to *act* and then *continue reasoning* based on the outcome of that action. Dux achieves this through the **recursive convergence loop**.

## Why a Loop Is Necessary

Imagine asking an agent: "Find all TODO comments in this codebase and create a summary." A single-shot LLM response cannot do this. The agent needs to:

1. Search the filesystem for Go files.
2. Read each file to scan for `TODO` strings.
3. Collate the findings.
4. Produce a summary.

Each step depends on the result of the previous one. The LLM does not know what files exist until it calls `file_list`. It cannot read `TODO`s until it knows the file paths. This is not a pipeline that can be pre-planned; it must be *discovered* through execution. The convergence loop is the mechanism that enables this discovery.

## What the Loop Looks Like

At its heart, the loop is simple. The engine repeatedly asks the LLM to generate a response, inspects that response for tool requests, executes any tools found, and feeds the results back into the conversation history before asking again. This continues until the LLM returns a response that contains no tool requests—at which point the loop has **converged** on a final answer.

A single iteration looks like this:

```text
┌─────────────────┐
│ Build Messages  │ ← Combine system prompt, history, tool definitions,
│   (Hooks)       │   and any pending tool results from prior turns.
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Stream from LLM │ ← The provider yields Parts: text chunks, reasoning
│   (Provider)    │   tokens, tool requests, telemetry.
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Detect Tools    │ ← Scan the streamed Parts for ToolRequestParts.
│   (Adapter)     │
└────────┬────────┘
         │
   ┌─────┴─────┐
   │ Any tools?│
   └─────┬─────┘
   Yes /   \ No
      /     \
     ▼       ▼
┌────────┐  ┌─────────────────┐
│ Execute│  │ Converged!      │
│  Tools │  │ Fire AfterComplete
│  (Hook)│  │ Return final text
└───┬────┘  └─────────────────┘
    │
    ▼
┌─────────────────┐
│ Append Results  │ ← ToolResultParts written to history.
│   (History)     │
└─────────────────┘
         │
         └──────→ Loop again (back to Build Messages)
```

## Synchronous Blocking per Turn

Dux makes an explicit architectural choice: **each turn is fully synchronous**. The engine does not start the next LLM call until every tool from the current turn has finished executing and its result has been safely appended to the session history.

This may seem conservative. Why not parallelize tool calls or pipeline the next generation while tools run? The answer is **predictability**. Agentic reasoning is not embarrassingly parallel. If `file_read` and `bash` both execute simultaneously, their side-effects may interleave in ways the LLM cannot foresee. Worse, if the LLM asks for `file_read("config.yaml")` and then `file_write("config.yaml", "...")` in the same turn, reordering them would be catastrophic.

By blocking, Dux guarantees that:

- The history is a strict, immutable sequence of events.
- Tool side-effects are ordered exactly as the LLM requested.
- There are no race conditions between the LLM's reasoning and the state of the world.

The cost is throughput; the benefit is **debuggability and safety**.

## Termination Conditions

The loop does not run forever. It terminates when one of three conditions is met:

1. **No pending tool calls.** The LLM's response contains only text, reasoning, or telemetry. The task is complete.
2. **A transition signal is emitted.** A special tool (the `transition` tool) returns a `TransitionSignalPart`, instructing the `WorkflowEngine` to swap the active mode. The inner loop terminates so the outer workflow can restart with a new engine configuration.
3. **Context cancellation.** The user presses `Ctrl+C`, the HTTP connection drops, or a parent timeout fires. The engine drains the current turn and exits cleanly.

## A Conceptual Illustration

The following Go code is not the exact engine implementation, but it captures the structural essence of the loop:

```go
package main

import (
	"context"
	"fmt"
)

// Simplified stand-in for llm.Message and llm.Part
type Part interface{ IsToolRequest() bool }
type TextPart struct{ Content string }
func (TextPart) IsToolRequest() bool { return false }

type ToolRequestPart struct {
	Name string
	Args map[string]interface{}
}
func (ToolRequestPart) IsToolRequest() bool { return true }

type Message struct {
	Role  string // "user", "assistant", "tool"
	Parts []Part
}

// A mock provider that decides whether to ask for a tool or answer directly.
type MockProvider struct{ turn int }

func (p *MockProvider) Generate(ctx context.Context, history []Message) []Part {
	p.turn++
	if p.turn == 1 {
		// First turn: the LLM asks to read a file.
		return []Part{ToolRequestPart{Name: "file_read", Args: map[string]interface{}{"path": "/etc/os-release"}}}
	}
	// Second turn: it answers based on the (simulated) tool result.
	return []Part{TextPart{Content: "You are running Ubuntu 22.04."}}
}

// A mock executor that handles the tool.
func executeTool(req ToolRequestPart) string {
	if req.Name == "file_read" {
		return "PRETTY_NAME=\"Ubuntu 22.04.4 LTS\""
	}
	return "unknown tool"
}

func main() {
	provider := &MockProvider{}
	var history []Message

	// THE CONVERGENCE LOOP
	for {
		// 1. Generate
		parts := provider.Generate(context.Background(), history)

		// 2. Detect tool requests
		var pending []ToolRequestPart
		var textParts []Part
		for _, p := range parts {
			if tr, ok := p.(ToolRequestPart); ok {
				pending = append(pending, tr)
			} else {
				textParts = append(textParts, p)
			}
		}

		// 3. Record assistant message
		history = append(history, Message{Role: "assistant", Parts: parts})

		// 4. If no tools, we have converged.
		if len(pending) == 0 {
			fmt.Println("Converged:", textParts[0].(TextPart).Content)
			break
		}

		// 5. Execute tools and append results
		for _, tr := range pending {
			result := executeTool(tr)
			history = append(history, Message{
				Role: "tool",
				Parts: []Part{TextPart{Content: result}},
			})
		}
	}
}
```

Running this produces:

```text
Converged: You are running Ubuntu 22.04.
```

The real Dux engine is substantially more complex—it handles streaming, hook interception, telemetry, and history backends—but the recursive structure remains identical: **generate, inspect, execute, recurse, converge**.

## Relation to Other Primitives

The convergence loop does not exist in isolation. It is the *stage* on which other primitives perform:

- **Hooks** fire at specific phases within each turn (before generation, before tool execution, after completion).
- **History** persists the accumulated messages so the loop maintains coherent context across turns.
- **Transitions** break the loop cleanly so a higher-order engine can restart it with new parameters.
- **Supervision** gates tool execution inside the loop, pausing it for human approval.

Understanding the loop is the key to understanding why Dux behaves the way it does: deterministic, sequential, and transparent.
