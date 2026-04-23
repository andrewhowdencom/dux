package librarian

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/andrewhowdencom/dux/pkg/llm"
	"github.com/andrewhowdencom/dux/pkg/memory/semantic"
)

type mockSemanticStore struct {
	statements []semantic.StatementFact
}

func (m *mockSemanticStore) WriteTriple(ctx context.Context, fact semantic.TripleFact) error { return nil }
func (m *mockSemanticStore) WriteStatement(ctx context.Context, fact semantic.StatementFact) error {
	m.statements = append(m.statements, fact)
	return nil
}
func (m *mockSemanticStore) WriteTriples(ctx context.Context, facts []semantic.TripleFact) error   { return nil }
func (m *mockSemanticStore) WriteStatements(ctx context.Context, facts []semantic.StatementFact) error { return nil }
func (m *mockSemanticStore) ReadFact(ctx context.Context, id string) (semantic.Fact, error) {
	return nil, semantic.ErrNotFound
}
func (m *mockSemanticStore) ReadTriple(ctx context.Context, id string) (semantic.TripleFact, error) {
	return semantic.TripleFact{}, semantic.ErrNotFound
}
func (m *mockSemanticStore) ReadStatement(ctx context.Context, id string) (semantic.StatementFact, error) {
	return semantic.StatementFact{}, semantic.ErrNotFound
}
func (m *mockSemanticStore) Search(ctx context.Context, query semantic.SearchQuery) ([]semantic.Fact, error) {
	return nil, nil
}
func (m *mockSemanticStore) DeleteFact(ctx context.Context, id string) error                        { return nil }
func (m *mockSemanticStore) DeleteByEntityAttribute(ctx context.Context, entity, attribute string) error { return nil }
func (m *mockSemanticStore) WriteRelationship(ctx context.Context, rel semantic.Relationship) error    { return nil }
func (m *mockSemanticStore) ReadRelationships(ctx context.Context, subject string) ([]semantic.Relationship, error) {
	return nil, nil
}
func (m *mockSemanticStore) DeleteRelationship(ctx context.Context, id string) error { return nil }
func (m *mockSemanticStore) TraverseGraph(ctx context.Context, query semantic.GraphQuery) (semantic.GraphResult, error) {
	return semantic.GraphResult{}, nil
}
func (m *mockSemanticStore) Close() error { return nil }

func TestAfterCompleteHook_PersistsSessionSummary(t *testing.T) {
	store := &mockSemanticStore{}
	service := semantic.NewService(store)
	hook := NewAfterCompleteHook(service)

	ctx := context.Background()
	req := llm.AfterCompleteRequest{
		SessionID: "session-abc",
		FinalMessage: llm.Message{
			Identity: llm.Identity{Role: "assistant"},
			Parts:    []llm.Part{llm.TextPart("Plan approved successfully.")},
		},
		ToolHistory: []llm.ToolExecutionRecord{
			{
				ToolCall: llm.ToolRequestPart{Name: "plan_approve"},
				Result:   llm.ToolResultPart{Result: "approved"},
				Duration: time.Second,
			},
		},
	}

	if err := hook(ctx, req); err != nil {
		t.Fatalf("hook returned error: %v", err)
	}

	if len(store.statements) != 1 {
		t.Fatalf("expected 1 statement written, got %d", len(store.statements))
	}

	fact := store.statements[0]
	if !strings.Contains(fact.Statement, "session-abc") {
		t.Errorf("expected statement to contain session ID, got: %s", fact.Statement)
	}
	if !strings.Contains(fact.Statement, "plan_approve") {
		t.Errorf("expected statement to contain tool history, got: %s", fact.Statement)
	}
	if !strings.Contains(fact.Statement, "Plan approved successfully.") {
		t.Errorf("expected statement to contain final message, got: %s", fact.Statement)
	}
	if len(fact.Tags) != 2 || fact.Tags[0] != "session-summary" {
		t.Errorf("expected tags [session-summary auto-extracted], got %v", fact.Tags)
	}
}

func TestAfterCompleteHook_NilService(t *testing.T) {
	hook := NewAfterCompleteHook(nil)
	ctx := context.Background()
	req := llm.AfterCompleteRequest{
		SessionID: "session-xyz",
		FinalMessage: llm.Message{
			Parts: []llm.Part{llm.TextPart("Done.")},
		},
	}

	if err := hook(ctx, req); err != nil {
		t.Fatalf("expected nil error with nil service, got %v", err)
	}
}

func TestAfterCompleteHook_ErrorToolHistory(t *testing.T) {
	store := &mockSemanticStore{}
	service := semantic.NewService(store)
	hook := NewAfterCompleteHook(service)

	ctx := context.Background()
	req := llm.AfterCompleteRequest{
		SessionID: "session-err",
		FinalMessage: llm.Message{
			Parts: []llm.Part{llm.TextPart("Finished with errors.")},
		},
		ToolHistory: []llm.ToolExecutionRecord{
			{
				ToolCall: llm.ToolRequestPart{Name: "bash"},
				Result:   llm.ToolResultPart{Result: "command failed", IsError: true},
				Error:    fmt.Errorf("exit status 1"),
			},
		},
	}

	if err := hook(ctx, req); err != nil {
		t.Fatalf("hook returned error: %v", err)
	}

	if len(store.statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(store.statements))
	}

	fact := store.statements[0]
	if !strings.Contains(fact.Statement, "error") {
		t.Errorf("expected statement to mention error status, got: %s", fact.Statement)
	}
}
