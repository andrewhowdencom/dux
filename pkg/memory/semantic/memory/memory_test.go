package memory_test

import (
	"context"
	"testing"
	"time"

	"github.com/andrewhowdencom/dux/pkg/memory/semantic"
	"github.com/andrewhowdencom/dux/pkg/memory/semantic/memory"
)

func TestMemoryStore_WriteReadTriple(t *testing.T) {
	store := memory.NewStore()
	defer func() { _ = store.Close() }()

	factID := "test-triple-1"
	now := time.Now()
	triple := semantic.TripleFact{
		ID:        factID,
		Entity:    "user",
		Attribute: "theme",
		Value:     "dark",
		Sources: []semantic.Source{
			{URI: "test://source", RetrievedAt: now},
		},
		Tags: []string{"preference", "ui"},
		Metadata: semantic.FactMetadata{
			CreatedAt:    now,
			ValidatedAt:  now,
			LastAccessed: now,
			AccessCount:  0,
			AccessScore:  0,
			Constraints: map[string]string{
				"project": "dux",
			},
		},
	}

	err := store.WriteTriple(context.Background(), triple)
	if err != nil {
		t.Fatalf("WriteTriple failed: %v", err)
	}

	retrieved, err := store.ReadTriple(context.Background(), factID)
	if err != nil {
		t.Fatalf("ReadTriple failed: %v", err)
	}

	if retrieved.ID != factID {
		t.Errorf("expected ID %s, got %s", factID, retrieved.ID)
	}
	if retrieved.Entity != "user" {
		t.Errorf("expected entity 'user', got %s", retrieved.Entity)
	}
	if retrieved.Attribute != "theme" {
		t.Errorf("expected attribute 'theme', got %s", retrieved.Attribute)
	}
	if retrieved.Value != "dark" {
		t.Errorf("expected value 'dark', got %s", retrieved.Value)
	}
	if len(retrieved.Tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(retrieved.Tags))
	}
	if retrieved.Metadata.Constraints["project"] != "dux" {
		t.Errorf("expected constraint project=dux, got %v", retrieved.Metadata.Constraints)
	}
}

func TestMemoryStore_WriteReadStatement(t *testing.T) {
	store := memory.NewStore()
	defer func() { _ = store.Close() }()

	factID := "test-statement-1"
	now := time.Now()
	statement := semantic.StatementFact{
		ID:        factID,
		Statement: "The project uses Go programming language",
		Sources: []semantic.Source{
			{URI: "test://doc", RetrievedAt: now},
		},
		Tags: []string{"tech", "language"},
		Metadata: semantic.FactMetadata{
			CreatedAt:    now,
			ValidatedAt:  now,
			LastAccessed: now,
			Constraints: map[string]string{
				"domain": "technical",
			},
		},
	}

	err := store.WriteStatement(context.Background(), statement)
	if err != nil {
		t.Fatalf("WriteStatement failed: %v", err)
	}

	retrieved, err := store.ReadStatement(context.Background(), factID)
	if err != nil {
		t.Fatalf("ReadStatement failed: %v", err)
	}

	if retrieved.Statement != "The project uses Go programming language" {
		t.Errorf("unexpected statement: %s", retrieved.Statement)
	}
	if retrieved.Metadata.Constraints["domain"] != "technical" {
		t.Errorf("expected constraint domain=technical, got %v", retrieved.Metadata.Constraints)
	}
}

func TestMemoryStore_SearchByConstraints(t *testing.T) {
	store := memory.NewStore()
	defer func() { _ = store.Close() }()

	now := time.Now()

	triples := []semantic.TripleFact{
		{
			ID: "fact-1", Entity: "user", Attribute: "pref", Value: "a",
			Metadata: semantic.FactMetadata{CreatedAt: now, ValidatedAt: now, LastAccessed: now, Constraints: map[string]string{"env": "prod"}},
		},
		{
			ID: "fact-2", Entity: "user", Attribute: "pref", Value: "b",
			Metadata: semantic.FactMetadata{CreatedAt: now, ValidatedAt: now, LastAccessed: now, Constraints: map[string]string{"env": "dev"}},
		},
		{
			ID: "fact-3", Entity: "user", Attribute: "pref", Value: "c",
			Metadata: semantic.FactMetadata{CreatedAt: now, ValidatedAt: now, LastAccessed: now, Constraints: map[string]string{"env": "prod", "team": "backend"}},
		},
	}

	for _, tri := range triples {
		err := store.WriteTriple(context.Background(), tri)
		if err != nil {
			t.Fatalf("WriteTriple failed: %v", err)
		}
	}

	tests := []struct {
		name        string
		constraints map[string]string
		expectCount int
	}{
		{
			name:        "match env=prod",
			constraints: map[string]string{"env": "prod"},
			expectCount: 2,
		},
		{
			name:        "match env=prod and team=backend",
			constraints: map[string]string{"env": "prod", "team": "backend"},
			expectCount: 1,
		},
		{
			name:        "no match env=staging",
			constraints: map[string]string{"env": "staging"},
			expectCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := semantic.SearchQuery{
				Constraints: tt.constraints,
			}
			facts, err := store.Search(context.Background(), query)
			if err != nil {
				t.Fatalf("Search failed: %v", err)
			}
			if len(facts) != tt.expectCount {
				t.Errorf("expected %d facts, got %d", tt.expectCount, len(facts))
			}
		})
	}
}

