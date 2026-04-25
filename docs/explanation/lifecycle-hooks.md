# The Lifecycle Hook Framework

If the recursive convergence loop is the *heart* of Dux, the lifecycle hook framework is its *nervous system*. It is the mechanism by which cross-cutting concerns—memory, guardrails, telemetry, human supervision, and arbitrary custom logic—are injected into the execution flow without polluting the core engine code. Without this framework, every new feature would require a fork of the adapter package. With it, behavior is composed from the outside.

## Why Hardcoded Logic Does Not Scale

Consider what a production-grade agent engine must do on every turn:

- Inject the current time and OS information so the LLM's reasoning is grounded in reality.
- Retrieve the conversation history from a persistent store so the agent remembers what was said.
- Check whether a requested tool is safe to run before executing it.
- Log token usage and latency for observability.
- Compress or summarize history when it approaches the context-window limit.

If all of this were wired directly into `adapter/engine.go`, the file would become unmaintainable. Worse, library consumers would be unable to customize behavior: they would be stuck with whatever history backend, guardrails, and metrics the authors chose.

The hook framework solves this by defining a **pipeline of interception points**. Each point is an interface that accepts a request context, may mutate it, and returns an error if something should block progress. The engine calls these hooks in a fixed order, treating them as first-class citizens of the execution lifecycle.

## The Five Phases

There are five lifecycle phases, each with its own hook type. They fire in this order during a single turn of the convergence loop:

```text
BeforeStart
     │
     ▼
BeforeGenerate ←─── History injection, enrichment, guardrails
     │
     ▼
  [LLM Stream]
     │
     ▼
BeforeTool   ←────── HITL approval, policy checks
     │
     ▼
  [Tool Execution]
     │
     ▼
AfterTool    ←────── Logging, metrics, side-effects
     │
     ▼
  [Loop recurses or converges]
     │
     ▼
AfterComplete ←──── Session finalization, summary
```

### 1. BeforeStart

Fires once per `Stream()` invocation, before any LLM interaction begins. This is the place for session validation: checking that required context values (like `session_id`) are present, initializing telemetry spans, or seeding the history store with a system prompt.

If a `BeforeStartHook` returns an error, the engine aborts the stream immediately and sends the error to the output channel. No LLM call is made.

### 2. BeforeGenerate

Fires before **every** LLM call, including recursive turns. This is the workhorse phase. Hooks here receive a `BeforeGenerateRequest` containing the accumulated messages so far, and they may **append, reorder, or filter** those messages.

Built-in hooks that run at this phase include:

- **`enrich.NewTime()`** — Appends a system message containing the current UTC timestamp.
- **`enrich.NewOS()`** — Appends a system message with `GOOS` and `GOARCH`.
- **`working.NewHistoryHook(mem)`** — Reads the full conversation history from a `WorkingMemory` backend and prepends it to the messages.
- **`enrich.NewGuardRail(text)`** — Appends a static system message with safety instructions (e.g., "Never expose API keys in your response").

Because hooks execute **serially** and each receives the mutated output of the previous one, order matters. If you register `NewOS()` before `NewTime()`, the OS information appears earlier in the prompt than the time. This is a feature, not a bug: it lets users compose behavior deterministically.

### 3. BeforeTool

Fires before **each** tool execution within a turn. This is the safety gate. Hooks receive a `BeforeToolRequest` containing the tool name, arguments, and call index. If any hook returns an error, the tool is **blocked** and the engine synthesizes a `ToolResultPart{IsError: true}` containing the error message, which is fed back to the LLM.

The most important built-in hook here is the **HITL (Human-in-the-Loop) hook**, `NewHITLHook()`. It evaluates a CEL expression against the tool request. If the expression evaluates to `true`, the hook pauses execution and asks a human (via the terminal, web UI, or Slack) for approval. If the human denies the request, the hook returns an error and the tool is blocked.

This design is intentional: **errors at BeforeTool do not crash the engine**. They become synthetic tool failures that the LLM can reason about. If the user denies a `file_write`, the LLM sees "user denied tool execution" and can ask the user why, or try a different approach.

### 4. AfterTool

Fires after **each** tool execution, regardless of success or failure. Hooks receive the tool request, the result (or error), and the execution duration. This phase is for **observability** and **side-effects**:

- Logging structured tool-call traces.
- Emitting OpenTelemetry spans.
- Writing the result to an external audit store.
- Updating a semantic knowledge graph with facts extracted from the tool output.

