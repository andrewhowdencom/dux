# Building a Customer Support Agent

This tutorial outlines how to use Dux to construct a **Customer Support Agent** — an assistant that can directly answer user interrogations by referencing a knowledge base.

*Note: This document serves as a standard test case for Dux architectural decisions.*

## Prerequisites

- Dux installed.
- A functional LLM provider specified in `config.yaml`.

## Step 1: Agent Configuration

In your `agents.yaml` file, define the support agent profile. You can also utilize Dux's built-in "enrichers" to ensure the agent has real-time context (like the current OS or time) when talking to a user.

### YAML Configuration Example

```yaml
- name: "support-assistant"
  provider: "ollama-local"
  context:
    system: |
      You are a helpful, empathetic Customer Support Assistant.
      Address the user's issue directly and concisely. If you do not know the answer,
      apologize and recommend they contact a human agent.
    enrichers:
      - type: "time"
      - type: "os"
```

### Go Library Example

If you are embedding this support workflow inside an existing Go application, you can orchestrate the context and enrichers dynamically:

```go
import (
	"github.com/andrewhowdencom/dux/pkg/llm/adapter"
	"github.com/andrewhowdencom/dux/pkg/llm/enrich"
	"github.com/andrewhowdencom/dux/pkg/llm/history"
)

engine := adapter.New(
	adapter.WithProvider(prv), // Pre-configured provider
	adapter.WithHistory(history.NewInMemory()),
	adapter.WithSystemPrompt("You are a helpful, empathetic Customer Support Assistant..."),
	adapter.WithEnrichers([]enrich.Enricher{
		enrich.NewTime(),
		enrich.NewOS(),
	}),
)
```

## Step 2: Interacting with the Agent

Launch the TUI with the configured agent:

```bash
dux chat --agent support-assistant
```

## Current Limitations & Known Gaps

To fully realize a Customer Support Agent, Dux needs further architectural investments to overcome these gaps:

- **RAG (Retrieval-Augmented Generation)**: Dux does not currently support connecting to vector databases or local knowledge bases. The agent can only answer questions based on its foundational training data, making it unsuitable for highly specific, proprietary support queries.
- **Conversation State Persistence**: While in-memory history is tracked during a session, Dux cannot recall past support tickets for a specific user across separate sessions.
- **Tone and Guardrails**: There is no built-in mechanism to enforce guardrails (e.g., preventing the AI from making legal promises or hallucinating refunds) outside of simple zero-shot prompting in the system file.