func TestMemoryStore_DeleteFact(t *testing.T) {
	store := memory.NewStore()
	defer func() { _ = store.Close() }()

	factID := "test-delete-1"
	triple := semantic.TripleFact{
		ID:        factID,
		Entity:    "user",
		Attribute: "pref",
		Value:     "test",
		Metadata:  semantic.FactMetadata{CreatedAt: time.Now(), ValidatedAt: time.Now(), LastAccessed: time.Now()},
	}

	_ = store.WriteTriple(context.Background(), triple)

	err := store.DeleteFact(context.Background(), factID)
	if err != nil {
		t.Fatalf("DeleteFact failed: %v", err)
	}

	_, err = store.ReadTriple(context.Background(), factID)
	if err != semantic.ErrNotFound {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestMemoryStore_DeleteByEntityAttribute(t *testing.T) {
	store := memory.NewStore()
	defer func() { _ = store.Close() }()

	now := time.Now()
	triples := []semantic.TripleFact{
		{ID: "f1", Entity: "user", Attribute: "theme", Value: "dark", Metadata: semantic.FactMetadata{CreatedAt: now, ValidatedAt: now, LastAccessed: now}},
		{ID: "f2", Entity: "user", Attribute: "lang", Value: "en", Metadata: semantic.FactMetadata{CreatedAt: now, ValidatedAt: now, LastAccessed: now}},
		{ID: "f3", Entity: "project", Attribute: "theme", Value: "light", Metadata: semantic.FactMetadata{CreatedAt: now, ValidatedAt: now, LastAccessed: now}},
	}

	for _, tri := range triples {
		_ = store.WriteTriple(context.Background(), tri)
	}

	err := store.DeleteByEntityAttribute(context.Background(), "user", "theme")
	if err != nil {
		t.Fatalf("DeleteByEntityAttribute failed: %v", err)
	}

	_, err = store.ReadTriple(context.Background(), "f1")
	if err != semantic.ErrNotFound {
		t.Errorf("expected f1 to be deleted, got %v", err)
	}

	_, err = store.ReadTriple(context.Background(), "f2")
	if err != nil {
		t.Errorf("expected f2 to still exist, got %v", err)
	}

	_, err = store.ReadTriple(context.Background(), "f3")
	if err != nil {
		t.Errorf("expected f3 to still exist, got %v", err)
	}
}

func TestMemoryStore_SearchSortByAccessScore(t *testing.T) {
	store := memory.NewStore()
	defer func() { _ = store.Close() }()

	now := time.Now()
	triples := []semantic.TripleFact{
		{ID: "f1", Entity: "e", Attribute: "a", Value: "1", Metadata: semantic.FactMetadata{CreatedAt: now, ValidatedAt: now, LastAccessed: now, AccessScore: 5.0}},
		{ID: "f2", Entity: "e", Attribute: "a", Value: "2", Metadata: semantic.FactMetadata{CreatedAt: now, ValidatedAt: now, LastAccessed: now, AccessScore: 10.0}},
		{ID: "f3", Entity: "e", Attribute: "a", Value: "3", Metadata: semantic.FactMetadata{CreatedAt: now, ValidatedAt: now, LastAccessed: now, AccessScore: 3.0}},
	}

	for _, tri := range triples {
		_ = store.WriteTriple(context.Background(), tri)
	}

	query := semantic.SearchQuery{
		SortBy:    semantic.SortByAccessScore,
		SortOrder: semantic.SortDesc,
	}

	facts, err := store.Search(context.Background(), query)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(facts) != 3 {
		t.Fatalf("expected 3 facts, got %d", len(facts))
	}

	if facts[0].GetMetadata().AccessScore != 10.0 {
		t.Errorf("expected first fact to have score 10.0, got %.2f", facts[0].GetMetadata().AccessScore)
	}
	if facts[2].GetMetadata().AccessScore != 3.0 {
		t.Errorf("expected last fact to have score 3.0, got %.2f", facts[2].GetMetadata().AccessScore)
	}
}

func TestMemoryStore_NotFound(t *testing.T) {
	store := memory.NewStore()
	defer func() { _ = store.Close() }()

	_, err := store.ReadTriple(context.Background(), "nonexistent")
	if err != semantic.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}
