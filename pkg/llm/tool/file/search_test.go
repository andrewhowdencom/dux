package file

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSearchTool_Execute(t *testing.T) {
	tempDir := t.TempDir()

	file1 := filepath.Join(tempDir, "file1.txt")
	_ = os.WriteFile(file1, []byte("Hello world\nThis is a test\nGoodbye world\n"), 0644)

	hiddenFile := filepath.Join(tempDir, ".hidden.txt")
	_ = os.WriteFile(hiddenFile, []byte("Secret data\nHello hidden\n"), 0644)

	binaryFile := filepath.Join(tempDir, "binary.bin")
	_ = os.WriteFile(binaryFile, []byte{0x00, 0x01, 0x02, 0x48, 0x65, 0x6c, 0x6c, 0x6f}, 0644) // Contains "Hello" but is binary

	tool := NewSearch()
	ctx := context.Background()

	t.Run("literal match single file", func(t *testing.T) {
		res, err := tool.Execute(ctx, map[string]interface{}{
			"path":     file1,
			"query":    "world",
			"is_regex": false,
		})
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}

		out := res.(string)
		if !strings.Contains(out, "file1.txt:1: Hello world") {
			t.Errorf("expected line 1 match in output, got: %v", out)
		}
		if !strings.Contains(out, "file1.txt:3: Goodbye world") {
			t.Errorf("expected line 3 match in output, got: %v", out)
		}
	})

	t.Run("regex match dir", func(t *testing.T) {
		res, err := tool.Execute(ctx, map[string]interface{}{
			"path":     tempDir,
			"query":    "^This is",
			"is_regex": true,
		})
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}

		out := res.(string)
		if !strings.Contains(out, "file1.txt:2: This is a test") {
			t.Errorf("expected regex match in output, got: %v", out)
		}
	})

	t.Run("ignore hidden and binary", func(t *testing.T) {
		res, err := tool.Execute(ctx, map[string]interface{}{
			"path":  tempDir,
			"query": "Hello",
		})
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}

		out := res.(string)
		if strings.Contains(out, ".hidden.txt") {
			t.Errorf("should not include hidden file matches: %v", out)
		}
		if strings.Contains(out, "binary.bin") {
			t.Errorf("should not include binary file matches: %v", out)
		}
	})

	t.Run("include hidden", func(t *testing.T) {
		res, err := tool.Execute(ctx, map[string]interface{}{
			"path":           tempDir,
			"query":          "Secret",
			"include_hidden": true,
		})
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}

		out := res.(string)
		if !strings.Contains(out, ".hidden.txt:1: Secret data") {
			t.Errorf("expected hidden file match, got: %v", out)
		}
	})
}
