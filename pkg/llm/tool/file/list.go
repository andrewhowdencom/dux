package file

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/andrewhowdencom/dux/pkg/llm"
)

// ListTool allows the agent to list files and directories.
type ListTool struct{}

// NewList returns a new instance of the file_list tool.
func NewList() llm.Tool {
	return &ListTool{}
}

func (t *ListTool) Name() string { return "file_list" }

func (t *ListTool) Definition() llm.ToolDefinitionPart {
	return llm.ToolDefinitionPart{
		Name:        t.Name(),
		Description: "Lists files and directories at the specified path. Limits output to a maximum number of files to prevent context cutoff. Silently skips hidden files (starting with '.') by default unless include_hidden is set to true.",
		Parameters: json.RawMessage(`{
			"type": "object",
			"properties": {
				"path": {
					"type": "string",
					"description": "Absolute or relative path to the directory to list."
				},
				"include_hidden": {
					"type": "boolean",
					"description": "If true, hidden directories and files (like .git or .env) will be traversed and listed. Defaults to false."
				}
			},
			"required": ["path"]
		}`),
	}
}

func (t *ListTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	pathInterface, ok := args["path"]
	if !ok {
		return nil, fmt.Errorf("missing required argument 'path'")
	}

	path, ok := pathInterface.(string)
	if !ok {
		return nil, fmt.Errorf("argument 'path' must be a string")
	}

	includeHidden := false
	if ih, ok := args["include_hidden"].(bool); ok {
		includeHidden = ih
	}

	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to stat path %q: %w", path, err)
	}

	if !info.IsDir() {
		return []string{path}, nil
	}

	var results []string
	const maxFiles = 1000

	err = filepath.WalkDir(path, func(walkPath string, d fs.DirEntry, err error) error {
		if err != nil {
			// Skip files/dirs we can't read rather than failing the whole walk
			return nil
		}

		// Prevent returning the root dir itself in the list
		if walkPath == path {
			return nil
		}

		base := filepath.Base(walkPath)
		if !includeHidden && strings.HasPrefix(base, ".") {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if len(results) >= maxFiles {
			// Stop traversal once we hit the capacity
			return fmt.Errorf("limit reached")
		}

		// Append trailing slash to dirs for clarity
		if d.IsDir() {
			results = append(results, walkPath+string(os.PathSeparator))
		} else {
			results = append(results, walkPath)
		}

		return nil
	})

	if err != nil && err.Error() != "limit reached" {
		return nil, fmt.Errorf("failed to traverse directory %q: %w", path, err)
	}

	if len(results) >= maxFiles {
		results = append(results, fmt.Sprintf("... (truncated after %d items) ...", maxFiles))
	}

	return results, nil
}
