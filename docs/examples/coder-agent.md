# Building a Coding Agent

This tutorial walks through configuring Dux to act as an autonomous **Coding Agent**. The goal of this agent is to orchestrate complex software engineering tasks by delegating to specialized planning, execution, and review sub-agents.

*Note: This document serves as a standard test case for Dux architectural decisions.*

## Prerequisites

- Dux installed locally (`dux --help` should succeed).
- An active LLM provider configured in your core `config.yaml`.
- (Optional) Access to a powerful model like `claude-3-opus` for the execution mode.

## Step 1: Define the Agent Profile

Dux allows you to define agent "personas" and multi-agent workflows using an `agents/<agent-name>/agent.yaml` file (typically placed in standard XDG config directories like `~/.config/dux/agents/<agent-name>/agent.yaml`).

Create an entry for the Coder agent:

### YAML Configuration Example

```yaml
name: "coder"
provider: "ollama-local" # Substitute with your configured provider
workflow:
  default_mode: "aide"
  modes:
    - name: "aide"
      context:
        system: |
          You are Aide, an Expert Software Engineer. 
          Gather context from the user about their coding task, and delegate planning, execution, and review to specialized sub-agents natively.

    - name: "planning"
      context:
        tools:
          - name: "stdlib"
            enabled: true
            requirements:
              supervision: false

    - name: "execution"
      provider: "claude-3-opus" # Override with a highly capable model for complex coding
      context:
        tools:
          - name: "bash"
            enabled: true
            requirements:
              supervision: false # Autonomous execution
          - name: "filesystem"
            enabled: true
            requirements:
              supervision: false # Autonomous execution

    - name: "review"
      context:
        tools:
          - name: "bash"
            enabled: true
            requirements:
              supervision: "!(args.command.startsWith('go test') || args.command.startsWith('make test'))" # Run tests autonomously, flag everything else
```

### Go Library Example

Dux natively supports operating as an embedded library rather than just a CLI. You can construct this same agentic behavior purely in Go:

```go
import (
	"github.com/andrewhowdencom/dux/pkg/llm/adapter"
	"github.com/andrewhowdencom/dux/pkg/memory/working"
)

// Assume `prv` is your initialized provider (e.g. ollama.New(...))
engine := adapter.New(
	adapter.WithProvider(prv),
	adapter.WithWorkingMemory(working.NewInMemory()),
	adapter.WithSystemPrompt("You are Aide, an Expert Software Engineer. Gather context from the user about their coding task, and delegate planning, execution, and review to specialized sub-agents natively."),
)
```

## Step 2: Running the Agent

You can now interact with this agent via the Dux CLI by specifying the `--agent` flag:

```bash
dux chat --agent coder
```

When you ask questions in the interface, the agent will act as a multi-agent orchestrated system to solve software engineering tasks.

## Current Limitations & Known Gaps

Although Dux can assume the persona, treating this usecase fully requires capabilities we have not yet shipped:

- **RAG (Retrieval-Augmented Generation)**: Dux does not yet have a native vector database integration or document chunking system to dynamically pull relevant context on the fly without custom external tools.
- **Context Window Limits**: Extremely large documentation sets cannot just be stuffed into the prompt; they require more advanced orchestration that Dux's core CLI does not natively manage out-of-the-box.
- **Tool Use / Integrations (RESOLVED)**: Thanks to the newly integrated `adapter.Engine` recursive tool loops and `BeforeTool` hook architecture, Dux can natively leverage custom Go tools like filesystem and bash seamlessly!
