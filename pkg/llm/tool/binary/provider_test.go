package binary

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/andrewhowdencom/dux/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBinaryProvider(t *testing.T) {
	t.Run("Definition encodes schema correctly", func(t *testing.T) {
		bt := config.BinaryTool{
			Executable: "echo",
			Inputs: map[string]config.ToolInput{
				"message": {
					Type:        "string",
					Description: "Echo message",
					Required:    true,
				},
			},
		}

		provider := NewProvider("echo_tool", &bt)
		assert.Equal(t, "echo_tool", provider.Namespace())

		tool, ok := provider.GetTool("echo_tool")
		require.True(t, ok)

		def := tool.Definition()
		assert.Equal(t, "echo_tool", def.Name)
		
		var schema map[string]interface{}
		err := json.Unmarshal(def.Parameters, &schema)
		require.NoError(t, err)

		assert.Equal(t, "object", schema["type"])
		props := schema["properties"].(map[string]interface{})
		
		msgProp := props["message"].(map[string]interface{})
		assert.Equal(t, "string", msgProp["type"])
		assert.Equal(t, "Echo message", msgProp["description"])

		reqs := schema["required"].([]interface{})
		assert.Contains(t, reqs, "message")
	})

	t.Run("Execute substitutes args correctly", func(t *testing.T) {
		bt := config.BinaryTool{
			Executable: "echo",
			Args:       []string{"-n", "Hello {name}"},
		}

		provider := NewProvider("echo_tool", &bt)
		tool, _ := provider.GetTool("echo_tool")

		out, err := tool.Execute(context.TODO(), map[string]interface{}{
			"name": "World",
		})

		require.NoError(t, err)
		assert.Equal(t, "Hello World", out) // -n ensures no trailing newline
	})
	
	t.Run("Execute fails nicely on missing executable", func(t *testing.T) {
		bt := config.BinaryTool{
			Executable: "this_executable_does_not_exist_xyz123",
			Args:       []string{},
		}

		provider := NewProvider("bad_tool", &bt)
		tool, _ := provider.GetTool("bad_tool")

		_, err := tool.Execute(context.TODO(), map[string]interface{}{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "executable file not found")
	})
}
