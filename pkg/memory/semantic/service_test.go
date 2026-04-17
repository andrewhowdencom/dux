package semantic_test

import (
	"context"
	"testing"
	"time"

	"github.com/andrewhowdencom/dux/pkg/memory/semantic"
	"github.com/andrewhowdencom/dux/pkg/memory/semantic/memory"
)

type mockStore struct {
	facts map[string]semantic.Fact
}

func newMockStore() *mockStore {
	return &mockStore{facts: make(map[string]semantic.Fact)}
}

func (m *mockStore) WriteTriple(ctx context.Context, fact semantic.TripleFact) error {
	m.facts[fact.ID] = fact
	return nil
}

func (m *mockStore) WriteStatement(ctx context.Context, fact semantic.StatementFact) error {
	m.facts[fact.ID] = fact
	return nil
}

func (m *mockStore) WriteTriples(ctx context.Context, facts []semantic.TripleFact) error {
	for _, f := range facts {
		m.facts[f.ID] = f
	}
	return nil
}

func (m *mockStore) WriteStatements(ctx context.Context, facts []semantic.StatementFact) error {
	for _, f := range facts {
		m.facts[f.ID] = f
	}
	return nil
}

func (m *mockStore) ReadFact(ctx context.Context, id string) (semantic.Fact, error) {
	fact, ok := m.facts[id]
	if !ok {
		return nil, semantic.ErrNotFound
	}
	return fact, nil
}

func (m *mockStore) ReadTriple(ctx context.Context, id string) (semantic.TripleFact, error) {
	fact, err := m.ReadFact(ctx, id)
	if err != nil {
		return semantic.TripleFact{}, err
	}
	tf, ok := fact.(semantic.TripleFact)
	if !ok {
		return semantic.TripleFact{}, semantic.ErrNotFound
	}
	return tf, nil
}

func (m *mockStore) ReadStatement(ctx context.Context, id string) (semantic.StatementFact, error) {
	fact, err := m.ReadFact(ctx, id)
	if err != nil {
		return semantic.StatementFact{}, err
	}
	sf, ok := fact.(semantic.StatementFact)
	if !ok {
		return semantic.StatementFact{}, semantic.ErrNotFound
	}
	return sf, nil
}

func (m *mockStore) Search(ctx context.Context, query semantic.SearchQuery) ([]semantic.Fact, error) {
	var results []semantic.Fact
	for _, fact := range m.facts {
		results = append(results, fact)
	}
	return results, nil
}

func (m *mockStore) DeleteFact(ctx context.Context, id string) error {
	delete(m.facts, id)
	return nil
}

func (m *mockStore) DeleteByEntityAttribute(ctx context.Context, entity, attr string) error {
	for id, fact := range m.facts {
		if tf, ok := fact.(semantic.TripleFact); ok {
			if tf.Entity == entity && tf.Attribute == attr {
				delete(m.facts, id)
			}
		}
	}
	return nil
}

func (m *mockStore) Close() error {
	return nil
}

func TestService_TrackAccess(t *testing.T) {
	tests := []struct {
		name          string
		initialScore  float64
		initialCount  int
		lastAccessed  time.Time
		expectedScore float64
		expectedCount int
	}{
		{
			name:          "first access",
			initialScore:  0,
			initialCount:  0,
			lastAccessed:  time.Now().Add(-24 * time.Hour),
			expectedScore: 1.0,
			expectedCount: 1,
		},
		{
			name:          "recent access with high score",
			initialScore:  5.0,
			initialCount:  5,
			lastAccessed:  time.Now().Add(-12 * time.Hour),
			expectedScore: 5.0 + 1.0,
			expectedCount: 6,
		},
		{
			name:          "old access with decay",
			initialScore:  3.0,
			initialCount:  3,
			lastAccessed:  time.Now().Add(-48 * time.Hour),
			expectedScore: 3.0 + 1.0/2.0,
			expectedCount: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newMockStore()
			service := semantic.NewService(store)

			factID := "test-fact-1"
			triple := semantic.TripleFact{
				ID:        factID,
				Entity:    "user",
				Attribute: "preference",
				Value:     "dark_mode",
				Metadata: semantic.FactMetadata{
					CreatedAt:    time.Now(),
					ValidatedAt:  time.Now(),
					LastAccessed: tt.lastAccessed,
					AccessCount:  tt.initialCount,
					AccessScore:  tt.initialScore,
				},
			}
			_ = store.WriteTriple(context.Background(), triple)

			err := service.TrackAccess(context.Background(), factID)
			if err != nil {
				t.Fatalf("TrackAccess failed: %v", err)
			}

			updatedFact, err := store.ReadTriple(context.Background(), factID)
			if err != nil {
				t.Fatalf("ReadTriple failed: %v", err)
			}

			if updatedFact.Metadata.AccessCount != tt.expectedCount {
				t.Errorf("expected access count %d, got %d", tt.expectedCount, updatedFact.Metadata.AccessCount)
			}

			scoreDiff := updatedFact.Metadata.AccessScore - tt.expectedScore
			if scoreDiff < -0.01 || scoreDiff > 0.01 {
				t.Errorf("expected access score %.2f, got %.2f", tt.expectedScore, updatedFact.Metadata.AccessScore)
			}
		})
	}
}

