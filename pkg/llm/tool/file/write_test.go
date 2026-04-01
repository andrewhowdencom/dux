package file

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteTool_Execute(t *testing.T) {
	tempDir := t.TempDir()
	tool := NewWrite()

	tests := []struct {
		name          string
		args          map[string]interface{}
		wantError     bool
		wantErrSubstr string
		wantFile      string
		wantContent   string
	}{
		{
			name: "missing path",
			args: map[string]interface{}{
				"content": "hello",
			},
			wantError:     true,
			wantErrSubstr: "missing required argument 'path'",
		},
		{
			name: "missing content",
			args: map[string]interface{}{
				"path": filepath.Join(tempDir, "test.txt"),
			},
			wantError:     true,
			wantErrSubstr: "missing required argument 'content'",
		},
		{
			name: "write new file",
			args: map[string]interface{}{
				"path":    filepath.Join(tempDir, "new.txt"),
				"content": "hello world",
			},
			wantError:   false,
			wantFile:    "new.txt",
			wantContent: "hello world",
		},
		{
			name: "write file in deep dir",
			args: map[string]interface{}{
				"path":    filepath.Join(tempDir, "deep", "nested", "dir", "test.txt"),
				"content": "deep nest",
			},
			wantError:   false,
			wantFile:    filepath.Join("deep", "nested", "dir", "test.txt"),
			wantContent: "deep nest",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := tool.Execute(context.TODO(), tc.args)
			if (err != nil) != tc.wantError {
				t.Fatalf("expected error: %v, got %v", tc.wantError, err)
			}
			if tc.wantError && tc.wantErrSubstr != "" {
				if !strings.Contains(err.Error(), tc.wantErrSubstr) {
					t.Errorf("expected error to contain %q, but got %v", tc.wantErrSubstr, err)
				}
			}
			if !tc.wantError && tc.wantFile != "" {
				absPath := filepath.Join(tempDir, tc.wantFile)
				b, err := os.ReadFile(absPath)
				if err != nil {
					t.Fatalf("failed to read expected file: %v", err)
				}
				if string(b) != tc.wantContent {
					t.Errorf("expected content %q, got %q", tc.wantContent, string(b))
				}
			}
		})
	}
}
