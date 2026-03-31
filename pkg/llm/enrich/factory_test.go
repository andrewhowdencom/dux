package enrich

import (
	"context"
	"testing"

	"github.com/andrewhowdencom/dux/internal/config"
)

func TestNewFromConfig(t *testing.T) {
	cfgs := []config.Enricher{
		{Type: "time"},
		{Type: "os"},
		{Type: "prompt", Text: "Do not rm -rf /."},
	}

	enrichers, err := NewFromConfig(cfgs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(enrichers) != 3 {
		t.Fatalf("expected 3 enrichers, got %d", len(enrichers))
	}

	if enrichers[0].Type() != "time" {
		t.Errorf("expected type time, got %s", enrichers[0].Type())
	}

	if enrichers[1].Type() != "os" {
		t.Errorf("expected type os, got %s", enrichers[1].Type())
	}

	if enrichers[2].Type() != "prompt" {
		t.Errorf("expected type prompt, got %s", enrichers[2].Type())
	}

	// Test Enrich execution on prompt
	ctx := context.Background()
	res, err := enrichers[2].Enrich(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedContent := "Do not rm -rf /."
	if len(res) < len(expectedContent) {
		t.Errorf("prompt enricher did not return expected content. Output: %s", res)
	}
}

func TestNewFromConfigUnknownType(t *testing.T) {
	cfgs := []config.Enricher{
		{Type: "invalid_mystery_type"},
	}

	_, err := NewFromConfig(cfgs)
	if err == nil {
		t.Fatalf("expected error for invalid type, got nil")
	}
}
