# Architecture Principles

The `dux` execution engine is structured to prioritize recursive parsing natively via the `adapter` layer. This decoupling separates the CLI/Server handling routes from the raw provider integrations.

Every provider logic implementation inherits a strict `Provider` interface abstraction, forcing compliance to normalized stream mappings. Because large language models constantly output data streams over WebSockets via chunked encoding layers, this simplifies the interface down to raw generator streams.

### Provider Factory
To avoid leaking provider semantics to consumers (e.g. your application), the CLI layer (`internal/ui/factory.go`) deserializes Viper generic data objects parsed dynamically (`map[string]any`) from `config.yaml` and constructs the concrete provider. Library consumers bypass this entirely by calling provider constructors directly (e.g. `ollama.New(...)`, `static.New(...)`).

### Adapter Layers
Unlike standard raw strings mapped straight to `.Stdout`, `pkg/llm/adapter` allows users to nest execution engines recursively. This creates standard abstractions useful for testing tools:
1.  **Static Adapter Mocks**: Useful for injecting pre-compiled response buffers directly into the agent network to simulate standard behaviors linearly.
2.  **Tool Mappings**: Tools execute arbitrary backend functions by exporting generic interfaces serialized automatically to strict JSON Schemas mapped natively to provider-level function calling parameters.

### Synchronous State Loops
Because execution loops rely strictly on predictable recursive rendering of tool executions and subsequent follow-ups to LLMs, the system leverages sync mapping loops natively blocking execution contexts until full stream conclusion. This produces a perfectly predictable, race-condition-free runtime environment.
