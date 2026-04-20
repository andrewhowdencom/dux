package semantic_test

import (
	"context"
	"testing"

	"github.com/andrewhowdencom/dux/pkg/llm/tool/semantic"
	semmem "github.com/andrewhowdencom/dux/pkg/memory/semantic"
	"github.com/andrewhowdencom/dux/pkg/memory/semantic/memory"
)

func TestWriteTripleTool(t *testing.T) {
	store := memory.NewStore()
	service := semmem.NewService(store)
	tool := semantic.NewWriteTripleTool(service)

	tests := []struct {
		name        string
		args        map[string]interface{}
		expectError bool
	}{
		{
			name: "valid triple",
			args: map[string]interface{}{
				"entity":    "user",
				"attribute": "theme",
				"value":     "dark",
			},
			expectError: false,
		},
		{
			name: "valid triple with constraints",
			args: map[string]interface{}{
				"entity":      "user",
				"attribute":   "language",
				"value":       "go",
				"constraints": map[string]interface{}{"project": "dux"},
			},
			expectError: false,
		},
		{
			name: "missing entity",
			args: map[string]interface{}{
				"attribute": "theme",
				"value":     "dark",
			},
			expectError: true,
		},
		{
			name: "missing attribute",
			args: map[string]interface{}{
				"entity": "user",
				"value":  "dark",
			},
			expectError: true,
		},
		{
			name: "missing value",
			args: map[string]interface{}{
				"entity":    "user",
				"attribute": "theme",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tool.Execute(context.Background(), tt.args)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			resultMap, ok := result.(map[string]string)
			if !ok {
				t.Fatalf("expected map[string]string result, got %T", result)
			}

			if resultMap["status"] != "saved" {
				t.Errorf("expected status 'saved', got %s", resultMap["status"])
			}
			if resultMap["id"] == "" {
				t.Error("expected non-empty id")
			}
		})
	}
}

func TestWriteStatementTool(t *testing.T) {
	store := memory.NewStore()
	service := semmem.NewService(store)
	tool := semantic.NewWriteStatementTool(service)

	tests := []struct {
		name        string
		args        map[string]interface{}
		expectError bool
	}{
		{
			name: "valid statement",
			args: map[string]interface{}{
				"statement": "The project uses Go programming language",
			},
			expectError: false,
		},
		{
			name: "valid statement with constraints",
			args: map[string]interface{}{
				"statement":   "Production deployment is on AWS",
				"constraints": map[string]interface{}{"env": "production"},
			},
			expectError: false,
		},
		{
			name:        "missing statement",
			args:        map[string]interface{}{},
			expectError: true,
		},
		{
			name: "empty statement",
			args: map[string]interface{}{
				"statement": "",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tool.Execute(context.Background(), tt.args)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			resultMap, ok := result.(map[string]string)
			if !ok {
				t.Fatalf("expected map[string]string result, got %T", result)
			}

			if resultMap["status"] != "saved" {
				t.Errorf("expected status 'saved', got %s", resultMap["status"])
			}
			if resultMap["id"] == "" {
				t.Error("expected non-empty id")
			}
		})
	}
}

