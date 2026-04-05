package file

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/andrewhowdencom/dux/pkg/llm"
)

// PatchTool allows the agent to edit a specific snippet of text in a file.
type PatchTool struct{}

// NewPatch returns a new instance of the file_patch tool.
func NewPatch() llm.Tool {
	return &PatchTool{}
}

func (t *PatchTool) Name() string { return "file_patch" }

func (t *PatchTool) Definition() llm.ToolDefinitionPart {
	return llm.ToolDefinitionPart{
		Name:        t.Name(),
		Description: "Edits an existing file by replacing a specific snippet of text. The original_snippet must exactly match the text in the file, including whitespace and line breaks. Will fail if the snippet is not found or is found multiple times.\n\n### Examples\n\n**Example 1: Replace a function call**\n```json\n{\n  \"path\": \"main.go\",\n  \"original_snippet\": \"fmt.Println(\\\"Hello\\\")\",\n  \"replacement_snippet\": \"fmt.Println(\\\"Hello, World!\\\")\"\n}\n```",
		Parameters: json.RawMessage(`{
			"type": "object",
			"properties": {
				"path": {
					"type": "string",
					"description": "Absolute or relative path to the file to edit."
				},
				"original_snippet": {
					"type": "string",
					"description": "The exact text to find and replace. Must match the target file perfectly."
				},
				"replacement_snippet": {
					"type": "string",
					"description": "The new text that will replace original_snippet."
				}
			},
			"required": ["path", "original_snippet", "replacement_snippet"]
		}`),
	}
}

func (t *PatchTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	pathInterface, ok := args["path"]
	if !ok {
		return nil, fmt.Errorf("missing required argument 'path'")
	}

	path, ok := pathInterface.(string)
	if !ok {
		return nil, fmt.Errorf("argument 'path' must be a string")
	}

	origInterface, ok := args["original_snippet"]
	if !ok {
		return nil, fmt.Errorf("missing required argument 'original_snippet'")
	}
	originalSnippet, ok := origInterface.(string)
	if !ok {
		return nil, fmt.Errorf("argument 'original_snippet' must be a string")
	}

	reprInterface, ok := args["replacement_snippet"]
	if !ok {
		return nil, fmt.Errorf("missing required argument 'replacement_snippet'")
	}
	replacementSnippet, ok := reprInterface.(string)
	if !ok {
		return nil, fmt.Errorf("argument 'replacement_snippet' must be a string")
	}

	if originalSnippet == "" {
		return nil, fmt.Errorf("original_snippet cannot be empty")
	}

	contentBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %q: %w", path, err)
	}

	// Normalize CRLF to LF for cross-platform robustness
	content := strings.ReplaceAll(string(contentBytes), "\r\n", "\n")
	origSnippet := strings.ReplaceAll(originalSnippet, "\r\n", "\n")
	replSnippet := strings.ReplaceAll(replacementSnippet, "\r\n", "\n")

	// LLMs often output literal "\n" or "\t" characters when they intend to format actual newlines or tabs in strings.
	// We unescape them here so that exact string matching works gracefully.
	origSnippet = strings.ReplaceAll(origSnippet, "\\n", "\n")
	origSnippet = strings.ReplaceAll(origSnippet, "\\t", "\t")
	replSnippet = strings.ReplaceAll(replSnippet, "\\n", "\n")
	replSnippet = strings.ReplaceAll(replSnippet, "\\t", "\t")

	count := strings.Count(content, origSnippet)
	if count == 0 {
		return nil, fmt.Errorf("failed to apply patch: original_snippet was not found in %q (check whitespace and line breaks)", path)
	}
	if count > 1 {
		return nil, fmt.Errorf("failed to apply patch: original_snippet was found %d times in %q (snippet must be unique to avoid ambiguous replacements)", count, path)
	}

	newContent := strings.Replace(content, origSnippet, replSnippet, 1)

	if err := os.WriteFile(path, []byte(newContent), 0644); err != nil {
		return nil, fmt.Errorf("failed to write patched file %q: %w", path, err)
	}

	return fmt.Sprintf("Successfully patched %q using search-and-replace.", path), nil
}
