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
		{Type: "guard_rail", Text: "Do not mention unicorns."},
	}

	enrichers, err := NewFromConfig(cfgs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(enrichers) != 4 {
		t.Fatalf("expected 4 enrichers, got %d", len(enrichers))
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

	if enrichers[3].Type() != "guard_rail" {
		t.Errorf("expected type guard_rail, got %s", enrichers[3].Type())
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

	// Test Enrich execution on guard_rail
	resGuardRail, err := enrichers[3].Enrich(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedGuardRailContent := "Do not mention unicorns."
	if len(resGuardRail) < len(expectedGuardRailContent) {
		t.Errorf("guard_rail enricher did not return expected content. Output: %s", resGuardRail)
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
