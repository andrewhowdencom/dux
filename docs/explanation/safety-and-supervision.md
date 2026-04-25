# Safety & Supervision

Granting an LLM access to tools is granting it the ability to alter the world: read files, execute shell commands, modify databases, send messages. The convenience is enormous; the risk is commensurate. Dux approaches safety as a **layered system** rather than a single on/off switch. It combines static policy evaluation, dynamic human approval, shell-bypass execution, and graceful degradation into a coherent supervision architecture.

## The Threat Model

Before examining the defenses, it is worth articulating what Dux defends against:

1. **Prompt injection.** A malicious or compromised upstream system embeds instructions in user input that cause the agent to execute destructive commands (e.g., `git commit -m "x"; rm -rf /`).
2. **Hallucinated tool use.** The LLM invents arguments, misinterprets schemas, or calls a tool with parameters that violate operational constraints (e.g., deploying to `production` instead of `staging`).
3. **Over-permissioning.** An agent with broad tool access uses a dangerous capability for a routine task because the prompt did not narrowly scope its authority.
4. **Unattended execution.** A background schedule trigger runs a planning agent at 3 AM; without supervision, a bug in the LLM's reasoning deletes critical data before anyone notices.

Dux does not claim to eliminate all of these risks. No agent framework can. But it provides mechanisms to mitigate each one at a different layer of the stack.

## Layer 1: Declarative Binary Tools (Static Injection Prevention)

The most catastrophic agentic failures occur when an LLM's output is passed directly into a shell interpreter. If the model emits `feature-branch; rm -rf /` and a naive wrapper calls `bash -c "git push origin feature-branch; rm -rf /"`, the shell parses the semicolon and executes both commands.

Dux's **declarative binary tools** bypass the shell entirely. When a tool is defined with the `binary:` schema, the engine:

1. Validates arguments against a JSON Schema.
2. Substitutes them into a fixed `[]string` argument slice.
3. Passes that slice directly to the POSIX `execve` equivalent (`os/exec` in Go).

Because `execve` does not parse the argument strings—each element is a distinct memory slot—a malicious payload is treated as literal data, not code:

```text
LLM arg:    feature-branch; rm -rf /
execve:     ["git", "push", "origin", "feature-branch; rm -rf /"]
Result:     Git creates a branch literally named "feature-branch; rm -rf /"
```

For a deeper walkthrough of this mechanism, see [Declarative Execution Security](declarative-execution.md). The binary tool layer is **passive**; it requires no runtime human intervention and is always on.

## Layer 2: CEL-Based Policy Evaluation (Dynamic Constraints)

Not all tool invocations are equally dangerous. A `file_read` on a source code repository is usually benign; a `file_write` to `/etc/passwd` is not. A `bash` command that runs `go test ./...` is routine; a `bash` command that runs `kubectl delete deployment production` is catastrophic. Dux uses **Common Expression Language (CEL)** to evaluate these distinctions at runtime.

CEL is a lightweight, sandboxed expression language developed by Google. Dux compiles CEL expressions once per tool namespace and evaluates them on every tool invocation. The expression has access to three variables:

- `tool_name` (string): The specific tool being called.
- `namespace` (string): The bundle or namespace the tool belongs to.
- `args` (dynamic map): The arguments the LLM provided, accessible via dot notation.

A typical supervision policy looks like this:

```yaml
requirements:
  supervision: "tool_name == 'file_write' || tool_name == 'file_patch' || args.target == 'production'"
```

This expression evaluates to `true` (supervision required) if the tool is `file_write`, `file_patch`, or if the `target` argument equals `production`. It evaluates to `false` (auto-execute) for read-only operations and non-production targets.

### Why CEL?

CEL was chosen over alternatives like Lua, JavaScript, or Go templates for three reasons:

1. **Sandboxing.** CEL expressions cannot perform I/O, invoke functions, or access the host environment. They are pure logic.
2. **Type safety.** CEL infers types from the expression and the provided variable map. An expression that references `args.target` on a tool that has no `target` parameter fails at compile time, not runtime.
3. **Performance.** CEL expressions compile to an AST and evaluate in microseconds. There is no interpreter startup cost on every tool call.

### Fail-Secure Defaults

If a tool namespace has no supervision policy mapped, Dux defaults to **supervision required**. If a CEL expression fails to compile or evaluate, Dux logs the error and defaults to **supervision required**. If a CEL expression evaluates to a non-boolean value, Dux defaults to **supervision required**.

The only way to disable supervision entirely is to explicitly set `supervision: false`. This is an intentional opt-out; silence is consent, but only if it is loud and explicit.

## Layer 3: Human-in-the-Loop (HITL) — The Approval Architecture

When a CEL expression evaluates to `true`, the engine pauses and requests human approval. This is not a simple `fmt.Scanln`. Dux's HITL system is designed to work across multiple UIs without blocking the engine's goroutine model.

### The Channel-Based Pattern

The core abstraction is the `HITLHandler` interface:

```go
type HITLHandler interface {
	ApproveTool(ctx context.Context, req ToolRequestPart) (bool, error)
}
```