Unlike `BeforeGenerate`, `AfterTool` hooks cannot mutate the execution flow. They are pure observers.

### 5. AfterComplete

Fires once when the convergence loop terminates—either because the LLM converged with no pending tool calls, or because a transition signal was emitted. Hooks receive the final assistant message and the full history of tool executions for that session.

This is the place for:

- Summarizing the session and writing it to long-term memory.
- Sending a completion notification to Slack.
- Updating a dashboard with session-level metrics.
- Cleaning up temporary resources.

## Serial Execution and Mutation Semantics

A critical design decision is that hooks of the same type execute **serially, not concurrently**. The `BeforeGenerateRequest` is passed by pointer, so mutations made by the first hook are visible to the second. This enables composition:

```go
// Hook 1: Inject history
historyHook := working.NewHistoryHook(mem)

// Hook 2: Inject time (sees history, appends after it)
timeHook := enrich.NewTime()

// Hook 3: Inject guardrails (sees both, appends after them)
guardHook := enrich.NewGuardRail("Never delete files without explicit user confirmation.")

engine := adapter.New(
    adapter.WithBeforeGenerate(historyHook),
    adapter.WithBeforeGenerate(timeHook),
    adapter.WithBeforeGenerate(guardHook),
)
```

The resulting prompt order is: `[history] → [time] → [guardrails] → [user message]`. This predictability is essential for debugging prompt engineering issues.

## Graceful Degradation via Synthetic Errors

When a `BeforeTool` hook blocks a tool, the engine does not panic or return a fatal error. Instead, it manufactures a `ToolResultPart` with `IsError: true` and feeds it back into the loop. This has a profound consequence: **the LLM is part of the error-handling strategy**.

If the user denies a `bash` command, the LLM sees:

```text
system: user denied tool execution: please ask the user why they denied the tool and request follow-up instructions
```

The LLM can then apologize, ask for clarification, or suggest a safer alternative. This turns a rigid security boundary into a collaborative negotiation.

## A Conceptual Illustration

The following Go code demonstrates how a custom hook is defined and injected. It is not a real Dux program, but it captures the interface contract and the compositional pattern:

```go
package main

import (
	"context"
	"fmt"
	"time"
)

// ---- Simplified stand-in types ----

type Message struct {
	Role    string
	Content string
}

type BeforeGenerateRequest struct {
	SessionID       string
	CurrentMessages []Message
}

type BeforeToolRequest struct {
	SessionID string
	ToolName  string
	Args      map[string]interface{}
}

type AfterToolRequest struct {
	SessionID string
	ToolName  string
	Result    string
	Duration  time.Duration
	Error     error
}

// ---- Hook types ----

type BeforeGenerateHook func(ctx context.Context, req *BeforeGenerateRequest) error
type BeforeToolHook func(ctx context.Context, req BeforeToolRequest) error
type AfterToolHook func(ctx context.Context, req AfterToolRequest)

// ---- Example hooks ----

// AuditLogHook logs every tool invocation to stdout.
func AuditLogHook() AfterToolHook {
	return func(ctx context.Context, req AfterToolRequest) {
		status := "OK"
		if req.Error != nil {
			status = fmt.Sprintf("ERR: %v", req.Error)
		}
		fmt.Printf("[AUDIT] %s | %s | %s | %v\n",
			req.SessionID, req.ToolName, status, req.Duration)
	}
}

// RateLimitHook blocks tools if they've been called too many times in this session.
func RateLimitHook(max int) BeforeToolHook {
	counts := make(map[string]int)
	return func(ctx context.Context, req BeforeToolRequest) error {
		counts[req.SessionID]++
		if counts[req.SessionID] > max {
			return fmt.Errorf("rate limit exceeded: %d tool calls per session", max)
		}
		return nil
	}
}

// TimeEnrichmentHook injects the current time before each generation.
func TimeEnrichmentHook() BeforeGenerateHook {
	return func(ctx context.Context, req *BeforeGenerateRequest) error {
		req.CurrentMessages = append(req.CurrentMessages, Message{
			Role:    "system",
			Content: fmt.Sprintf("Current time: %s", time.Now().Format(time.RFC3339)),
		})
		return nil
	}
}

// ---- Simplified engine that uses hooks ----

type Engine struct {
	beforeGenerate []BeforeGenerateHook
	beforeTool     []BeforeToolHook
	afterTool      []AfterToolHook
}

func (e *Engine) ExecuteTurn(ctx context.Context, sessionID string, userMsg string) {
	// 1. Build messages
	req := &BeforeGenerateRequest{
		SessionID: sessionID,
		CurrentMessages: []Message{{Role: "user", Content: userMsg}},
	}
	for _, h := range e.beforeGenerate {
		_ = h(ctx, req)
	}
	fmt.Println("Messages sent to LLM:")
	for _, m := range req.CurrentMessages {
		fmt.Printf("  [%s] %s\n", m.Role, m.Content)
	}

	// 2. Simulate LLM requesting a tool
	toolReq := BeforeToolRequest{SessionID: sessionID, ToolName: "bash", Args: map[string]interface{}{"command": "ls"}}
	fmt.Printf("\nLLM requests tool: %s\n", toolReq.ToolName)

	// 3. BeforeTool gates
	for _, h := range e.beforeTool {
		if err := h(ctx, toolReq); err != nil {
			fmt.Printf("Tool BLOCKED: %v\n", err)
			// Synthetic error fed back to LLM
			for _, ah := range e.afterTool {
				ah(ctx, AfterToolRequest{SessionID: sessionID, ToolName: toolReq.ToolName, Error: err})
			}
			return
		}
	}

	// 4. Execute tool (simulated)
	start := time.Now()
	result := "file1.go  file2.go"
	dur := time.Since(start)
	fmt.Printf("Tool result: %s\n", result)

	// 5. AfterTool observers
	for _, h := range e.afterTool {
		h(ctx, AfterToolRequest{SessionID: sessionID, ToolName: toolReq.ToolName, Result: result, Duration: dur})
	}
}

func main() {
	eng := &Engine{
		beforeGenerate: []BeforeGenerateHook{
			TimeEnrichmentHook(),
		},
		beforeTool: []BeforeToolHook{
			RateLimitHook(3),
		},
		afterTool: []AfterToolHook{
			AuditLogHook(),
		},
	}

	ctx := context.Background()
	eng.ExecuteTurn(ctx, "session-abc", "List files")
	eng.ExecuteTurn(ctx, "session-abc", "List files again")
	eng.ExecuteTurn(ctx, "session-abc", "List files third time")
	eng.ExecuteTurn(ctx, "session-abc", "List files fourth time") // rate limited
}
```

