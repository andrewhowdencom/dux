package gemini

import (
	"encoding/json"
	"testing"

	"github.com/andrewhowdencom/dux/pkg/llm"
	"github.com/stretchr/testify/assert"
)

func TestBuildGeminiRequest_ToolDefinitionParameters(t *testing.T) {
	msg := llm.Message{
		Identity: llm.Identity{Role: "system"},
		Parts: []llm.Part{
			llm.ToolDefinitionPart{
				Name:        "test_tool",
				Description: "A tool to test parameter passing",
				Parameters: json.RawMessage(`{
					"type": "object",
					"properties": {
						"example_param": {
							"type": "string",
							"description": "An example parameter"
						}
					},
					"required": ["example_param"]
				}`),
			},
		},
	}

	_, cfg := buildGeminiRequest([]llm.Message{msg})

	assert.NotNil(t, cfg, "Config should not be nil")
	assert.NotEmpty(t, cfg.Tools, "Tools should not be empty")
	
	tool := cfg.Tools[0]
	assert.NotEmpty(t, tool.FunctionDeclarations, "Function declarations should not be empty")
	
	decl := tool.FunctionDeclarations[0]
	assert.Equal(t, "test_tool", decl.Name)
	assert.Equal(t, "A tool to test parameter passing", decl.Description)
	
	// This is the crucial part that was failing before
	assert.NotNil(t, decl.Parameters, "Parameters schema should not be nil")
	assert.NotNil(t, decl.Parameters.Properties, "Parameters schema should have properties mapped")
	assert.Contains(t, decl.Parameters.Properties, "example_param", "Parameters schema should contain example_param property")
}
