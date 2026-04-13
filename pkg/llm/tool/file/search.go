package file

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/andrewhowdencom/dux/pkg/llm"
)

// SearchTool allows the agent to search for text or regex in a directory or file.
type SearchTool struct{}

// NewSearch returns a new instance of the file_search tool.
func NewSearch() llm.Tool {
	return &SearchTool{}
}

func (t *SearchTool) Name() string { return "file_search" }

func (t *SearchTool) Definition() llm.ToolDefinitionPart {
	return llm.ToolDefinitionPart{
		Name:        t.Name(),
		Description: "Searches for a string or regular expression within a specific file or recursively through a directory. Silently skips hidden files and directories (starting with '.') by default, as well as binary files. Use this instead of running grep in bash.\n\n### Examples\n\n**Example 1: Search for a simple string in a directory**\n```json\n{\n  \"path\": \".\",\n  \"query\": \"TODO(:\"\n}\n```\n\n**Example 2: Search for a regex pattern**\n```json\n{\n  \"path\": \"src/\",\n  \"query\": \"func\\\\s+main\\\\(\",\n  \"is_regex\": true\n}\n```",
		Parameters: json.RawMessage(`{
			"type": "object",
			"properties": {
				"path": {
					"type": "string",
					"description": "Absolute or relative path to the directory or file to search in."
				},
				"query": {
					"type": "string",
					"description": "The exact string or regular expression pattern to search for."
				},
				"is_regex": {
					"type": "boolean",
					"description": "If true, treats the query as a regular expression. Defaults to false (exact string match)."
				},
				"include_hidden": {
					"type": "boolean",
					"description": "If true, hidden directories and files (like .git or .env) will be searched. Defaults to false."
				}
			},
			"required": ["path", "query"]
		}`),
	}
}

func (t *SearchTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	pathInterface, ok := args["path"]
	if !ok {
		return nil, fmt.Errorf("missing required argument 'path'")
	}
	path, ok := pathInterface.(string)
	if !ok {
		return nil, fmt.Errorf("argument 'path' must be a string")
	}

	queryInterface, ok := args["query"]
	if !ok {
		return nil, fmt.Errorf("missing required argument 'query'")
	}
	query, ok := queryInterface.(string)
	if !ok {
		return nil, fmt.Errorf("argument 'query' must be a string")
	}

	isRegex := false
	if ir, ok := args["is_regex"].(bool); ok {
		isRegex = ir
	}

	includeHidden := false
	if ih, ok := args["include_hidden"].(bool); ok {
		includeHidden = ih
	}

	var re *regexp.Regexp
	if isRegex {
		var err error
		re, err = regexp.Compile(query)
		if err != nil {
			return nil, fmt.Errorf("invalid regular expression: %w", err)
		}
	}

	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to stat path %q: %w", path, err)
	}

	var results []string
	const maxMatches = 1000 // Arbitrary limit to prevent context explosion

	searchFile := func(filePath string) error {
		file, err := os.Open(filePath)
		if err != nil {
			return nil // Skip files we cannot read
		}
		defer func() { _ = file.Close() }()

		// Heuristic binary detection: Read first 1024 bytes and check for null chars.
		head := make([]byte, 1024)
		n, err := file.Read(head)
		if err != nil && err != io.EOF {
			return nil // Skip unreadable headers
		}
		if bytes.Contains(head[:n], []byte{0}) {
			return nil // Skip binary files
		}

		if _, err := file.Seek(0, 0); err != nil {
			return nil
		}

		scanner := bufio.NewScanner(file)
		const maxCapacity = 1024 * 1024 // 1MB per line max
		buf := make([]byte, maxCapacity)
		scanner.Buffer(buf, maxCapacity)

		lineNumber := 1
		for scanner.Scan() {
			if len(results) >= maxMatches {
				return fmt.Errorf("limit reached")
			}
			line := scanner.Text()
			match := false

			if isRegex {
				match = re.MatchString(line)
			} else {
				match = strings.Contains(line, query)
			}

			if match {
				results = append(results, fmt.Sprintf("%s:%d: %s", filePath, lineNumber, line))
			}
			lineNumber++
		}
		return nil
	}

	if !info.IsDir() {
		// Single file search
		err = searchFile(path)
		if err != nil && err.Error() != "limit reached" {
			return nil, fmt.Errorf("error searching file %q: %w", path, err)
		}
	} else {
		// Directory walk
		err = filepath.WalkDir(path, func(walkPath string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}

			base := filepath.Base(walkPath)
			if !includeHidden && strings.HasPrefix(base, ".") {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}

			if !d.IsDir() {
				searchErr := searchFile(walkPath)
				if searchErr != nil {
					return searchErr
				}
			}

			return nil
		})

		if err != nil && err.Error() != "limit reached" {
			return nil, fmt.Errorf("failed to traverse directory %q: %w", path, err)
		}
	}

	if len(results) == 0 {
		return "No matches found.", nil
	}

	if len(results) >= maxMatches {
		results = append(results, fmt.Sprintf("... (truncated after %d matches) ...", maxMatches))
	}

	return strings.Join(results, "\n"), nil
}
