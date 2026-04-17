package sqlite_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/andrewhowdencom/dux/pkg/memory/semantic"
	"github.com/andrewhowdencom/dux/pkg/memory/semantic/sqlite"
)

func setupTestStore(t *testing.T) (*sqlite.Store, func()) {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	store, err := sqlite.NewStore(dbPath)
	if err != nil {
		t.Fatalf("failed to create test store: %v", err)
	}

	cleanup := func() {
		_ = store.Close()
		_ = os.Remove(dbPath)
	}

	return store, cleanup
}

func TestSQLiteStore_WriteReadTriple(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

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

func TestSQLiteStore_WriteReadStatement(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

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

func TestSQLiteStore_WriteTriples(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	now := time.Now()
	triples := []semantic.TripleFact{
		{
			ID: "f1", Entity: "user", Attribute: "pref", Value: "a",
			Metadata: semantic.FactMetadata{CreatedAt: now, ValidatedAt: now, LastAccessed: now},
		},
		{
			ID: "f2", Entity: "user", Attribute: "pref", Value: "b",
			Metadata: semantic.FactMetadata{CreatedAt: now, ValidatedAt: now, LastAccessed: now},
		},
	}

	err := store.WriteTriples(context.Background(), triples)
	if err != nil {
		t.Fatalf("WriteTriples failed: %v", err)
	}

	for _, tri := range triples {
		_, err := store.ReadTriple(context.Background(), tri.ID)
		if err != nil {
			t.Errorf("failed to read triple %s: %v", tri.ID, err)
		}
	}
}

func TestSQLiteStore_WriteStatements(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	now := time.Now()
	statements := []semantic.StatementFact{
		{
			ID: "s1", Statement: "First statement",
			Metadata: semantic.FactMetadata{CreatedAt: now, ValidatedAt: now, LastAccessed: now},
		},
		{
			ID: "s2", Statement: "Second statement",
			Metadata: semantic.FactMetadata{CreatedAt: now, ValidatedAt: now, LastAccessed: now},
		},
	}

	err := store.WriteStatements(context.Background(), statements)
	if err != nil {
		t.Fatalf("WriteStatements failed: %v", err)
	}

	for _, stmt := range statements {
		_, err := store.ReadStatement(context.Background(), stmt.ID)
		if err != nil {
			t.Errorf("failed to read statement %s: %v", stmt.ID, err)
		}
	}
}

func TestSQLiteStore_SearchByEntityAttribute(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	now := time.Now()
	triples := []semantic.TripleFact{
		{ID: "f1", Entity: "user", Attribute: "theme", Value: "dark", Metadata: semantic.FactMetadata{CreatedAt: now, ValidatedAt: now, LastAccessed: now}},
		{ID: "f2", Entity: "user", Attribute: "lang", Value: "en", Metadata: semantic.FactMetadata{CreatedAt: now, ValidatedAt: now, LastAccessed: now}},
		{ID: "f3", Entity: "project", Attribute: "theme", Value: "light", Metadata: semantic.FactMetadata{CreatedAt: now, ValidatedAt: now, LastAccessed: now}},
	}

	for _, tri := range triples {
		err := store.WriteTriple(context.Background(), tri)
		if err != nil {
			t.Fatalf("WriteTriple failed: %v", err)
		}
	}

	tests := []struct {
		name      string
		query     semantic.SearchQuery
		expectLen int
	}{
		{
			name:      "search by entity",
			query:     semantic.SearchQuery{Entity: strPtr("user")},
			expectLen: 2,
		},
		{
			name:      "search by attribute",
			query:     semantic.SearchQuery{Attribute: strPtr("theme")},
			expectLen: 2,
		},
		{
			name:      "search by entity and attribute",
			query:     semantic.SearchQuery{Entity: strPtr("user"), Attribute: strPtr("theme")},
			expectLen: 1,
		},
		{
			name:      "search by value",
			query:     semantic.SearchQuery{Value: strPtr("dark")},
			expectLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			facts, err := store.Search(context.Background(), tt.query)
			if err != nil {
				t.Fatalf("Search failed: %v", err)
			}
			if len(facts) != tt.expectLen {
				t.Errorf("expected %d facts, got %d", tt.expectLen, len(facts))
			}
		})
	}
}

func TestSQLiteStore_SearchByStatement(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	now := time.Now()
	statements := []semantic.StatementFact{
		{ID: "s1", Statement: "Go is a programming language", Metadata: semantic.FactMetadata{CreatedAt: now, ValidatedAt: now, LastAccessed: now}},
		{ID: "s2", Statement: "Python is great for ML", Metadata: semantic.FactMetadata{CreatedAt: now, ValidatedAt: now, LastAccessed: now}},
		{ID: "s3", Statement: "Go has strong typing", Metadata: semantic.FactMetadata{CreatedAt: now, ValidatedAt: now, LastAccessed: now}},
	}

	for _, stmt := range statements {
		err := store.WriteStatement(context.Background(), stmt)
		if err != nil {
			t.Fatalf("WriteStatement failed: %v", err)
		}
	}

	query := semantic.SearchQuery{Statement: strPtr("Go")}
	facts, err := store.Search(context.Background(), query)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(facts) != 2 {
		t.Errorf("expected 2 facts containing 'Go', got %d", len(facts))
	}
}

func TestSQLiteStore_SearchByTag(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	now := time.Now()
	triples := []semantic.TripleFact{
		{ID: "f1", Entity: "user", Attribute: "pref", Value: "a", Tags: []string{"important", "ui"}, Metadata: semantic.FactMetadata{CreatedAt: now, ValidatedAt: now, LastAccessed: now}},
		{ID: "f2", Entity: "user", Attribute: "pref", Value: "b", Tags: []string{"ui"}, Metadata: semantic.FactMetadata{CreatedAt: now, ValidatedAt: now, LastAccessed: now}},
		{ID: "f3", Entity: "user", Attribute: "pref", Value: "c", Tags: []string{"backend"}, Metadata: semantic.FactMetadata{CreatedAt: now, ValidatedAt: now, LastAccessed: now}},
	}

	for _, tri := range triples {
		err := store.WriteTriple(context.Background(), tri)
		if err != nil {
			t.Fatalf("WriteTriple failed: %v", err)
		}
	}

	query := semantic.SearchQuery{Tag: strPtr("ui")}
	facts, err := store.Search(context.Background(), query)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(facts) != 2 {
		t.Errorf("expected 2 facts with tag 'ui', got %d", len(facts))
	}
}

func TestSQLiteStore_SearchByConstraints(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	now := time.Now()
	triples := []semantic.TripleFact{
		{ID: "f1", Entity: "user", Attribute: "pref", Value: "a", Metadata: semantic.FactMetadata{CreatedAt: now, ValidatedAt: now, LastAccessed: now, Constraints: map[string]string{"env": "prod"}}},
		{ID: "f2", Entity: "user", Attribute: "pref", Value: "b", Metadata: semantic.FactMetadata{CreatedAt: now, ValidatedAt: now, LastAccessed: now, Constraints: map[string]string{"env": "dev"}}},
		{ID: "f3", Entity: "user", Attribute: "pref", Value: "c", Metadata: semantic.FactMetadata{CreatedAt: now, ValidatedAt: now, LastAccessed: now, Constraints: map[string]string{"env": "prod", "team": "backend"}}},
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

func TestSQLiteStore_SearchByType(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	now := time.Now()
	triple := semantic.TripleFact{
		ID: "t1", Entity: "user", Attribute: "pref", Value: "a",
		Metadata: semantic.FactMetadata{CreatedAt: now, ValidatedAt: now, LastAccessed: now},
	}
	statement := semantic.StatementFact{
		ID: "s1", Statement: "Test statement",
		Metadata: semantic.FactMetadata{CreatedAt: now, ValidatedAt: now, LastAccessed: now},
	}

	_ = store.WriteTriple(context.Background(), triple)
	_ = store.WriteStatement(context.Background(), statement)

	tripleType := semantic.FactTypeTriple
	tripleQuery := semantic.SearchQuery{Type: &tripleType}
	tripleFacts, err := store.Search(context.Background(), tripleQuery)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(tripleFacts) != 1 {
		t.Errorf("expected 1 triple, got %d", len(tripleFacts))
	}

	stmtType := semantic.FactTypeStatement
	stmtQuery := semantic.SearchQuery{Type: &stmtType}
	stmtFacts, err := store.Search(context.Background(), stmtQuery)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(stmtFacts) != 1 {
		t.Errorf("expected 1 statement, got %d", len(stmtFacts))
	}
}

func TestSQLiteStore_SearchLimitAndOffset(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	now := time.Now()
	for i := 0; i < 10; i++ {
		triple := semantic.TripleFact{
			ID: "f" + string(rune('0'+i)), Entity: "user", Attribute: "pref", Value: "v",
			Metadata: semantic.FactMetadata{CreatedAt: now, ValidatedAt: now, LastAccessed: now},
		}
		_ = store.WriteTriple(context.Background(), triple)
	}

	query := semantic.SearchQuery{Limit: 3, Offset: 0}
	facts, err := store.Search(context.Background(), query)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(facts) != 3 {
		t.Errorf("expected 3 facts, got %d", len(facts))
	}

	query = semantic.SearchQuery{Limit: 3, Offset: 5}
	facts, err = store.Search(context.Background(), query)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(facts) != 3 {
		t.Errorf("expected 3 facts with offset, got %d", len(facts))
	}
}

func TestSQLiteStore_SearchSortByAccessScore(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

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

func TestSQLiteStore_DeleteFact(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

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

func TestSQLiteStore_DeleteByEntityAttribute(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

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

func TestSQLiteStore_NotFound(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	_, err := store.ReadTriple(context.Background(), "nonexistent")
	if err != semantic.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestSQLiteStore_IdempotentWrite(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	now := time.Now()
	triple := semantic.TripleFact{
		ID:        "idempotent-1",
		Entity:    "user",
		Attribute: "pref",
		Value:     "initial",
		Metadata:  semantic.FactMetadata{CreatedAt: now, ValidatedAt: now, LastAccessed: now},
	}

	_ = store.WriteTriple(context.Background(), triple)

	triple.Value = "updated"
	_ = store.WriteTriple(context.Background(), triple)

	retrieved, err := store.ReadTriple(context.Background(), "idempotent-1")
	if err != nil {
		t.Fatalf("ReadTriple failed: %v", err)
	}

	if retrieved.Value != "updated" {
		t.Errorf("expected value 'updated', got %s", retrieved.Value)
	}
}

func strPtr(s string) *string {
	return &s
}
