# Building a Guild Management Agent

This tutorial explains how to use Dux to power an **Internal Guild Management Agent**. This agent helps run internal developer "guilds" or communities of practice by handling administrative overhead: sourcing talks, summarizing meeting notes, and generating internal marketing emails.

*Note: This document serves as a standard test case for Dux architectural decisions.*

## Prerequisites

- Dux configured with a capable text-generation LLM.
- `agents.yaml` configured.

## Step 1: Profile Setup

Create an agent profile tailored to corporate communication and summarization:

### YAML Configuration Example

```yaml
- name: "guild-manager"
  provider: "ollama-local"
  context:
    system: |
      You are the Lead Administrator for the internal Engineering Guild.
      Your tone should be professional, encouraging, and highly organized.
      Format all meeting recaps distinctly with Action Items and Key Takeaways.
```

### Go Library Example

If you're building a Slack bot or internal web service to handle guild management using Dux's library, initialization looks like this:

```go
import (
	"github.com/andrewhowdencom/dux/pkg/llm/adapter"
	"github.com/andrewhowdencom/dux/pkg/llm/history"
)

engine := adapter.New(
	adapter.WithProvider(prv),
	adapter.WithHistory(history.NewInMemory()),
	adapter.WithSystemPrompt("You are the Lead Administrator for the internal Engineering Guild..."),
)
```

## Step 2: Generating Artifacts

Run the agent dynamically:

```bash
dux chat --agent guild-manager
```

You can feed raw transcripts from meeting recordings or bullet points to have the agent write fully fleshed-out recap emails.

## Current Limitations & Known Gaps

Managing an internal guild requires multifaceted coordination. Dux is currently limited by the following gaps:

- **Multi-Agent Workflows**: Typically, you require a pipeline: Agent A extracts transcripts, Agent B summarizes, and Agent C writes the marketing copy. Dux does not currently support orchestrating multiple agents in a single workflow.
- **External API Integrations**: Dux cannot read Google Calendar to identify upcoming guild meetings, nor can it post recaps directly to internal Slack channels or Confluence wikis.
- **Asynchronous Execution**: There is no daemon or cron-like orchestration in Dux to process long background jobs (e.g., waiting for an hour-long meeting video to be transcribed, then summarizing it).
