package bash_test

import (
	"context"
	"testing"

	"github.com/andrewhowdencom/dux/pkg/llm/tool/bash"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBashTool_Name(t *testing.T) {
	tool := bash.New()
	assert.Equal(t, "bash", tool.Name())
}

func TestBashTool_Definition(t *testing.T) {
	tool := bash.New()
	def := tool.Definition()
	
	assert.Equal(t, "bash", def.Name)
	assert.NotEmpty(t, def.Description)
	assert.NotNil(t, def.Parameters)
	
	// Basic validation of schema
	schema := string(def.Parameters)
	assert.Contains(t, schema, `"type": "object"`)
	assert.Contains(t, schema, `"command"`)
}

func TestBashTool_Execute(t *testing.T) {
	tool := bash.New()
	ctx := context.Background()

	t.Run("successful command", func(t *testing.T) {
		args := map[string]interface{}{
			"command": "echo 'hello world'",
		}
		
		result, err := tool.Execute(ctx, args)
		require.NoError(t, err)
		
		resMap, ok := result.(map[string]interface{})
		require.True(t, ok)
		
		assert.Equal(t, "hello world\n", resMap["stdout"])
		assert.Equal(t, "", resMap["stderr"])
		assert.Equal(t, 0, resMap["exit_code"])
	})

	t.Run("command with error", func(t *testing.T) {
		args := map[string]interface{}{
			"command": "ls /nonexistent_path_that_should_fail",
		}
		
		result, err := tool.Execute(ctx, args)
		require.NoError(t, err, "execution of a failing command itself shouldn't return Go error, just the stderr/exit_code result")
		
		resMap, ok := result.(map[string]interface{})
		require.True(t, ok)
		
		assert.Empty(t, resMap["stdout"])
		assert.Contains(t, resMap["stderr"].(string), "No such file or directory")
		assert.NotEqual(t, 0, resMap["exit_code"])
	})

	t.Run("missing parameter", func(t *testing.T) {
		args := map[string]interface{}{}
		
		_, err := tool.Execute(ctx, args)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing required argument")
	})

	t.Run("invalid parameter type", func(t *testing.T) {
		args := map[string]interface{}{
			"command": 123,
		}
		
		_, err := tool.Execute(ctx, args)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "must be a string")
	})
}