func TestService_ValidateFact(t *testing.T) {
	store := newMockStore()
	service := semantic.NewService(store)

	factID := "test-fact-1"
	beforeValidate := time.Now().Add(-1 * time.Hour)
	triple := semantic.TripleFact{
		ID:        factID,
		Entity:    "user",
		Attribute: "preference",
		Value:     "dark_mode",
		Metadata: semantic.FactMetadata{
			CreatedAt:    time.Now(),
			ValidatedAt:  beforeValidate,
			LastAccessed: time.Now(),
		},
	}
	_ = store.WriteTriple(context.Background(), triple)

	err := service.ValidateFact(context.Background(), factID)
	if err != nil {
		t.Fatalf("ValidateFact failed: %v", err)
	}

	updatedFact, err := store.ReadTriple(context.Background(), factID)
	if err != nil {
		t.Fatalf("ReadTriple failed: %v", err)
	}

	if !updatedFact.Metadata.ValidatedAt.After(beforeValidate) {
		t.Errorf("expected ValidatedAt to be updated, got %v", updatedFact.Metadata.ValidatedAt)
	}
}

func TestService_CalculateConfidence(t *testing.T) {
	tests := []struct {
		name        string
		metadata    semantic.FactMetadata
		sourceCount int
		minScore    float64
		maxScore    float64
	}{
		{
			name: "high confidence",
			metadata: semantic.FactMetadata{
				AccessScore:  10.0,
				ValidatedAt:  time.Now(),
				LastAccessed: time.Now(),
			},
			sourceCount: 3,
			minScore:    0.8,
			maxScore:    1.0,
		},
		{
			name: "low confidence - old validation",
			metadata: semantic.FactMetadata{
				AccessScore:  1.0,
				ValidatedAt:  time.Now().Add(-365 * 24 * time.Hour),
				LastAccessed: time.Now(),
			},
			sourceCount: 0,
			minScore:    0.0,
			maxScore:    0.3,
		},
		{
			name: "medium confidence",
			metadata: semantic.FactMetadata{
				AccessScore:  5.0,
				ValidatedAt:  time.Now().Add(-180 * 24 * time.Hour),
				LastAccessed: time.Now(),
			},
			sourceCount: 1,
			minScore:    0.3,
			maxScore:    0.7,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := semantic.NewService(newMockStore())
			score := service.CalculateConfidence(tt.metadata, tt.sourceCount)

			if score < tt.minScore || score > tt.maxScore {
				t.Errorf("expected confidence between %.2f and %.2f, got %.2f", tt.minScore, tt.maxScore, score)
			}
		})
	}
}

func TestService_Constraints(t *testing.T) {
	store := memory.NewStore()
	service := semantic.NewService(store)

	constraints := map[string]string{
		"project": "dux",
		"env":     "production",
	}

	factID := "test-constrained-fact"
	triple := semantic.TripleFact{
		ID:        factID,
		Entity:    "user",
		Attribute: "preference",
		Value:     "dark_mode",
		Metadata: semantic.FactMetadata{
			CreatedAt:    time.Now(),
			ValidatedAt:  time.Now(),
			LastAccessed: time.Now(),
			Constraints:  constraints,
		},
	}

	err := store.WriteTriple(context.Background(), triple)
	if err != nil {
		t.Fatalf("WriteTriple failed: %v", err)
	}

	query := semantic.SearchQuery{
		Constraints: map[string]string{
			"project": "dux",
		},
	}

	facts, err := service.Search(context.Background(), query)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(facts) != 1 {
		t.Fatalf("expected 1 fact, got %d", len(facts))
	}

	retrievedConstraints := facts[0].GetMetadata().Constraints
	if retrievedConstraints["project"] != "dux" || retrievedConstraints["env"] != "production" {
		t.Errorf("constraints mismatch: got %v", retrievedConstraints)
	}
}
