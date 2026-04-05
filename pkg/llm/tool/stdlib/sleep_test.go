package stdlib

import (
	"context"
	"testing"
	"time"
)

func TestSleepTool_Execute(t *testing.T) {
	tool := NewSleep()
	ctx := context.Background()

	start := time.Now()
	res, err := tool.Execute(ctx, map[string]interface{}{"milliseconds": float64(50)})
	
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	
	elapsed := time.Since(start)
	if elapsed < 50*time.Millisecond {
		t.Fatalf("sleep tool woke up too early, elapsed: %v", elapsed)
	}

	resMap, ok := res.(map[string]interface{})
	if !ok || resMap["status"] != "slept" {
		t.Fatalf("expected 'slept' status, got %v", res)
	}
}
