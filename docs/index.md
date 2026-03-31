---
hide:
  - navigation
---

# dux 🦆

![Dux Architecture](assets/banner.jpg)

<p align="center"><em>Fast harness for agents. Latin for "guide". Short enough for the CLI!</em></p>

---

Dux is a lightning-fast, highly modular execution engine for running and testing Large Language Model (LLM) agents locally. Build and iterate on intricate provider streams, tool abstractions, and recursive convergence loops straight from the synchronous terminal.

## Key Features

- **Agnostic LLM Engine**: Implemented via a deep recursive `adapter` mapping sequence, allowing your pipeline to scale continuously across `static` testing mocks, raw `ollama` daemon inference endpoints, and beyond!
- **Dynamic Viper Configurations**: Connect any provider natively via `config.yaml` using powerful generic ID-based mappings without muddying CLI source boundaries.
- **Lightning Fast CLI Repl**: Run `dux chat` completely locally. Dux's synchronous stream REPL flawlessly maps raw output chunks linearly without ugly asynchronous rendering race conditions.
- **Strictly Typed Tool Abstractions**: Write Go functions and easily export them to LLMs directly via standard JSON Schema mappings natively supported by Go interfaces.

## Where to go next?

Explore the documentation through its Diátaxis structure:

- 🎓 **[Tutorials](tutorials/litellm.md)**: Step-by-step generic integration lessons.
- 🚀 **[How-To Guides](how-to/running-locally.md)**: Target guides for local configuration and building.
- 📖 **[Reference](reference/cli.md)**: API specifications and CLI Commands.
- 🧠 **[Explanation](explanation/architecture.md)**: Deep dives into the LLM "convergence loop" theory and history abstractions.
