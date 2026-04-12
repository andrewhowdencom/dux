# Building a Conversation Bot Agent

This tutorial walks through configuring Dux to act as a **Conversation Bot Agent**. The goal of this agent is to ingest knowledge bases, documentation, or unstructured data and answer user queries accurately based solely on the provided context.

*Note: This document serves as a standard test case for Dux architectural decisions.*

## Prerequisites

- Dux installed locally (`dux --help` should succeed).
- An active LLM provider configured in your core `config.yaml`.

## Step 1: Define the Agent Profile

Dux allows you to define agent "personas" using an `agents/<agent-name>/agent.yaml` file (typically placed in standard XDG config directories like `~/.config/dux/agents/<agent-name>/agent.yaml`).

Create an entry for the Conversation Bot agent:

### YAML Configuration Example

```yaml
name: "conversation-bot"
provider: "ollama-local" # Substitute with your configured provider
workflow:
  default_mode: "conversation-bot"
  modes:
    - name: "conversation-bot"
      context:
        system: |
          You are a precise, helpful Conversation Bot.
          Your job is to answer user questions based on the provided context or knowledge base.
          If you do not know the answer based on the provided context, clearly state that you do not know.
          Provide concise, direct answers and optionally cite the relevant part of the source material.
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
	adapter.WithSystemPrompt("You are a precise, helpful Conversation Bot. Your job is to answer user questions based on the provided context..."),
)
```

## Step 2: Running the Agent

You can now interact with this agent via the Dux CLI by specifying the `--agent` flag:

```bash
dux chat --agent conversation-bot
```

When you ask questions in the interface, the agent will reply based on the given context (which can be injected into the chat context or tool outputs).

## Current Limitations & Known Gaps

Although Dux can assume the persona, treating this usecase fully requires capabilities we have not yet shipped:

- **RAG (Retrieval-Augmented Generation)**: Dux does not yet have a native vector database integration or document chunking system to dynamically pull relevant context on the fly without custom external tools.
- **Context Window Limits**: Extremely large documentation sets cannot just be stuffed into the prompt; they require more advanced orchestration that Dux's core CLI does not natively manage out-of-the-box.
- **Tool Use / Integrations (RESOLVED)**: Thanks to the newly integrated `adapter.Engine` recursive tool loops and `ToolMiddleware` architecture, Dux can natively leverage custom Go tools to query an external search index or vector database seamlessly!
