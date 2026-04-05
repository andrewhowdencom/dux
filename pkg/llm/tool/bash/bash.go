package bash

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/andrewhowdencom/dux/pkg/llm"
)

// BashTool allows the agent to execute arbitrary bash commands.
type BashTool struct{}

// New returns a fresh instance of the bash tool.
func New() llm.Tool {
	return &BashTool{}
}

func (t *BashTool) Name() string { return "bash" }

func (t *BashTool) Definition() llm.ToolDefinitionPart {
	return llm.ToolDefinitionPart{
		Name:        t.Name(),
		Description: "Executes an arbitrary bash command and returns its standard output and standard error.\n\n### Examples\n\n**Example 1: List directory contents**\n```json\n{\n  \"command\": \"ls -la\"\n}\n```\n\n**Example 2: Check current working directory**\n```json\n{\n  \"command\": \"pwd\"\n}\n```",
		Parameters: json.RawMessage(`{
			"type": "object",
			"properties": {
				"command": {
					"type": "string",
					"description": "The bash command to execute."
				}
			},
			"required": ["command"]
		}`),
	}
}

func (t *BashTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	cmdInterface, ok := args["command"]
	if !ok {
		return nil, fmt.Errorf("missing required argument 'command'")
	}

	cmdStr, ok := cmdInterface.(string)
	if !ok {
		return nil, fmt.Errorf("argument 'command' must be a string")
	}

	cmd := exec.CommandContext(ctx, "bash", "-c", cmdStr)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return nil, fmt.Errorf("failed to run bash command: %w", err)
		}
	}

	return map[string]interface{}{
		"stdout":    stdout.String(),
		"stderr":    stderr.String(),
		"exit_code": exitCode,
	}, nil
}
