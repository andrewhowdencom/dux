# Building a Customer Feedback Agent

This tutorial walks through configuring Dux to act as a **Customer Feedback Agent**. The goal of this agent is to ingest raw user feedback, categorize it (e.g., bug, feature request, praise), and synthesize a summary for product teams.

*Note: This document serves as a standard test case for Dux architectural decisions.*

## Prerequisites

- Dux installed locally (`dux --help` should succeed).
- An active LLM provider configured in your core `config.yaml`.

## Step 1: Define the Agent Profile

Dux allows you to define agent "personas" using an `agents/<agent-name>/agent.yaml` file (typically placed in standard XDG config directories like `~/.config/dux/agents/<agent-name>/agent.yaml`).

Create an entry for the feedback agent:

### YAML Configuration Example

```yaml
- name: "feedback-analyzer"
  provider: "ollama-local" # Substitute with your configured provider
  context:
    system: |
      You are a Customer Feedback Analyzer.
      Your job is to read user feedback and output a structured summary.
      Categorize the feedback into: BUG, FEATURE, or PRAISE.
      Provide a 1-sentence summary of the core issue.
```

### Go Library Example

Dux natively supports operating as an embedded library rather than just a CLI. You can construct this same agentic behavior purely in Go:

```go
import (
	"github.com/andrewhowdencom/dux/pkg/llm/adapter"
	"github.com/andrewhowdencom/dux/pkg/llm/workmem"
)

// Assume `prv` is your initialized provider (e.g. ollama.New(...))
engine := adapter.New(
	adapter.WithProvider(prv),
	adapter.WithWorkingMemory(workmem.NewInMemory()),
	adapter.WithSystemPrompt("You are a Customer Feedback Analyzer. Your job is to read user feedback and output a structured summary..."),
)
```

## Step 2: Running the Agent

You can now interact with this agent via the Dux CLI by specifying the `--agent` flag:

```bash
dux chat --agent feedback-analyzer
```

When you paste customer feedback into the interface, the agent will automatically apply the configured system prompt and categorization instructions.

## Current Limitations & Known Gaps

Although Dux can assume the persona, treating this use case fully requires capabilities we have not yet shipped. Future architectural decisions should aim to close these gaps:

- **Batch Processing**: Dux currently operates primarily via interactive chat. It lacks a native mechanism to pipe large CSVs or databases of feedback directly into the agent for bulk processing.
- **Output Schemas (Structured JSON)**: We cannot yet force the LLM to strictly output valid JSON schema to be piped into a downstream ticketing system (like Jira).
- **Tool Use / Integrations (RESOLVED)**: Thanks to the newly integrated `adapter.Engine` recursive tool loops and `ToolMiddleware` architecture, Dux can natively leverage custom Go tools to seamlessly query a Zendesk API or write directly to a Notion board!
