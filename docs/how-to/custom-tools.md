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

Dux resolves tools asynchronously using the `llm.ToolResolver` interface, ensuring capabilities like MCP (remote tool protocols) can dynamically provide schemas.

For simple compiled Go structures like the one above, bundle them using the built-in `static` resolver and map them as a functional option to the `SessionHandler`.

```go
import (
	"github.com/andrewhowdencom/dux/pkg/llm"
	"github.com/andrewhowdencom/dux/pkg/llm/tool/static"
)

// Construct your standard business tools
myTools := static.New(
	&WeatherTool{},
)

// Orchestrate the new session
handler := llm.NewSessionHandler(
	engine, 
	receiver, 
	sender,
	llm.WithResolver(myTools), // Native Tool interception & recursion happens here!
)

handler.ListenAndServe(ctx)
```
