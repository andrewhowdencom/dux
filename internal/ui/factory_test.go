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

	// Test BeforeGenerate hook execution on prompt
	ctx := context.Background()
	req := &llm.BeforeGenerateRequest{SessionID: "test"}
	err = enrichers[2](ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedContent := "Do not rm -rf /."
	found := false
	for _, msg := range req.CurrentMessages {
		if strings.Contains(msg.Text(), expectedContent) {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("prompt enricher did not return expected content. Output: %v", req.CurrentMessages)
	}

	// Test BeforeGenerate hook execution on guard_rail
	reqGuardRail := &llm.BeforeGenerateRequest{SessionID: "test"}
	err = enrichers[3](ctx, reqGuardRail)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedGuardRailContent := "Do not mention unicorns."
	foundGuardRail := false
	for _, msg := range reqGuardRail.CurrentMessages {
		if strings.Contains(msg.Text(), expectedGuardRailContent) {
			foundGuardRail = true
			break
		}
	}
	if !foundGuardRail {
		t.Errorf("guard_rail enricher did not return expected content. Output: %v", reqGuardRail.CurrentMessages)
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
