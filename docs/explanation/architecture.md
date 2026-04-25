# Architecture Principles

Dux is structured around a clean separation between its **library surface** (`pkg/`) and its **application mapping layer** (`internal/`). This boundary ensures that the core agentic primitives remain free of configuration-framework leaks (YAML tags, Viper maps, CLI concerns) and can be embedded directly into other Go programs.

## Core Primitives

The engine is built on three foundational agentic primitives, each documented in dedicated explanation pages:

1. **[The Recursive Convergence Loop](convergence-loop.md)** — The core execution cycle that transforms a simple LLM chat into an autonomous agent by repeatedly streaming generation, detecting tool requests, executing them, and feeding results back until the model converges on a final answer.

2. **[Mode Transitions & Workflow Orchestration](mode-transitions.md)** — A state-machine layer that allows a single agent to host multiple personas (orchestrator, planner, executor, reviewer) and transition between them without dropping the session or UI connection.

3. **[The Lifecycle Hook Framework](lifecycle-hooks.md)** — A five-phase interception pipeline (BeforeStart, BeforeGenerate, BeforeTool, AfterTool, AfterComplete) that decouples cross-cutting concerns such as memory injection, guardrails, telemetry, and human supervision from the core engine logic.

A fourth cross-cutting primitive, **[Safety & Supervision](safety-and-supervision.md)**, layers human-in-the-loop approval and policy-based gating on top of the execution loop.

## Provider Abstraction

Every LLM backend (OpenAI, Gemini, Ollama, Static mocks) implements a uniform `Provider` interface that emits strongly-typed `Part` values—`TextPart`, `ReasoningPart`, `ToolRequestPart`, `TelemetryPart`—rather than raw byte streams. This eliminates brittle string-parsing and allows downstream adapters (terminal, web, Slack) to branch their rendering logic based on semantic intent rather than heuristics.

The CLI layer (`internal/`) deserializes YAML configurations into concrete provider constructors via Viper, but library consumers bypass this entirely by calling provider packages directly (`ollama.New(...)`, `gemini.New(...)`, `static.New(...)`).

## Synchronous Turn Blocking

Each iteration of the convergence loop blocks until the LLM stream has fully concluded and all pending tool executions have resolved. This produces a race-condition-free runtime where tool side-effects and history mutations are strictly ordered, at the cost of some throughput. The design prioritizes **predictability over parallelism** because agentic tool chains are inherently sequential: the LLM cannot reason about the result of `file_read` until `file_read` has actually executed.

## Adapter Engine

The `pkg/llm/adapter` package hosts the recursive convergence engine. It can be wrapped by higher-order engines—such as the `WorkflowEngine`—which intercept transition signals and hermetically hot-swap the inner engine's configuration (provider, system prompt, tools) while preserving the shared session history. This nesting is what makes multi-mode agents possible without external orchestrators.

---

For deeper understanding of each primitive, follow the linked explanation pages above.