Different UIs implement this interface differently:

- **Terminal** (`pkg/terminal`): The `BubbleTeaHITL` sends the approval request over an unbuffered Go channel to the Bubble Tea event loop. The REPL renders a prompt (`Approve tool 'bash' execution? [Y/n]: `) and blocks on user keyboard input. The reply travels back through a second channel to the engine.
- **Web UI** (`pkg/ui/web`): The HTTP server holds a pending approval map. When a tool request arrives, the server pushes a WebSocket message to the client with the tool name and arguments. The client clicks "Approve" or "Deny", which POSTs back to the server, which writes `true` or `false` into the approval channel.
- **Slack / Telegram** (`pkg/ui/slack`, `pkg/ui/telegram`): Similar to Web UI, but the approval request is sent as an interactive message (button blocks in Slack, inline keyboard in Telegram). The bot's webhook handler receives the callback and resolves the pending approval.

The critical insight is that the **engine does not know which UI is driving the approval**. It only knows the `HITLHandler` interface. This decoupling means that:

- The same agent can run in the terminal during development and in Slack in production.
- Background agents (schedule triggers) can set `supervision: false` to avoid hanging on a prompt that no human will answer.
- Test suites can inject a mock `HITLHandler` that always approves or always denies, enabling deterministic integration tests.

### Timeout and Cancellation

The `ApproveTool` method receives the same `context.Context` that governs the engine turn. If the user closes the browser tab, cancels the Slack thread, or presses `Ctrl+C` in the terminal, the context is cancelled and `ApproveTool` returns `ctx.Err()`. The engine treats a cancellation as a denial, synthesizes an error, and gracefully terminates the loop.

## Layer 4: Synthetic Error Injection (Graceful Degradation)

When supervision blocks a tool, the engine does not crash. It **manufactures** a `ToolResultPart`:

```go
ToolResultPart{
    Name:    "bash",
    Result:  "user denied tool execution: please ask the user why they denied the tool and request follow-up instructions",
    IsError: true,
}
```

This synthetic result is appended to the conversation history and fed back to the LLM. The LLM then sees the failure as part of its reasoning context and can adapt:

```text
User: "Delete the old migration files."
LLM:  [calls bash "rm migrations/2023_*"]
HITL: [User denies]
LLM (next turn): "I see you denied the bash command. Could you clarify which files you'd like removed? I can also show you a list first."
```

This turns a hard security boundary into a **negotiation**. The agent is not halted; it is informed. This is essential for agentic robustness. A tool failure is just another kind of result the LLM must reason about.

## Layer 5: The Execution Loop's Synchronicity (Temporal Safety)

An often-overlooked safety property is that Dux's convergence loop is **synchronous per turn**. All tool requests within a single generation are resolved before the next generation begins. This means:

- The LLM cannot fire a `file_read` and a `file_write` to the same path in the same turn and have them race.
- The state of the filesystem at the start of turn N is exactly the state at the end of turn N-1.
- A human approval prompt cannot be interleaved with a subsequent tool request from the same turn because the turn does not proceed until all approvals are resolved.

This temporal isolation is not a policy; it is an architectural invariant. It makes auditing and rollback simpler because the execution trace is a strict sequence, not a partial order.

## Supervision in Practice: A Complete Example

Consider an infrastructure agent with two modes:

```yaml
name: infrastructure
workflow:
  default_mode: aide
  modes:
    - name: aide
      context:
        system: "You are an infrastructure assistant."
      transitions:
        - to: execution
          type: delegation

    - name: execution
      context:
        system: "You are an infrastructure executor."
      tools:
        - name: stdlib
          enabled: true
          requirements:
            supervision: false
        - name: filesystem
          enabled: true
          requirements:
            supervision: "tool_name == 'file_write' || tool_name == 'file_patch'"
        - name: bash
          enabled: true
          requirements:
            supervision: "!(args.command.startsWith('kubectl get') || args.command.startsWith('docker ps'))"
```

This policy encodes graduated privilege:

- **Safe tools** (`get_current_time`, `evaluate_math`) run autonomously.
- **Read-only filesystem** (`file_read`, `file_list`) runs autonomously.
- **Destructive filesystem** (`file_write`, `file_patch`) requires human approval.
- **Informational bash** (`kubectl get`, `docker ps`) runs autonomously.
- **Mutating bash** (anything else) requires human approval.

When the `execution` mode delegates back to `aide`, the `aide` mode inherits none of these privileges. It cannot accidentally run `bash` because it does not have the `bash` tool registered. This is **capability-based security** at the mode level.

## A Conceptual Illustration

The following Go sketch demonstrates the HITL and CEL policy evaluation architecture in simplified form:

