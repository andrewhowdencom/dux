# How-To: Using Dux as a Go Library

While Dux offers a robust command-line interface (CLI) to orchestrate Large Language Models (LLMs) via YAML configurations, its core architecture is actively designed as a purely idiomatic Go library. This means you can drop Dux directly into your own Go applications, APIs, or data pipelines, without interacting with the `dux` CLI or YAML parsing natively.

## The Dual-Use Architecture

Dux enforces a strict boundary between its application mapping layer (`internal/`) and its foundational primitives (`pkg/`). This prevents "configuration framework leakages" (like Viper maps or YAML tags) from tainting the pure structural capabilities of the library.

### Option 1: The CLI Flow (YAML)

For users running the `dux` binary directly, agents are easily defined inside `agents/<agent-name>/agent.yaml`. String values are mapped dynamically via the internal `cli` package:

```yaml
# agents/<agent-name>/agent.yaml
- name: "qa"
  provider: "static"
  context:
    system: "You are a helpful agent."
    enrichers:
      - type: "time"
      - type: "os"
```

The CLI application (`internal/cli/factory.go`) deserializes `"time"` and `"os"`, translating them into internal `pkg/` constructors.

### Option 2: The Library Flow (Pure Go)

For library consumers, you sidestep YAML mapping entirely. You instantiate the exact same `qa` agent using pure Go constructor injection and the **Variadic Options Pattern**:

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/andrewhowdencom/dux/pkg/llm"
	"github.com/andrewhowdencom/dux/pkg/llm/adapter"
	"github.com/andrewhowdencom/dux/pkg/llm/enrich"
	"github.com/andrewhowdencom/dux/pkg/llm/workmem"
	"github.com/andrewhowdencom/dux/pkg/llm/provider/static"
)

func main() {
	// 1. Initialize the Provider
	// Use your desired LLM backend. The static provider simply returns canned text.
	providerCfg := map[string]interface{}{"text": "Hello from the pure Go library!"}
	prv, err := static.New(providerCfg)
	if err != nil {
		log.Fatalf("failed to create provider: %v", err)
	}

	// 2. Initialize the Engine using Variadic Options
	// Notice how we explicitly call explicit constructors instead of resolving strings.
	engine := adapter.New(
		adapter.WithProvider(prv),
		adapter.WithWorkingMemory(workmem.NewInMemory()),
		adapter.WithSystemPrompt("You are a helpful agent."),
		adapter.WithEnrichers(
			enrich.NewTime(),
			enrich.NewOS(),
		),
	)

	// 3. Coordinate messaging
	ctx := context.Background()
	msg := llm.Message{
		SessionID: "my-custom-session",
		Identity:  llm.Identity{Role: "user"},
		Parts:     []llm.Part{llm.TextPart("Can you verify this setup?")},
	}

	// 4. Stream Results
	stream, err := engine.Stream(ctx, msg)
	if err != nil {
		log.Fatalf("failed to stream: %v", err)
	}

	for yield := range stream {
		for _, part := range yield.Parts {
			if textPart, ok := part.(llm.TextPart); ok {
				fmt.Print(textPart)
			}
		}
	}
}
```

## Why Pure Go Interfaces?

When importing `github.com/andrewhowdencom/dux/pkg/llm`, consumers are completely spared from internal dependencies (such as the `dux/internal` config models). 
* **Type-safety**: Options like `enrich.NewTime()` are strictly bound by the `enrich.Enricher` interface natively at compile time.
* **Flexibility**: You can write your *own* custom providers or enrichers in your application and inject them variadically into `adapter.New()` without having to "register" them globally.
