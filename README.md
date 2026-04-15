# dux

![Dux Architecture](docs/assets/banner.jpg)
*Fast harness for agents. Latin for "guide". Short enough for the CLI!*

Dux is a lightning-fast, highly modular execution engine for running and testing Large Language Model (LLM) agents locally. Build and iterate on intricate provider streams, tool abstractions, and recursive convergence loops straight from the synchronous terminal.

## Key Features

- **Agnostic LLM Engine**: Implemented via a deep recursive `adapter` mapping sequence, allowing your pipeline to scale continuously across `static` testing mocks, raw `ollama` daemon inference endpoints, and beyond!
- **Dynamic Viper Configurations**: Connect any provider natively via `config.yaml` using powerful generic ID-based mappings without muddying CLI source boundaries.
- **Lightning Fast CLI Repl**: Run `dux chat` completely locally. Dux's synchronous stream REPL flawlessly maps raw output chunks linearly without ugly asynchronous rendering race conditions.
- **Strictly Typed Tool Abstractions**: Write Go functions and easily export them to LLMs directly via standard JSON Schema mappings natively supported by Go interfaces.

## Quick Start

```bash
# Clone the repository
git clone https://github.com/andrewhowdencom/dux.git
cd dux

# Compile Dux
go build

# Set up your environment (Targeting your local Ollama)
cp config.example.yaml ~/.config/dux/config.yaml

# Start an interactive chat session
./dux chat --agent my-agent

# Run an agent with its configured triggers (schedules, events, chat)
./dux run my-agent

# Execute a one-shot query via stdin
echo "What time is it?" | ./dux invoke my-agent

# Start UI servers (web, telegram, slack) defined in config
./dux serve
```

## Available Commands

- `dux chat` - Interactive REPL chat session with an agent
- `dux run [agent]` - Run an agent with all its configured triggers
- `dux invoke [agent]` - Execute a single query via stdin
- `dux serve [type]` - Start UI servers (web/telegram/slack)
- `dux llm` - Manage LLM provider configurations

## Documentation

For extensive project breakdown covering development, architecture, and guides, refer to our [Diátaxis-compliant](https://diataxis.fr/) documentation structure within `docs/`:

- [README](README.md) (You are here!)
- `docs/tutorials/`: Step-by-step agent integration lessons.
- `docs/how-to/`: Quick answers for complex LLM deployment configuration.
- `docs/reference/`: API specifications and JSON Schema mappings.
- `docs/explanation/`: Deep dives into LLM "convergence loop" theory and history abstractions.
