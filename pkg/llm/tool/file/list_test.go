package file

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestListTool_Execute(t *testing.T) {
	tempDir := t.TempDir()
	tool := NewList()

	// Setup a directory structure
	// tempDir/
	//   public/
	//      file1.txt
	//   .hidden/
	//      secret.txt
	//   node_modules/
	//      lib.js
	//   .gitignore
	if err := os.MkdirAll(filepath.Join(tempDir, "public"), 0755); err != nil {
		t.Fatalf("setup failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "public", "file1.txt"), []byte("data"), 0644); err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(tempDir, ".hidden"), 0755); err != nil {
		t.Fatalf("setup failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, ".hidden", "secret.txt"), []byte("secret"), 0644); err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(tempDir, "node_modules"), 0755); err != nil {
		t.Fatalf("setup failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "node_modules", "lib.js"), []byte("js"), 0644); err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	if err := os.WriteFile(filepath.Join(tempDir, ".gitignore"), []byte("node_modules"), 0644); err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	tests := []struct {
		name          string
		args          map[string]interface{}
		wantError     bool
		wantErrSubstr string
		wantPaths     []string
		notWantPaths  []string
	}{
		{
			name: "missing path",
			args: map[string]interface{}{},
			wantError:     true,
			wantErrSubstr: "missing required argument 'path'",
		},
		{
			name: "list omitting hidden by default",
			args: map[string]interface{}{
				"path": tempDir,
			},
			wantError: false,
			wantPaths: []string{
				filepath.Join(tempDir, "public") + string(os.PathSeparator),
				filepath.Join(tempDir, "public", "file1.txt"),
				filepath.Join(tempDir, "node_modules") + string(os.PathSeparator),
				filepath.Join(tempDir, "node_modules", "lib.js"),
			},
			notWantPaths: []string{
				filepath.Join(tempDir, ".hidden") + string(os.PathSeparator),
				filepath.Join(tempDir, ".hidden", "secret.txt"),
				filepath.Join(tempDir, ".gitignore"),
			},
		},
		{
			name: "list including hidden",
			args: map[string]interface{}{
				"path":           tempDir,
				"include_hidden": true,
			},
			wantError: false,
			wantPaths: []string{
				filepath.Join(tempDir, "public") + string(os.PathSeparator),
				filepath.Join(tempDir, "public", "file1.txt"),
				filepath.Join(tempDir, "node_modules") + string(os.PathSeparator),
				filepath.Join(tempDir, "node_modules", "lib.js"),
				filepath.Join(tempDir, ".hidden") + string(os.PathSeparator),
				filepath.Join(tempDir, ".hidden", "secret.txt"),
				filepath.Join(tempDir, ".gitignore"),
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			res, err := tool.Execute(context.TODO(), tc.args)
			if (err != nil) != tc.wantError {
				t.Fatalf("expected error: %v, got %v", tc.wantError, err)
			}
			if tc.wantError && tc.wantErrSubstr != "" {
				if err.Error() == "" || !contains(err.Error(), tc.wantErrSubstr) {
					t.Errorf("expected error to contain %q, but got %v", tc.wantErrSubstr, err)
				}
			}
			if !tc.wantError {
				paths, ok := res.([]string)
				if !ok {
					t.Fatalf("expected []string result, got %T", res)
				}

				for _, want := range tc.wantPaths {
					if !containsList(paths, want) {
						t.Errorf("expected paths to contain %q, but it didn't. Got: %v", want, paths)
					}
				}
				for _, notWant := range tc.notWantPaths {
					if containsList(paths, notWant) {
						t.Errorf("expected paths to NOT contain %q, but it did", notWant)
					}
				}
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr || stringsContains(s, substr)
}

func stringsContains(s, substr string) bool {
	return strings.Contains(s, substr)
}

// real helper
func containsList(list []string, item string) bool {
	for _, l := range list {
		if l == item {
			return true
		}
	}
	return false
}