func TestReadTool(t *testing.T) {
	store := memory.NewStore()
	service := semmem.NewService(store)
	tool := semantic.NewReadTool(service)

	t.Run("read existing fact", func(t *testing.T) {
		_ = store.WriteTriple(context.Background(), semmem.TripleFact{
			ID:        "test-read-1",
			Entity:    "user",
			Attribute: "theme",
			Value:     "dark",
			Metadata:  semmem.FactMetadata{},
		})

		result, err := tool.Execute(context.Background(), map[string]interface{}{
			"id": "test-read-1",
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		fact, ok := result.(semmem.TripleFact)
		if !ok {
			t.Fatalf("expected TripleFact result, got %T", result)
		}

		if fact.Entity != "user" || fact.Attribute != "theme" || fact.Value != "dark" {
			t.Errorf("unexpected fact values: %+v", fact)
		}
	})

	t.Run("read non-existent fact", func(t *testing.T) {
		_, err := tool.Execute(context.Background(), map[string]interface{}{
			"id": "non-existent",
		})

		if err == nil {
			t.Error("expected error for non-existent fact, got nil")
		}
	})

	t.Run("missing id", func(t *testing.T) {
		_, err := tool.Execute(context.Background(), map[string]interface{}{})

		if err == nil {
			t.Error("expected error for missing id, got nil")
		}
	})
}

func TestSearchTool(t *testing.T) {
	store := memory.NewStore()
	service := semmem.NewService(store)
	tool := semantic.NewSearchTool(service)

	_ = store.WriteTriple(context.Background(), semmem.TripleFact{
		ID:        "search-1",
		Entity:    "user",
		Attribute: "theme",
		Value:     "dark",
		Metadata:  semmem.FactMetadata{},
	})

	_ = store.WriteTriple(context.Background(), semmem.TripleFact{
		ID:        "search-2",
		Entity:    "user",
		Attribute: "language",
		Value:     "go",
		Metadata:  semmem.FactMetadata{},
	})

	t.Run("search by entity", func(t *testing.T) {
		result, err := tool.Execute(context.Background(), map[string]interface{}{
			"entity": "user",
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		resultMap, ok := result.(map[string]interface{})
		if !ok {
			t.Fatalf("expected map[string]interface{} result, got %T", result)
		}

		count, ok := resultMap["count"].(int)
		if !ok {
			t.Fatalf("expected count to be int, got %T", resultMap["count"])
		}

		if count != 2 {
			t.Errorf("expected 2 facts, got %d", count)
		}
	})

	t.Run("search with constraints", func(t *testing.T) {
		_ = store.WriteTriple(context.Background(), semmem.TripleFact{
			ID:        "search-3",
			Entity:    "user",
			Attribute: "pref",
			Value:     "a",
			Metadata: semmem.FactMetadata{
				Constraints: map[string]string{"env": "prod"},
			},
		})

		result, err := tool.Execute(context.Background(), map[string]interface{}{
			"constraints": map[string]interface{}{"env": "prod"},
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		resultMap, ok := result.(map[string]interface{})
		if !ok {
			t.Fatalf("expected map[string]interface{} result, got %T", result)
		}

		count, ok := resultMap["count"].(int)
		if !ok {
			t.Fatalf("expected count to be int, got %T", resultMap["count"])
		}

		if count != 1 {
			t.Errorf("expected 1 fact with env=prod, got %d", count)
		}
	})
}

func TestDeleteTool(t *testing.T) {
	store := memory.NewStore()
	service := semmem.NewService(store)
	tool := semantic.NewDeleteTool(service)

	_ = store.WriteTriple(context.Background(), semmem.TripleFact{
		ID:        "delete-1",
		Entity:    "user",
		Attribute: "theme",
		Value:     "dark",
		Metadata:  semmem.FactMetadata{},
	})

	t.Run("delete existing fact", func(t *testing.T) {
		result, err := tool.Execute(context.Background(), map[string]interface{}{
			"id": "delete-1",
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		resultMap, ok := result.(map[string]string)
		if !ok {
			t.Fatalf("expected map[string]string result, got %T", result)
		}

		if resultMap["status"] != "deleted" {
			t.Errorf("expected status 'deleted', got %s", resultMap["status"])
		}

		_, err = store.ReadTriple(context.Background(), "delete-1")
		if err != semmem.ErrNotFound {
			t.Errorf("expected fact to be deleted, got %v", err)
		}
	})

	t.Run("delete non-existent fact", func(t *testing.T) {
		_, err := tool.Execute(context.Background(), map[string]interface{}{
			"id": "non-existent",
		})

		if err != nil {
			t.Errorf("expected no error for deleting non-existent fact, got %v", err)
		}
	})

	t.Run("missing id", func(t *testing.T) {
		_, err := tool.Execute(context.Background(), map[string]interface{}{})

		if err == nil {
			t.Error("expected error for missing id, got nil")
		}
	})
}

func TestValidateTool(t *testing.T) {
	store := memory.NewStore()
	service := semmem.NewService(store)
	tool := semantic.NewValidateTool(service)

	_ = store.WriteTriple(context.Background(), semmem.TripleFact{
		ID:        "validate-1",
		Entity:    "user",
		Attribute: "theme",
		Value:     "dark",
		Metadata:  semmem.FactMetadata{},
	})

	t.Run("validate existing fact", func(t *testing.T) {
		result, err := tool.Execute(context.Background(), map[string]interface{}{
			"id": "validate-1",
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		resultMap, ok := result.(map[string]string)
		if !ok {
			t.Fatalf("expected map[string]string result, got %T", result)
		}

		if resultMap["status"] != "validated" {
			t.Errorf("expected status 'validated', got %s", resultMap["status"])
		}
	})

	t.Run("validate non-existent fact", func(t *testing.T) {
		_, err := tool.Execute(context.Background(), map[string]interface{}{
			"id": "non-existent",
		})

		if err == nil {
			t.Error("expected error for non-existent fact, got nil")
		}
	})

	t.Run("missing id", func(t *testing.T) {
		_, err := tool.Execute(context.Background(), map[string]interface{}{})

		if err == nil {
			t.Error("expected error for missing id, got nil")
		}
	})
}
