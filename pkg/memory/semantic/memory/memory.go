package memory

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/andrewhowdencom/dux/pkg/memory/semantic"
)

type Store struct {
	mu    sync.RWMutex
	facts map[string]semantic.Fact
}

func NewStore() *Store {
	return &Store{
		facts: make(map[string]semantic.Fact),
	}
}

func (s *Store) WriteTriple(ctx context.Context, fact semantic.TripleFact) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.facts[fact.ID] = fact
	return nil
}

func (s *Store) WriteStatement(ctx context.Context, fact semantic.StatementFact) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.facts[fact.ID] = fact
	return nil
}

func (s *Store) WriteTriples(ctx context.Context, facts []semantic.TripleFact) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, f := range facts {
		s.facts[f.ID] = f
	}
	return nil
}

func (s *Store) WriteStatements(ctx context.Context, facts []semantic.StatementFact) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, f := range facts {
		s.facts[f.ID] = f
	}
	return nil
}

func (s *Store) ReadFact(ctx context.Context, id string) (semantic.Fact, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	fact, ok := s.facts[id]
	if !ok {
		return nil, semantic.ErrNotFound
	}
	return fact, nil
}

func (s *Store) ReadTriple(ctx context.Context, id string) (semantic.TripleFact, error) {
	fact, err := s.ReadFact(ctx, id)
	if err != nil {
		return semantic.TripleFact{}, err
	}
	tf, ok := fact.(semantic.TripleFact)
	if !ok {
		return semantic.TripleFact{}, fmt.Errorf("fact %s is not a triple", id)
	}
	return tf, nil
}

func (s *Store) ReadStatement(ctx context.Context, id string) (semantic.StatementFact, error) {
	fact, err := s.ReadFact(ctx, id)
	if err != nil {
		return semantic.StatementFact{}, err
	}
	sf, ok := fact.(semantic.StatementFact)
	if !ok {
		return semantic.StatementFact{}, fmt.Errorf("fact %s is not a statement", id)
	}
	return sf, nil
}

func (s *Store) Search(ctx context.Context, query semantic.SearchQuery) ([]semantic.Fact, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []semantic.Fact
	for _, fact := range s.facts {
		if !matchesQuery(fact, query) {
			continue
		}
		results = append(results, fact)
	}

	sortFacts(results, query)

	limit := query.Limit
	if limit <= 0 {
		limit = 50
	}
	offset := query.Offset

	if offset >= len(results) {
		return []semantic.Fact{}, nil
	}

	end := offset + limit
	if end > len(results) {
		end = len(results)
	}
	return results[offset:end], nil
}

func (s *Store) DeleteFact(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.facts, id)
	return nil
}

func (s *Store) DeleteByEntityAttribute(ctx context.Context, entity, attr string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, fact := range s.facts {
		if tf, ok := fact.(semantic.TripleFact); ok {
			if tf.Entity == entity && tf.Attribute == attr {
				delete(s.facts, id)
			}
		}
	}
	return nil
}

func (s *Store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.facts = make(map[string]semantic.Fact)
	return nil
}

func matchesQuery(fact semantic.Fact, query semantic.SearchQuery) bool {
	if query.Type != nil && fact.GetType() != *query.Type {
		return false
	}

	if tf, ok := fact.(semantic.TripleFact); ok {
		if query.Entity != nil && tf.Entity != *query.Entity {
			return false
		}
		if query.Attribute != nil && tf.Attribute != *query.Attribute {
			return false
		}
		if query.Value != nil && tf.Value != *query.Value {
			return false
		}
	}

	if sf, ok := fact.(semantic.StatementFact); ok {
		if query.Statement != nil && !strings.Contains(sf.Statement, *query.Statement) {
			return false
		}
	}

	if query.Tag != nil {
		hasTag := false
		for _, tag := range fact.GetTags() {
			if tag == *query.Tag {
				hasTag = true
				break
			}
		}
		if !hasTag {
			return false
		}
	}

	if query.Source != nil {
		hasSource := false
		for _, src := range fact.GetSources() {
			if src.URI == *query.Source {
				hasSource = true
				break
			}
		}
		if !hasSource {
			return false
		}
	}

	if len(query.Constraints) > 0 {
		factConstraints := fact.GetMetadata().Constraints
		if factConstraints == nil {
			return false
		}
		for k, v := range query.Constraints {
			if factConstraints[k] != v {
				return false
			}
		}
	}

	return true
}

func sortFacts(facts []semantic.Fact, query semantic.SearchQuery) {
	sortField := query.SortBy
	if sortField == "" {
		sortField = semantic.SortByCreatedAt
	}
	sortOrder := query.SortOrder
	if sortOrder == "" {
		sortOrder = semantic.SortDesc
	}

	sort.Slice(facts, func(i, j int) bool {
		var less bool
		switch sortField {
		case semantic.SortByCreatedAt:
			less = facts[i].GetMetadata().CreatedAt.After(facts[j].GetMetadata().CreatedAt)
		case semantic.SortByValidatedAt:
			less = facts[i].GetMetadata().ValidatedAt.After(facts[j].GetMetadata().ValidatedAt)
		case semantic.SortByAccessScore:
			less = facts[i].GetMetadata().AccessScore > facts[j].GetMetadata().AccessScore
		default:
			less = facts[i].GetMetadata().CreatedAt.After(facts[j].GetMetadata().CreatedAt)
		}

		if sortOrder == semantic.SortAsc {
			return !less
		}
		return less
	})
}
