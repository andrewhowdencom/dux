package ui

import (
	"context"
	"strings"
	"testing"

	"github.com/andrewhowdencom/dux/internal/config"
	"github.com/andrewhowdencom/dux/pkg/llm"
)

func TestNewEnrichersFromConfig(t *testing.T) {
	cfgs := []config.Enricher{
		{Type: "time"},
		{Type: "os"},
		{Type: "prompt", Text: "Do not rm -rf /."},
		{Type: "guard_rail", Text: "Do not mention unicorns."},
	}

	enrichers, err := NewEnrichersFromConfig(cfgs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(enrichers) != 4 {
		t.Fatalf("expected 4 enrichers, got %d", len(enrichers))
	}

	// Test Inject execution on prompt
	ctx := context.Background()
	res, err := enrichers[2].Inject(ctx, llm.InjectQuery{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedContent := "Do not rm -rf /."
	if len(res) == 0 || !strings.Contains(res[0].Text(), expectedContent) {
		t.Errorf("prompt enricher did not return expected content. Output: %v", res)
	}

	// Test Inject execution on guard_rail
	resGuardRail, err := enrichers[3].Inject(ctx, llm.InjectQuery{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedGuardRailContent := "Do not mention unicorns."
	if len(resGuardRail) == 0 || !strings.Contains(resGuardRail[0].Text(), expectedGuardRailContent) {
		t.Errorf("guard_rail enricher did not return expected content. Output: %v", resGuardRail)
	}
}

func TestNewEnrichersFromConfigUnknownType(t *testing.T) {
	cfgs := []config.Enricher{
		{Type: "invalid_mystery_type"},
	}

	_, err := NewEnrichersFromConfig(cfgs)
	if err == nil {
		t.Fatalf("expected error for invalid type, got nil")
	}
}
