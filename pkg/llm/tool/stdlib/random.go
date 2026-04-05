package stdlib

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"github.com/andrewhowdencom/dux/pkg/llm"
)

type RandomTool struct {
	rng *rand.Rand
}

func NewRandom() llm.Tool {
	return &RandomTool{
		rng: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (t *RandomTool) Name() string { return "generate_random_number" }

func (t *RandomTool) Definition() llm.ToolDefinitionPart {
	return llm.ToolDefinitionPart{
		Name:        t.Name(),
		Description: "Produces a random integer between the min (inclusive) and max (exclusive) ranges. Use this rather than hallucinating your own random numbers.",
		Parameters: json.RawMessage(`{"type":"object","properties":{"min":{"type":"integer","description":"The minimum value, inclusive."},"max":{"type":"integer","description":"The maximum value, exclusive. Must be higher than min."}},"required":["min","max"]}`),
	}
}

func (t *RandomTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	minVal, okMin := args["min"].(float64)
	maxVal, okMax := args["max"].(float64)

	if !okMin || !okMax {
		return nil, fmt.Errorf("missing or invalid 'min' or 'max' parameter")
	}

	minInt := int(minVal)
	maxInt := int(maxVal)

	if minInt >= maxInt {
		return nil, fmt.Errorf("'min' must be strictly less than 'max'")
	}

	result := minInt + t.rng.Intn(maxInt-minInt)

	return map[string]interface{}{
		"result": result,
	}, nil
}