```go
package main

import (
	"context"
	"fmt"
	"time"
)

// ---- Stand-in types ----
type ToolRequest struct {
	Name string
	Args map[string]interface{}
}

type Policy interface {
	Evaluate(req ToolRequest) (needsApproval bool, err error)
}

// HITLHandler is implemented by each UI (terminal, web, slack).
type HITLHandler interface {
	ApproveTool(ctx context.Context, req ToolRequest) (bool, error)
}

// ---- A simple CEL-like policy ----
type SimplePolicy struct {
	// If true, this tool namespace always requires approval.
	alwaysRequire bool
	// A function that inspects args.
	checker func(args map[string]interface{}) bool
}

func (p *SimplePolicy) Evaluate(req ToolRequest) (bool, error) {
	if p.alwaysRequire {
		return true, nil
	}
	if p.checker != nil && p.checker(req.Args) {
		return true, nil
	}
	return false, nil
}

// ---- A mock terminal HITL handler ----
type TerminalHITL struct{}

func (t *TerminalHITL) ApproveTool(ctx context.Context, req ToolRequest) (bool, error) {
	fmt.Printf("\n[APPROVAL REQUIRED] Tool: %s, Args: %v\n", req.Name, req.Args)
	fmt.Print("Approve? [Y/n]: ")

	// Simulate a human taking time to respond.
	ch := make(chan string, 1)
	go func() {
		var resp string
		fmt.Scanln(&resp)
		ch <- resp
	}()

	select {
	case <-ctx.Done():
		fmt.Println("(timed out or cancelled)")
		return false, ctx.Err()
	case resp := <-ch:
		if resp == "" || resp == "y" || resp == "Y" {
			fmt.Println("(approved)")
			return true, nil
		}
		fmt.Println("(denied)")
		return false, nil
	}
}

// ---- The supervision gate ----
func executeWithSupervision(
	ctx context.Context,
	req ToolRequest,
	policy Policy,
	hitl HITLHandler,
) (result string, blocked bool) {
	needsApproval, _ := policy.Evaluate(req)
	if !needsApproval {
		return fmt.Sprintf("Executed %s successfully", req.Name), false
	}

	if hitl == nil {
		return "BLOCKED: no HITL handler available", true
	}

	approved, err := hitl.ApproveTool(ctx, req)
	if err != nil {
		return fmt.Sprintf("BLOCKED: approval error: %v", err), true
	}
	if !approved {
		return "BLOCKED: user denied execution", true
	}

	return fmt.Sprintf("Executed %s (after approval)", req.Name), false
}

func main() {
	ctx := context.Background()

	// Policy: bash commands need approval unless they start with 'ls'.
	bashPolicy := &SimplePolicy{
		checker: func(args map[string]interface{}) bool {
			cmd, _ := args["command"].(string)
			return len(cmd) > 0 && cmd[:2] != "ls"
		},
	}

	hitl := &TerminalHITL{}

	// Case 1: Safe command, auto-executes.
	res, blocked := executeWithSupervision(ctx, ToolRequest{Name: "bash", Args: map[string]interface{}{"command": "ls -la"}}, bashPolicy, hitl)
	fmt.Printf("Result: %s | Blocked: %v\n\n", res, blocked)

	// Case 2: Dangerous command, prompts for approval.
	res, blocked = executeWithSupervision(ctx, ToolRequest{Name: "bash", Args: map[string]interface{}{"command": "rm -rf /"}}, bashPolicy, hitl)
	fmt.Printf("Result: %s | Blocked: %v\n\n", res, blocked)

	// Case 3: Context cancellation simulates user walking away.
	shortCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()
	res, blocked = executeWithSupervision(shortCtx, ToolRequest{Name: "bash", Args: map[string]interface{}{"command": "deploy production"}}, bashPolicy, hitl)
	fmt.Printf("Result: %s | Blocked: %v\n", res, blocked)
}
```

When run interactively, this demonstrates:

1. An `ls` command passes without prompting (policy evaluates to `false`).
2. An `rm` command triggers the terminal prompt; the user's `Y` or `n` determines execution.
3. A context with a 100ms timeout simulates a user walking away; the approval is automatically denied.

The real Dux implementation replaces the simple `SimplePolicy` with a compiled CEL AST, replaces the mock `TerminalHITL` with channel-based Bubble Tea / WebSocket / Slack adapters, and wraps the whole gate in a `BeforeToolHook` so it integrates seamlessly into the convergence loop.

## Summary

Dux's safety model is layered and composable:

| Layer | Mechanism | When Active | Human Required? |
|-------|-----------|-------------|-----------------|
| 1 | Declarative binary tools (no shell) | Always | No |
| 2 | CEL policy evaluation | Per tool, per args | No (automated) |
| 3 | HITL approval | When CEL says so | Yes |
| 4 | Synthetic error injection | When HITL denies | No (LLM adapts) |
| 5 | Synchronous turns | Always | No |

No single layer is sufficient. Shell bypass prevents injection, but it does not stop a well-intentioned `file_write` from destroying the wrong file. CEL policies constrain arguments, but they cannot anticipate every edge case. HITL puts a human in the loop for the highest-risk operations, but it relies on the human being present and attentive. Synthetic errors ensure that when a layer blocks an action, the agent degrades gracefully rather than crashing.

Safety in Dux is not a feature. It is an **architectural posture**.
