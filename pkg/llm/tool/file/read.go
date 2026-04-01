package file

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/andrewhowdencom/dux/pkg/llm"
)

const maxReadLines = 800

// ReadTool allows the agent to read the contents of a file up to a line limit.
type ReadTool struct{}

// NewRead returns a new instance of the file_read tool.
func NewRead() llm.Tool {
	return &ReadTool{}
}

func (t *ReadTool) Name() string { return "file_read" }

func (t *ReadTool) Definition() llm.ToolDefinitionPart {
	return llm.ToolDefinitionPart{
		Name:        t.Name(),
		Description: "Reads the contents of a file. By default reads up to 800 lines. Use start_line and end_line for pagination on larger files. Will refuse to read binary files.",
		Parameters: json.RawMessage(`{
			"type": "object",
			"properties": {
				"path": {
					"type": "string",
					"description": "Absolute or relative path to the file to read."
				},
				"start_line": {
					"type": "integer",
					"description": "The line number to start reading from (1-indexed). Inclusive."
				},
				"end_line": {
					"type": "integer",
					"description": "The line number to end reading at (1-indexed). Inclusive."
				}
			},
			"required": ["path"]
		}`),
	}
}

func (t *ReadTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	pathInterface, ok := args["path"]
	if !ok {
		return nil, fmt.Errorf("missing required argument 'path'")
	}

	path, ok := pathInterface.(string)
	if !ok {
		return nil, fmt.Errorf("argument 'path' must be a string")
	}

	startLine := 1
	if sl, ok := args["start_line"].(float64); ok && sl > 0 {
		startLine = int(sl)
	}

	endLine := startLine + maxReadLines - 1
	if el, ok := args["end_line"].(float64); ok && el > 0 {
		endLineInt := int(el)
		if endLineInt >= startLine {
			// Cap the pagination to maxReadLines to prevent context blowing out, even if requested.
			if endLineInt-startLine+1 > maxReadLines {
				endLine = startLine + maxReadLines - 1
			} else {
				endLine = endLineInt
			}
		} else {
			return nil, fmt.Errorf("end_line cannot be less than start_line")
		}
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %q: %w", path, err)
	}
	defer func() { _ = file.Close() }()

	// Heuristic binary detection: Read first 1024 bytes and check for null chars.
	head := make([]byte, 1024)
	n, err := file.Read(head)
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("failed to read file %q header: %w", path, err)
	}
	if bytes.Contains(head[:n], []byte{0}) {
		return nil, fmt.Errorf("cannot read %q: appears to be a binary file", path)
	}

	// Rewind after reading header
	if _, err := file.Seek(0, 0); err != nil {
		return nil, fmt.Errorf("failed to seek file %q: %w", path, err)
	}

	scanner := bufio.NewScanner(file)
	// Some lines of code can be exceptionally long, allocate a custom initial buffer and set a high max to avoid token.
	const maxCapacity = 1024 * 1024 // 1MB per line max
	buf := make([]byte, maxCapacity)
	scanner.Buffer(buf, maxCapacity)

	currentLine := 1
	var outBuf bytes.Buffer
	outBuf.WriteString(fmt.Sprintf("--- File: %s ---\n", path))

	truncated := false
	for scanner.Scan() {
		if currentLine > endLine {
			truncated = true
			break
		}
		if currentLine >= startLine {
			outBuf.WriteString(fmt.Sprintf("%d: %s\n", currentLine, scanner.Text()))
		}
		currentLine++
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file %q at line %d: %w", path, currentLine, err)
	}

	if truncated {
		outBuf.WriteString(fmt.Sprintf("--- (Truncated after %d lines) ---\n", endLine))
	} else if currentLine == 1 {
		outBuf.WriteString("--- (Empty file) ---\n")
	}

	return outBuf.String(), nil
}
