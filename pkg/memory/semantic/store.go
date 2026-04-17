package semantic

import (
	"context"
	"errors"
)

var ErrNotFound = errors.New("fact not found")

type Store interface {
	WriteTriple(ctx context.Context, fact TripleFact) error
	WriteStatement(ctx context.Context, fact StatementFact) error
	WriteTriples(ctx context.Context, facts []TripleFact) error
	WriteStatements(ctx context.Context, facts []StatementFact) error

	ReadFact(ctx context.Context, id string) (Fact, error)
	ReadTriple(ctx context.Context, id string) (TripleFact, error)
	ReadStatement(ctx context.Context, id string) (StatementFact, error)

	Search(ctx context.Context, query SearchQuery) ([]Fact, error)

	DeleteFact(ctx context.Context, id string) error
	DeleteByEntityAttribute(ctx context.Context, entity, attribute string) error

	WriteRelationship(ctx context.Context, rel Relationship) error
	ReadRelationships(ctx context.Context, subject string) ([]Relationship, error)
	DeleteRelationship(ctx context.Context, id string) error
	TraverseGraph(ctx context.Context, query GraphQuery) (GraphResult, error)

	Close() error
}

type SearchQuery struct {
	Type        *FactType
	Entity      *string
	Attribute   *string
	Value       *string
	Tag         *string
	Source      *string
	Statement   *string
	Constraints map[string]string
	Limit       int
	Offset      int
	SortBy      SortField
	SortOrder   SortOrder
}

type SortField string

const (
	SortByCreatedAt   SortField = "created_at"
	SortByValidatedAt SortField = "validated_at"
	SortByAccessScore SortField = "access_score"
	SortByRelevance   SortField = "relevance"
)

type SortOrder string

const (
	SortAsc  SortOrder = "asc"
	SortDesc SortOrder = "desc"
)
