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

func TestMemoryStore_WriteReadRelationship(t *testing.T) {
	store := memory.NewStore()
	defer func() { _ = store.Close() }()

	relID := "test-rel-1"
	now := time.Now()
	rel := semantic.Relationship{
		ID:        relID,
		Subject:   "person:john",
		Predicate: "has_condition",
		Object:    "condition:pvcs",
		Metadata: semantic.RelationshipMetadata{
			CreatedAt: now,
		},
	}

	err := store.WriteRelationship(context.Background(), rel)
	if err != nil {
		t.Fatalf("WriteRelationship failed: %v", err)
	}

	rels, err := store.ReadRelationships(context.Background(), "person:john")
	if err != nil {
		t.Fatalf("ReadRelationships failed: %v", err)
	}

	if len(rels) != 1 {
		t.Fatalf("expected 1 relationship, got %d", len(rels))
	}

	if rels[0].Predicate != "has_condition" {
		t.Errorf("expected predicate 'has_condition', got %s", rels[0].Predicate)
	}
	if rels[0].Object != "condition:pvcs" {
		t.Errorf("expected object 'condition:pvcs', got %s", rels[0].Object)
	}
}

func TestMemoryStore_DeleteRelationship(t *testing.T) {
	store := memory.NewStore()
	defer func() { _ = store.Close() }()

	relID := "test-rel-delete"
	now := time.Now()
	rel := semantic.Relationship{
		ID:        relID,
		Subject:   "person:jane",
		Predicate: "works_at",
		Object:    "company:acme",
		Metadata: semantic.RelationshipMetadata{
			CreatedAt: now,
		},
	}

	_ = store.WriteRelationship(context.Background(), rel)

	err := store.DeleteRelationship(context.Background(), relID)
	if err != nil {
		t.Fatalf("DeleteRelationship failed: %v", err)
	}

	rels, err := store.ReadRelationships(context.Background(), "person:jane")
	if err != nil {
		t.Fatalf("ReadRelationships failed: %v", err)
	}

	if len(rels) != 0 {
		t.Errorf("expected 0 relationships after delete, got %d", len(rels))
	}
}

func TestMemoryStore_TraverseGraph(t *testing.T) {
	store := memory.NewStore()
	defer func() { _ = store.Close() }()

	now := time.Now()

	triples := []semantic.TripleFact{
		{ID: "f1", Entity: "person:john", Attribute: "name", Value: "John Doe", Metadata: semantic.FactMetadata{CreatedAt: now, ValidatedAt: now, LastAccessed: now}},
		{ID: "f2", Entity: "condition:pvcs", Attribute: "name", Value: "Premature Ventricular Contractions", Metadata: semantic.FactMetadata{CreatedAt: now, ValidatedAt: now, LastAccessed: now}},
		{ID: "f3", Entity: "symptom:palpitations", Attribute: "description", Value: "Heart palpitations", Metadata: semantic.FactMetadata{CreatedAt: now, ValidatedAt: now, LastAccessed: now}},
	}

	for _, tri := range triples {
		_ = store.WriteTriple(context.Background(), tri)
	}

	rels := []semantic.Relationship{
		{ID: "r1", Subject: "person:john", Predicate: "has_condition", Object: "condition:pvcs", Metadata: semantic.RelationshipMetadata{CreatedAt: now}},
		{ID: "r2", Subject: "condition:pvcs", Predicate: "has_symptom", Object: "symptom:palpitations", Metadata: semantic.RelationshipMetadata{CreatedAt: now}},
	}

	for _, rel := range rels {
		_ = store.WriteRelationship(context.Background(), rel)
	}

	query := semantic.GraphQuery{
		StartEntity: "person:john",
		MaxDepth:    2,
	}

	result, err := store.TraverseGraph(context.Background(), query)
	if err != nil {
		t.Fatalf("TraverseGraph failed: %v", err)
	}

	if len(result.Nodes) < 1 {
		t.Errorf("expected at least 1 node, got %d", len(result.Nodes))
	}

	if len(result.Edges) != 2 {
		t.Errorf("expected 2 edges, got %d", len(result.Edges))
	}

	foundJohn := false
	for _, node := range result.Nodes {
		if node.Entity == "person:john" {
			foundJohn = true
			break
		}
	}

	if !foundJohn {
		t.Error("expected to find person:john node")
	}
}

func TestMemoryStore_TraverseGraphWithPredicateFilter(t *testing.T) {
	store := memory.NewStore()
	defer func() { _ = store.Close() }()

	now := time.Now()

	rels := []semantic.Relationship{
		{ID: "r1", Subject: "person:john", Predicate: "has_condition", Object: "condition:pvcs", Metadata: semantic.RelationshipMetadata{CreatedAt: now}},
		{ID: "r2", Subject: "person:john", Predicate: "works_at", Object: "company:acme", Metadata: semantic.RelationshipMetadata{CreatedAt: now}},
	}

	for _, rel := range rels {
		_ = store.WriteRelationship(context.Background(), rel)
	}

	query := semantic.GraphQuery{
		StartEntity: "person:john",
		Predicates:  []string{"has_condition"},
		MaxDepth:    1,
	}

	result, err := store.TraverseGraph(context.Background(), query)
	if err != nil {
		t.Fatalf("TraverseGraph failed: %v", err)
	}

	if len(result.Edges) != 1 {
		t.Errorf("expected 1 edge with predicate filter, got %d", len(result.Edges))
	}

	if len(result.Edges) > 0 && result.Edges[0].Predicate != "has_condition" {
		t.Errorf("expected predicate 'has_condition', got %s", result.Edges[0].Predicate)
	}
}

func TestMemoryStore_TraverseGraphMaxDepth(t *testing.T) {
	store := memory.NewStore()
	defer func() { _ = store.Close() }()

	now := time.Now()

	rels := []semantic.Relationship{
		{ID: "r1", Subject: "a", Predicate: "rel1", Object: "b", Metadata: semantic.RelationshipMetadata{CreatedAt: now}},
		{ID: "r2", Subject: "b", Predicate: "rel2", Object: "c", Metadata: semantic.RelationshipMetadata{CreatedAt: now}},
		{ID: "r3", Subject: "c", Predicate: "rel3", Object: "d", Metadata: semantic.RelationshipMetadata{CreatedAt: now}},
	}

	for _, rel := range rels {
		_ = store.WriteRelationship(context.Background(), rel)
	}

	query := semantic.GraphQuery{
		StartEntity: "a",
		MaxDepth:    1,
	}

	result, err := store.TraverseGraph(context.Background(), query)
	if err != nil {
		t.Fatalf("TraverseGraph failed: %v", err)
	}

	if len(result.Edges) != 1 {
		t.Errorf("expected 1 edge with depth 1, got %d", len(result.Edges))
	}

	query.MaxDepth = 3
	result, err = store.TraverseGraph(context.Background(), query)
	if err != nil {
		t.Fatalf("TraverseGraph failed: %v", err)
	}

	if len(result.Edges) != 3 {
		t.Errorf("expected 3 edges with depth 3, got %d", len(result.Edges))
	}
}
