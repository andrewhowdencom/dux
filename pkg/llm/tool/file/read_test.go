package file

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadTool_Execute(t *testing.T) {
	tempDir := t.TempDir()
	tool := NewRead()

	// Setup a standard file
	stdFilePath := filepath.Join(tempDir, "std.txt")
	testContent := "line 1\nline 2\nline 3\nline 4\nline 5\n"
	err := os.WriteFile(stdFilePath, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Setup a binary file
	binFilePath := filepath.Join(tempDir, "bin.dat")
	err = os.WriteFile(binFilePath, []byte{0x00, 0x01, 0x02, 0x03}, 0644)
	if err != nil {
		t.Fatalf("failed to write binary file: %v", err)
	}

	tests := []struct {
		name          string
		args          map[string]interface{}
		wantError     bool
		wantErrSubstr string
		wantSubstr    []string
		notWantSubstr []string
	}{
		{
			name: "missing path",
			args: map[string]interface{}{},
			wantError: true,
			wantErrSubstr: "missing required argument",
		},
		{
			name: "read entire text file",
			args: map[string]interface{}{
				"path": stdFilePath,
			},
			wantError: false,
			wantSubstr: []string{
				"--- File: " + stdFilePath + " ---",
				"1: line 1",
				"5: line 5",
			},
		},
		{
			name: "read with pagination",
			args: map[string]interface{}{
				"path":       stdFilePath,
				"start_line": float64(2),
				"end_line":   float64(4),
			},
			wantError: false,
			wantSubstr: []string{
				"2: line 2",
				"4: line 4",
			},
			notWantSubstr: []string{
				"1: line 1",
				"5: line 5",
			},
		},
		{
			name: "reject binary file",
			args: map[string]interface{}{
				"path": binFilePath,
			},
			wantError: true,
			wantErrSubstr: "binary file",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			res, err := tool.Execute(context.TODO(), tc.args)
			if (err != nil) != tc.wantError {
				t.Fatalf("expected error: %v, got %v", tc.wantError, err)
			}
			if tc.wantError && tc.wantErrSubstr != "" {
				if !strings.Contains(err.Error(), tc.wantErrSubstr) {
					t.Errorf("expected error to contain %q, but got %v", tc.wantErrSubstr, err)
				}
			}
			if err == nil {
				resStr := res.(string)
				for _, sub := range tc.wantSubstr {
					if !strings.Contains(resStr, sub) {
						t.Errorf("expected result to contain %q, but got:\n%s", sub, resStr)
					}
				}
				for _, sub := range tc.notWantSubstr {
					if strings.Contains(resStr, sub) {
						t.Errorf("expected result to NOT contain %q, but got:\n%s", sub, resStr)
					}
				}
			}
		})
	}
}
