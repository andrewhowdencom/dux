# Building a Customer Support Agent

This tutorial outlines how to use Dux to construct a **Customer Support Agent** — an assistant that can directly answer user interrogations by referencing a knowledge base.

*Note: This document serves as a standard test case for Dux architectural decisions.*

## Prerequisites

- Dux installed.
- A functional LLM provider specified in `config.yaml`.

## Step 1: Agent Configuration

In your `agents/<agent-name>/agent.yaml` file, define the support agent profile. You can also utilize Dux's built-in "enrichers" to ensure the agent has real-time context (like the current OS or time) when talking to a user.

### YAML Configuration Example

```yaml
name: "support-assistant"
provider: "ollama-local"
workflow:
  default_mode: "support"
  modes:
    - name: "support"
      context:
        system: |
          You are a helpful, empathetic Customer Support Assistant.
          Address the user's issue directly and concisely. If you do not know the answer,
          apologize and recommend they contact a human agent.
        tools:
          - name: "time"
            enabled: true
            requirements:
              supervision: false
        enrichers:
          - type: "os"
```

### Go Library Example

If you are embedding this support workflow inside an existing Go application, you can orchestrate the context and enrichers dynamically:

```go
import (
	"github.com/andrewhowdencom/dux/pkg/llm"
	"github.com/andrewhowdencom/dux/pkg/llm/adapter"
	"github.com/andrewhowdencom/dux/pkg/llm/enrich"
	"github.com/andrewhowdencom/dux/pkg/memory/working"
	"github.com/andrewhowdencom/dux/pkg/llm/tool/static"
	"github.com/andrewhowdencom/dux/pkg/llm/tool/stdlib"
)

// Configure core execution engine
engine := adapter.New(
	adapter.WithProvider(prv), // Pre-configured provider
	adapter.WithWorkingMemory(working.NewInMemory()),
	adapter.WithSystemPrompt("You are a helpful, empathetic Customer Support Assistant..."),
	adapter.WithEnrichers([]llm.BeforeGenerateHook{
		enrich.NewOS(),
	}),
	adapter.WithResolver(static.New("stdlib", stdlib.New())),
)

```

## Step 2: Interacting with the Agent

Launch the TUI with the configured agent:

```bash
dux chat --agent support-assistant
```

## Current Limitations & Known Gaps

To fully realize a Customer Support Agent, Dux needs further architectural investments to overcome these gaps:

- **RAG (Retrieval-Augmented Generation) (RESOLVED)**: `adapter.Engine` now natively loops tool calls. By injecting a simple custom Go `ToolProvider` equipped with a vector DB search integration, Dux can actively query local knowledge bases and seamlessly retrieve context necessary for complex support queries without any external orchestrators!
- **Conversation State Persistence**: While in-memory history is tracked during a session, Dux cannot recall past support tickets for a specific user across separate sessions.
- **Tone and Guardrails**: There is no built-in mechanism to enforce guardrails (e.g., preventing the AI from making legal promises or hallucinating refunds) outside of simple zero-shot prompting in the system file.
