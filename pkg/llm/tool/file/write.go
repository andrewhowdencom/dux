package file

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/andrewhowdencom/dux/pkg/llm"
)

// WriteTool allows the agent to create new files or completely overwrite existing ones.
type WriteTool struct{}

// NewWrite returns a new instance of the file_write tool.
func NewWrite() llm.Tool {
	return &WriteTool{}
}

func (t *WriteTool) Name() string { return "file_write" }

func (t *WriteTool) Definition() llm.ToolDefinitionPart {
	return llm.ToolDefinitionPart{
		Name:        t.Name(),
		Description: "Creates a new file or completely overwrites an existing file with the provided content. Parent directories will be automatically created.",
		Parameters: json.RawMessage(`{
			"type": "object",
			"properties": {
				"path": {
					"type": "string",
					"description": "Absolute or relative path to the file to write to."
				},
				"content": {
					"type": "string",
					"description": "The exact string content to write to the file."
				}
			},
			"required": ["path", "content"]
		}`),
	}
}

func (t *WriteTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	pathInterface, ok := args["path"]
	if !ok {
		return nil, fmt.Errorf("missing required argument 'path'")
	}

	path, ok := pathInterface.(string)
	if !ok {
		return nil, fmt.Errorf("argument 'path' must be a string")
	}

	contentInterface, ok := args["content"]
	if !ok {
		return nil, fmt.Errorf("missing required argument 'content'")
	}

	content, ok := contentInterface.(string)
	if !ok {
		return nil, fmt.Errorf("argument 'content' must be a string")
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create parent directories for %q: %w", path, err)
	}

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return nil, fmt.Errorf("failed to write to file %q: %w", path, err)
	}

	return fmt.Sprintf("Successfully wrote %d bytes to %s", len(content), path), nil
}
