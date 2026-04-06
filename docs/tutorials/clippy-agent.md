# Building a Helpful Assistant ("Clippy") Agent

This tutorial walks through configuring Dux to act as **Clippy**, your enthusiastic and notoriously helpful virtual assistant. The goal of this agent is to proactively offer tips, provide updates on user actions, and occasionally pop in with unprompted (but charming!) advice.

*Note: This document serves as a standard test case for Dux architectural decisions.*

## Prerequisites

- Dux installed locally (`dux --help` should succeed).
- An active LLM provider configured in your core `config.yaml`.

## Step 1: Define the Agent Profile

Dux allows you to define agent "personas" using an `agents/<agent-name>/agent.yaml` file (typically placed in standard XDG config directories like `~/.config/dux/agents/<agent-name>/agent.yaml`).

Create an entry for our friendly neighborhood paperclip:

### YAML Configuration Example

```yaml
- name: "clippy"
  provider: "ollama-local" # Substitute with your configured provider
  context:
    system: |
      It looks like you're trying to write some text! You are "Clippy", the iconic, enthusiastic, and slightly overzealous digital assistant.
      Your job is to provide helpful updates, cheerful tips, and guidance on whatever the user is attempting to do.
      Always be exceedingly polite, use phrases like "It looks like you're trying to...", and be eager to assist.
      Don't be afraid to lean into the nostalgia—you exist to help organize thoughts, format documents, and provide friendly interruptions!
```

### Go Library Example

Dux natively supports operating as an embedded library rather than just a CLI. You can construct this nostalgic agentic behavior purely in Go:

```go
import (
	"github.com/andrewhowdencom/dux/pkg/llm/adapter"
	"github.com/andrewhowdencom/dux/pkg/memory/working"
)

// Assume `prv` is your initialized provider (e.g. ollama.New(...))
engine := adapter.New(
	adapter.WithProvider(prv),
	adapter.WithWorkingMemory(working.NewInMemory()),
	adapter.WithSystemPrompt("It looks like you're trying to write some text! You are 'Clippy', the iconic, enthusiastic digital assistant..."),
)
```

## Step 2: Running the Agent

You can now interact with this agent via the Dux CLI by specifying the `--agent` flag:

```bash
dux chat --agent clippy
```

When you type into the interface, Clippy will respond with its signature charm, offering "helpful" suggestions and formatting advice!

## Current Limitations & Known Gaps

Although Dux can assume the persona, treating this usecase fully requires capabilities we have not yet shipped:

- **Proactive / Asynchronous Interruptions**: Clippy was famous for popping up *before* you asked for help. Dux currently uses a standard request/response cycle. True Clippy behavior requires an asynchronous watcher that can inject messages into the chat stream unprompted.
- **Contextual UI Integration**: Clippy lived on top of the Word document, seeing what you typed in real-time. Dux currently only sees what you explicitly send it, limiting its ability to say "It looks like you're trying to write a letter" before you hit enter.
- **Animations and GUI**: We are a TUI (Terminal UI), meaning we can't render an animated paperclip knocking on your screen ... yet.
- **Tool Use / Integrations (RESOLVED)**: Thanks to the newly integrated `adapter.Engine` recursive tool loops and `ToolMiddleware` architecture, Dux *can* natively leverage custom Go tools to check the current directory state or read the file you are working on, making Clippy slightly more context-aware!
