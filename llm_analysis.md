> **Note:** This project is archived. While this design was interesting, the author (Andy) is going to have another go at this that's much slimmer. This document remains as a historical artifact of the original design thinking.

# Exhaustive Analysis: Elevating `pkg/llm` to a Versatile Standalone Library

This document contains a comprehensive analysis of the `pkg/llm` and `pkg/llm/provider` packages. It outlines their current architectural state and details the critical concepts and components required to transform these packages into a robust, standalone library capable of powering advanced AI applications (including memory management, tool orchestration, and RAG).

---

## 1. Current State Assessment

Currently, `pkg/llm` acts as an excellent, abstracted interface layer for the `dux` CLI. 

### Strengths & Good Design Choices
- **Clean Modality Separation (`llm.Part`)**: By breaking down `Message` structures into strongly-typed `Part` interfaces (`TextPart`, `ReasoningPart`, `ToolRequestPart`, `TelemetryPart`), you have successfully avoided the brittle string-parsing traps of early LLM wrappers.
- **I/O Abstraction**: The `Sender` and `Receiver` interfaces paired with the `SessionHandler` decouple the LLM engine from the underlying transport (CLI, HTTP, Stdio), making it highly adaptable regarding *where* messages come from and go to.
- **Provider Agnosticism**: The `Generator` and `Embedder` interfaces cleanly hide the underlying complexities of Gemini and Ollama. The use of functional options (`WithModel`, `WithAddress`) is idiomatic and clean.

### Limitations for External Reuse
- **Deep Coupling**: It currently lives as a sub-package inside `dux/pkg/llm`. A true library should ideally have its own Go module with decoupled dependencies.
- **Bring-Your-Own-Orchestration**: The library provides the raw blocks to talk to an LLM, but places the entire burden of state management, context sliding, and data retrieval on the caller.

---

## 2. Introducing "Memory" Concepts

Currently, `GenerateStream` accepts an arbitrary `[]llm.Message`. There is no built-in state. To be truly useful in complex applications, the library needs formalized memory management.

### Short-Term Memory (Conversation State)
Applications need to easily track ongoing conversational turns.
- **`HistoryStore` Interface**: An abstraction representing where messages live. Built-in implementations should include:
  - `InMemoryHistoryStore` (slice-based, volatile)
  - `SQLiteHistoryStore` or `FileHistoryStore` (persistent)
- **Token-Aware Windowing (Sliding Memory)**: As conversations grow, they breach context limits. The library needs a `WindowManager` that intelligently prunes history.
  - Using the `TelemetryPart` emitted by providers, the `WindowManager` tracks token usage and can truncate the oldest messages (excluding `System` instructions) dynamically.
- **Summary Memory**: For very long contexts, instead of simply evicting old messages, a BeforeGenerate hook could trigger a background LLM call to summarize evicted messages and prepend the summary to the history.

### Long-Term Memory (Semantic / Episodic)
For an agent to "remember" items from past, separate sessions.
- **Entity & Fact Extraction**: The library could provide an `Interceptor` or agent step that automatically extracts and stores facts (e.g., "User's timezone is EST") from the stream.
- **Semantic Retrieval**: Automatically fetching relevant past interactions by comparing the current query embedding against a Vector Store of past memories.

---

## 3. Implementing Retrieval-Augmented Generation (RAG)

While `Embedder` yields `[][]float32`, there is zero pipeline infrastructure for RAG. A generic LLM library usually provides a set of composable primitives for this.

### Required Primitives
1. **Documents & Loaders**
   - **`Document` Struct**: Needs to hold `PageContent (string)` and `Metadata (map[string]any)`.
   - **`DocumentLoader` Interface**: Standardized way to ingest data (`FileLoader`, `URLLoader`, `MarkdownLoader`).
   
2. **Text Splitters / Chunkers**
   - A naive LLM wrapper sends whole files. A good library provides configurable text splitters based on tokens, characters, or markdown headers (e.g., `RecursiveCharacterTextSplitter`) to optimize embedding payloads and retrieval accuracy.

3. **Vector Stores**
   - Standardized interface bridging your `Embedder` with a database.
   ```go
   type VectorStore interface {
       AddDocuments(ctx context.Context, docs []Document) error
       SimilaritySearch(ctx context.Context, query string, topK int) ([]Document, error)
   }
   ```
   - *Implementations*: You currently use `chromem-go`. By strictly interfacing it here, users of the library could swap it out for PostgreSQL (pgvector) or Qdrant later.

4. **The Retriever Abstraction**
   - The link between the Vector Store and the Prompt.
   - `Retriever` interface that takes a string and returns `[]Document`. This abstracts away whether the documents come from vector search, keyword search, or an API call.

---

## 4. Agent Frameworking & Tool Orchestration

Currently, your `pkg/llm` returns `ToolRequestPart`s perfectly. However, the logic to execute that tool and feed the result back (the ReAct loop) is likely bespoke logic elsewhere (perhaps inside `Engine` implementations or hitl loops). 

To be a powerful library, it should offer a standardized Agent Runtime:
- **`ToolRegistry`**: A clean way to register tools (like your bash, file, and MCP tools) with the system.
- **`AgentExecutor`**: A loop that automatically:
  1. Calls `GenerateStream`.
  2. If an array of `TextPart` is yielded, it streams to the user.
  3. If a `ToolRequestPart` is yielded, it automatically pauses, routes the execution to the `ToolRegistry`, gets the result, appends a `ToolResultPart`, and **automatically re-invokes** the LLM for the conclusion.

## Summary Architecture for the "Ideal Library"

If abstracted into an independent project (e.g., `go-llm`), the taxonomy would look like:

- `go-llm/core`: `Message`, `Part`, Interfaces.
- `go-llm/provider`: Gemini, Ollama, Vertex, Groq integrations.
- `go-llm/memory`: Sliding windows, Summary buffers, persistence.
- `go-llm/rag`: Document Loaders, Chunkers, VectorStore interfaces.
- `go-llm/agent`: Tool routing, ReAct loops, multi-agent orchestrators.

## Next Steps
To evolve `pkg/llm` in this direction within `dux`, consider:
1. **Creating a `memory` subpackage**: Define a `History` interface and implement context sliding rather than passing raw slices directly to `GenerateStream`.
2. **Elevating the RAG primitives**: Lift your `chromem-go` implementation out of specific CLI commands into a formalized `pkg/llm/rag` or `vectorstore` package using the interfaces mentioned above.