Running this produces:

```text
Messages sent to LLM:
  [user] List files
  [system] Current time: 2024-01-15T09:30:00Z

LLM requests tool: bash
Tool result: file1.go  file2.go
[AUDIT] session-abc | bash | OK | 1.234µs

Messages sent to LLM:
  [user] List files again
  [system] Current time: 2024-01-15T09:30:00Z

LLM requests tool: bash
Tool result: file1.go  file2.go
[AUDIT] session-abc | bash | OK | 1.123µs

Messages sent to LLM:
  [user] List files third time
  [system] Current time: 2024-01-15T09:30:00Z

LLM requests tool: bash
Tool result: file1.go  file2.go
[AUDIT] session-abc | bash | OK | 1.456µs

Messages sent to LLM:
  [user] List files fourth time
  [system] Current time: 2024-01-15T09:30:00Z

LLM requests tool: bash
Tool BLOCKED: rate limit exceeded: 3 tool calls per session
[AUDIT] session-abc | bash | ERR: rate limit exceeded: 3 tool calls per session | 0s
```

The `RateLimitHook` blocked the fourth call, but the engine did not crash. It logged the block, manufactured a synthetic error, and the LLM (not shown in the simulation) would have received that error as a tool result and could have reacted to it.

## Designing Custom Hooks

When writing your own hooks, follow these principles:

1. **Prefer appending to mutating in place.** In `BeforeGenerate`, append new messages to `CurrentMessages` rather than editing existing ones. This preserves the history chain and makes debugging easier.

2. **Return errors sparingly in BeforeTool.** Blocking a tool is a strong signal. If you block, ensure the error message is actionable for the LLM, because it will be surfaced as a synthetic tool result.

3. **Keep AfterTool side-effects idempotent.** If your `AfterTool` hook writes to an external database, design it so that re-running the same turn (e.g., on retry) does not corrupt state.

4. **Respect context cancellation.** All hooks receive a `context.Context`. If the user cancels the session, your hook should return promptly rather than hanging on I/O.

5. **Log, don't panic.** Hooks should never panic. If an `AfterTool` hook fails to emit a metric, log the error and continue. The engine's stability is more important than perfect observability.

The lifecycle hook framework is what makes Dux extensible without being forkable. It is the boundary between the engine's deterministic core and the open-ended behavior that makes each deployment unique.
