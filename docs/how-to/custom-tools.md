# Adding Custom Tools to the Engine

When using Dux as a generic library, extending your LLM workflows with highly specific system capabilities is straightforward. Unlike hidden global registries, Dux expects you to implement clear Go interfaces and inject them into the generative loop.

## 1. Implement the `llm.Tool` Interface

A `Tool` simply requires an identifier, a JSON Schema providing strict LLM parameter constraints, and the logical execution code.

```go
package main

import (
	"context"
	"encoding/json"

	"github.com/andrewhowdencom/dux/pkg/llm"
)

type WeatherTool struct{}

func (w *WeatherTool) Name() string { return "get_weather" }

func (w *WeatherTool) Definition() llm.ToolDefinitionPart {
	schema := `{
	  "type": "object",
	  "properties": {
		"location": {
		  "type": "string",
		  "description": "The city name, e.g. London"
		}
	  },
	  "required": ["location"]
	}`
	
	return llm.ToolDefinitionPart{
		Name:        w.Name(),
		Description: "Fetches current weather information.",
		Parameters:  json.RawMessage(schema),
	}
}

// Execute performs your side-effects when the stream resolves a request matching Name()
func (w *WeatherTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	location, _ := args["location"].(string)
	
	// Perform arbitrary Go code here (e.g. hitting an API)
	return map[string]string{
		"weather": "25C",
		"location": location,
	}, nil
}
```

## 2. Injecting your custom Tools

Dux resolves tools asynchronously using the `llm.ToolProvider` interface, ensuring capabilities like MCP (remote tool protocols) can dynamically provide schemas.

For simple compiled Go structures like the one above, bundle them using the built-in `static` resolver and pass them as a functional option when constructing the engine.

```go
import (
	"context"
	"fmt"

	"github.com/andrewhowdencom/dux/pkg/llm"
	"github.com/andrewhowdencom/dux/pkg/llm/adapter"
	"github.com/andrewhowdencom/dux/pkg/llm/tool/static"
)

// Construct your standard business tools
myTools := static.New("weather", &WeatherTool{})

// Build the engine with the custom tool resolver
engine := adapter.New(
	adapter.WithProvider(prv),   // your pre-configured provider
	adapter.WithResolver(myTools),
)

// Stream a message that may invoke get_weather
ctx := llm.WithSessionID(context.Background(), "demo-session")
msg := llm.Message{
	Identity: llm.Identity{Role: "user"},
	Parts:    []llm.Part{llm.TextPart("What is the weather in London?")},
}

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
```
