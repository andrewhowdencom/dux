package file

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPatchTool_Execute(t *testing.T) {
	tempDir := t.TempDir()
	tool := NewPatch()

	testPath := filepath.Join(tempDir, "patch_test.txt")
	initialContent := "line 1\nline 2\nline exactly the same\nline 4\nline exactly the same\nline 6\n"
	err := os.WriteFile(testPath, []byte(initialContent), 0644)
	if err != nil {
		t.Fatalf("failed to setup test file: %v", err)
	}

	tests := []struct {
		name          string
		args          map[string]interface{}
		wantError     bool
		wantErrSubstr string
		wantFile      string
		wantContent   string
	}{
		{
			name: "missing original_snippet",
			args: map[string]interface{}{
				"path":                testPath,
				"replacement_snippet": "new",
			},
			wantError:     true,
			wantErrSubstr: "missing required argument 'original_snippet'",
		},
		{
			name: "successful single replace",
			args: map[string]interface{}{
				"path":                testPath,
				"original_snippet":    "line 2\n",
				"replacement_snippet": "new line 2\n",
			},
			wantError:   false,
			wantFile:    testPath,
			wantContent: "line 1\nnew line 2\nline exactly the same\nline 4\nline exactly the same\nline 6\n",
		},
		{
			name: "not found original_snippet",
			args: map[string]interface{}{
				"path":                testPath,
				"original_snippet":    "line does not exist",
				"replacement_snippet": "new",
			},
			wantError:     true,
			wantErrSubstr: "was not found in",
		},
		{
			name: "successful replace with literal escaped newline",
			args: map[string]interface{}{
				"path":                testPath,
				"original_snippet":    "line 4\\n",
				"replacement_snippet": "new line 4\\n",
			},
			wantError:   false,
			wantFile:    testPath,
			wantContent: "line 1\nline 2\nline exactly the same\nnew line 4\nline exactly the same\nline 6\n",
		},
		{
			name: "ambiguous multiple matches",
			args: map[string]interface{}{
				"path":                testPath,
				"original_snippet":    "line exactly the same\n",
				"replacement_snippet": "changed line\n",
			},
			wantError:     true,
			wantErrSubstr: "found 2 times in",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Reload baseline for successful replace case so it doesn't pollute subsequent tests
			// that assume the file is back to normal (though we only have one successful test right now)
			_ = os.WriteFile(testPath, []byte(initialContent), 0644)

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
				b, err := os.ReadFile(tc.wantFile)
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
