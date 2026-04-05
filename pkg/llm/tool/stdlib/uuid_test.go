package stdlib

import (
	"context"
	"testing"
)

func TestUUIDTool_Execute(t *testing.T) {
	tool := NewUUID()
	ctx := context.Background()

	res, err := tool.Execute(ctx, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	resMap, ok := res.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map[string]interface{}")
	}

	id, ok := resMap["uuid"].(string)
	if !ok || len(id) != 36 {
		t.Fatalf("expected valid UUID string, got: %v", id)
	}
}
