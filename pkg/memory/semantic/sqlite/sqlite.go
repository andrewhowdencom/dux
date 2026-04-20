package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/andrewhowdencom/dux/pkg/memory/semantic"
	_ "github.com/mattn/go-sqlite3"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

type Store struct {
	db     *sql.DB
	tracer trace.Tracer
	meter  metric.Meter
}

type Option func(*Store)

func WithTracer(tracer trace.Tracer) Option {
	return func(s *Store) { s.tracer = tracer }
}

func WithMeter(meter metric.Meter) Option {
	return func(s *Store) { s.meter = meter }
}

func NewStore(dbPath string, opts ...Option) (*Store, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	s := &Store{db: db}
	for _, opt := range opts {
		opt(s)
	}
	if s.tracer == nil {
		s.tracer = otel.GetTracerProvider().Tracer("github.com/andrewhowdencom/dux/pkg/memory/semantic/sqlite")
	}
	if s.meter == nil {
		s.meter = otel.GetMeterProvider().Meter("github.com/andrewhowdencom/dux/pkg/memory/semantic/sqlite")
	}

	if err := migrate(db); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return s, nil
}

func migrate(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS semantic_facts (
		id TEXT PRIMARY KEY,
		type TEXT NOT NULL,
		entity TEXT,
		attribute TEXT,
		value TEXT,
		statement TEXT,
		sources TEXT,
		tags TEXT,
		constraints TEXT,
		created_at TEXT,
		validated_at TEXT,
		last_accessed TEXT,
		access_count INTEGER DEFAULT 0,
		access_score REAL DEFAULT 0.0
	);
	CREATE TABLE IF NOT EXISTS semantic_relationships (
		id TEXT PRIMARY KEY,
		subject TEXT NOT NULL,
		predicate TEXT NOT NULL,
		object TEXT NOT NULL,
		created_at TEXT NOT NULL,
		metadata TEXT
	);
	CREATE INDEX IF NOT EXISTS idx_semantic_facts_type ON semantic_facts (type);
	CREATE INDEX IF NOT EXISTS idx_semantic_facts_entity_attr ON semantic_facts (entity, attribute);
	CREATE INDEX IF NOT EXISTS idx_semantic_facts_tags ON semantic_facts (tags);
	CREATE INDEX IF NOT EXISTS idx_semantic_facts_constraints ON semantic_facts (constraints);
	CREATE INDEX IF NOT EXISTS idx_relationships_subject ON semantic_relationships (subject);
	CREATE INDEX IF NOT EXISTS idx_relationships_predicate ON semantic_relationships (predicate);
	CREATE INDEX IF NOT EXISTS idx_relationships_object ON semantic_relationships (object);
	CREATE INDEX IF NOT EXISTS idx_relationships_subject_predicate ON semantic_relationships (subject, predicate);
	`
	_, err := db.Exec(schema)
	return err
}

func (s *Store) WriteTriple(ctx context.Context, fact semantic.TripleFact) error {
	ctx, span := s.tracer.Start(ctx, "sqlite.WriteTriple", trace.WithSpanKind(trace.SpanKindClient))
	defer span.End()
	span.SetAttributes(attribute.String("fact.id", fact.ID))

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	sourcesJSON, _ := json.Marshal(fact.Sources)
	tagsJSON, _ := json.Marshal(fact.Tags)
	constraintsJSON, _ := json.Marshal(fact.Metadata.Constraints)

	query := `
		INSERT INTO semantic_facts (id, type, entity, attribute, value, sources, tags, constraints, created_at, validated_at, last_accessed, access_count, access_score)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			entity=excluded.entity,
			attribute=excluded.attribute,
			value=excluded.value,
			sources=excluded.sources,
			tags=excluded.tags,
			constraints=excluded.constraints,
			created_at=excluded.created_at,
			validated_at=excluded.validated_at,
			last_accessed=excluded.last_accessed,
			access_count=excluded.access_count,
			access_score=excluded.access_score
	`
	_, err := s.db.ExecContext(ctx, query,
		fact.ID, semantic.FactTypeTriple, fact.Entity, fact.Attribute, fact.Value,
		string(sourcesJSON), string(tagsJSON), string(constraintsJSON),
		fact.Metadata.CreatedAt.Format(time.RFC3339),
		fact.Metadata.ValidatedAt.Format(time.RFC3339),
		fact.Metadata.LastAccessed.Format(time.RFC3339),
		fact.Metadata.AccessCount, fact.Metadata.AccessScore,
	)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("failed to write triple: %w", err)
	}
	return nil
}

func (s *Store) WriteStatement(ctx context.Context, fact semantic.StatementFact) error {
	ctx, span := s.tracer.Start(ctx, "sqlite.WriteStatement", trace.WithSpanKind(trace.SpanKindClient))
	defer span.End()
	span.SetAttributes(attribute.String("fact.id", fact.ID))

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	sourcesJSON, _ := json.Marshal(fact.Sources)
	tagsJSON, _ := json.Marshal(fact.Tags)
	constraintsJSON, _ := json.Marshal(fact.Metadata.Constraints)

	query := `
		INSERT INTO semantic_facts (id, type, statement, sources, tags, constraints, created_at, validated_at, last_accessed, access_count, access_score)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			statement=excluded.statement,
			sources=excluded.sources,
			tags=excluded.tags,
			constraints=excluded.constraints,
			created_at=excluded.created_at,
			validated_at=excluded.validated_at,
			last_accessed=excluded.last_accessed,
			access_count=excluded.access_count,
			access_score=excluded.access_score
	`
	_, err := s.db.ExecContext(ctx, query,
		fact.ID, semantic.FactTypeStatement, fact.Statement,
		string(sourcesJSON), string(tagsJSON), string(constraintsJSON),
		fact.Metadata.CreatedAt.Format(time.RFC3339),
		fact.Metadata.ValidatedAt.Format(time.RFC3339),
		fact.Metadata.LastAccessed.Format(time.RFC3339),
		fact.Metadata.AccessCount, fact.Metadata.AccessScore,
	)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("failed to write statement: %w", err)
	}
	return nil
}

func (s *Store) WriteTriples(ctx context.Context, facts []semantic.TripleFact) error {
	ctx, span := s.tracer.Start(ctx, "sqlite.WriteTriples", trace.WithSpanKind(trace.SpanKindClient))
	defer span.End()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.Prepare(`
		INSERT INTO semantic_facts (id, type, entity, attribute, value, sources, tags, constraints, created_at, validated_at, last_accessed, access_count, access_score)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			entity=excluded.entity, attribute=excluded.attribute, value=excluded.value,
			sources=excluded.sources, tags=excluded.tags, constraints=excluded.constraints,
			created_at=excluded.created_at, validated_at=excluded.validated_at,
			last_accessed=excluded.last_accessed, access_count=excluded.access_count, access_score=excluded.access_score
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer func() { _ = stmt.Close() }()

	for _, f := range facts {
		sourcesJSON, _ := json.Marshal(f.Sources)
		tagsJSON, _ := json.Marshal(f.Tags)
		constraintsJSON, _ := json.Marshal(f.Metadata.Constraints)

		_, err := stmt.Exec(
			f.ID, semantic.FactTypeTriple, f.Entity, f.Attribute, f.Value,
			string(sourcesJSON), string(tagsJSON), string(constraintsJSON),
			f.Metadata.CreatedAt.Format(time.RFC3339),
			f.Metadata.ValidatedAt.Format(time.RFC3339),
			f.Metadata.LastAccessed.Format(time.RFC3339),
			f.Metadata.AccessCount, f.Metadata.AccessScore,
		)
		if err != nil {
			return fmt.Errorf("failed to write triple %s: %w", f.ID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}

func (s *Store) WriteStatements(ctx context.Context, facts []semantic.StatementFact) error {
	ctx, span := s.tracer.Start(ctx, "sqlite.WriteStatements", trace.WithSpanKind(trace.SpanKindClient))
	defer span.End()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.Prepare(`
		INSERT INTO semantic_facts (id, type, statement, sources, tags, constraints, created_at, validated_at, last_accessed, access_count, access_score)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			statement=excluded.statement, sources=excluded.sources, tags=excluded.tags, constraints=excluded.constraints,
			created_at=excluded.created_at, validated_at=excluded.validated_at,
			last_accessed=excluded.last_accessed, access_count=excluded.access_count, access_score=excluded.access_score
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer func() { _ = stmt.Close() }()

	for _, f := range facts {
		sourcesJSON, _ := json.Marshal(f.Sources)
		tagsJSON, _ := json.Marshal(f.Tags)
		constraintsJSON, _ := json.Marshal(f.Metadata.Constraints)

		_, err := stmt.Exec(
			f.ID, semantic.FactTypeStatement, f.Statement,
			string(sourcesJSON), string(tagsJSON), string(constraintsJSON),
			f.Metadata.CreatedAt.Format(time.RFC3339),
			f.Metadata.ValidatedAt.Format(time.RFC3339),
			f.Metadata.LastAccessed.Format(time.RFC3339),
			f.Metadata.AccessCount, f.Metadata.AccessScore,
		)
		if err != nil {
			return fmt.Errorf("failed to write statement %s: %w", f.ID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}

func (s *Store) ReadFact(ctx context.Context, id string) (semantic.Fact, error) {
	ctx, span := s.tracer.Start(ctx, "sqlite.ReadFact", trace.WithSpanKind(trace.SpanKindClient))
	defer span.End()
	span.SetAttributes(attribute.String("fact.id", id))

	query := `SELECT id, type, entity, attribute, value, statement, sources, tags, constraints, created_at, validated_at, last_accessed, access_count, access_score FROM semantic_facts WHERE id = ?`
	row := s.db.QueryRowContext(ctx, query, id)

	fact, err := scanFact(row)
	if err == sql.ErrNoRows {
		span.SetStatus(codes.Error, "not found")
		return nil, semantic.ErrNotFound
	} else if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("failed to read fact: %w", err)
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
	ctx, span := s.tracer.Start(ctx, "sqlite.Search", trace.WithSpanKind(trace.SpanKindClient))
	defer span.End()

	conditions := []string{}
	args := []interface{}{}

	if query.Type != nil {
		conditions = append(conditions, "type = ?")
		args = append(args, *query.Type)
	}
	if query.Entity != nil {
		conditions = append(conditions, "entity = ?")
		args = append(args, *query.Entity)
	}
	if query.Attribute != nil {
		conditions = append(conditions, "attribute = ?")
		args = append(args, *query.Attribute)
	}
	if query.Value != nil {
		conditions = append(conditions, "value = ?")
		args = append(args, *query.Value)
	}
	if query.Statement != nil {
		conditions = append(conditions, "statement LIKE ?")
		args = append(args, "%"+*query.Statement+"%")
	}
	if query.Tag != nil {
		conditions = append(conditions, "tags LIKE ?")
		args = append(args, `%"`+*query.Tag+`"%`)
	}
	if query.Source != nil {
		conditions = append(conditions, "sources LIKE ?")
		args = append(args, `%"`+*query.Source+`"%`)
	}
	if len(query.Constraints) > 0 {
		for k, v := range query.Constraints {
			conditions = append(conditions, "constraints LIKE ?")
			args = append(args, `%"`+k+`":"`+v+`"%`)
		}
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	sortField := string(query.SortBy)
	if sortField == "" {
		sortField = "created_at"
	}
	sortOrder := string(query.SortOrder)
	if sortOrder == "" {
		sortOrder = "desc"
	}

	limit := query.Limit
	if limit <= 0 {
		limit = 50
	}
	offset := query.Offset

	sqlQuery := fmt.Sprintf(
		"SELECT id, type, entity, attribute, value, statement, sources, tags, constraints, created_at, validated_at, last_accessed, access_count, access_score FROM semantic_facts %s ORDER BY %s %s LIMIT ? OFFSET ?",
		whereClause, sortField, sortOrder,
	)
	args = append(args, limit, offset)

	rows, err := s.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("failed to search facts: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var facts []semantic.Fact
	for rows.Next() {
		fact, err := scanFact(rows)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, fmt.Errorf("failed to scan fact: %w", err)
		}
		facts = append(facts, fact)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return facts, nil
}

func (s *Store) DeleteFact(ctx context.Context, id string) error {
	ctx, span := s.tracer.Start(ctx, "sqlite.DeleteFact", trace.WithSpanKind(trace.SpanKindClient))
	defer span.End()
	span.SetAttributes(attribute.String("fact.id", id))

	_, err := s.db.ExecContext(ctx, "DELETE FROM semantic_facts WHERE id = ?", id)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("failed to delete fact: %w", err)
	}
	return nil
}

func (s *Store) DeleteByEntityAttribute(ctx context.Context, entity, attr string) error {
	ctx, span := s.tracer.Start(ctx, "sqlite.DeleteByEntityAttribute", trace.WithSpanKind(trace.SpanKindClient))
	defer span.End()
	span.SetAttributes(attribute.String("entity", entity), attribute.String("attribute", attr))

	_, err := s.db.ExecContext(ctx, "DELETE FROM semantic_facts WHERE entity = ? AND attribute = ?", entity, attr)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("failed to delete by entity/attribute: %w", err)
	}
	return nil
}

func (s *Store) WriteRelationship(ctx context.Context, rel semantic.Relationship) error {
	ctx, span := s.tracer.Start(ctx, "sqlite.WriteRelationship", trace.WithSpanKind(trace.SpanKindClient))
	defer span.End()
	span.SetAttributes(attribute.String("relationship.id", rel.ID))

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	metadataJSON, _ := json.Marshal(rel.Metadata)

	query := `
		INSERT INTO semantic_relationships (id, subject, predicate, object, created_at, metadata)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			subject=excluded.subject,
			predicate=excluded.predicate,
			object=excluded.object,
			created_at=excluded.created_at,
			metadata=excluded.metadata
	`
	_, err := s.db.ExecContext(ctx, query,
		rel.ID, rel.Subject, rel.Predicate, rel.Object,
		rel.Metadata.CreatedAt.Format(time.RFC3339),
		string(metadataJSON),
	)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("failed to write relationship: %w", err)
	}
	return nil
}

func (s *Store) ReadRelationships(ctx context.Context, subject string) ([]semantic.Relationship, error) {
	ctx, span := s.tracer.Start(ctx, "sqlite.ReadRelationships", trace.WithSpanKind(trace.SpanKindClient))
	defer span.End()
	span.SetAttributes(attribute.String("relationship.subject", subject))

	query := `SELECT id, subject, predicate, object, created_at, metadata FROM semantic_relationships WHERE subject = ?`
	rows, err := s.db.QueryContext(ctx, query, subject)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("failed to read relationships: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var rels []semantic.Relationship
	for rows.Next() {
		rel, err := scanRelationship(rows)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, fmt.Errorf("failed to scan relationship: %w", err)
		}
		rels = append(rels, rel)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return rels, nil
}

func (s *Store) DeleteRelationship(ctx context.Context, id string) error {
	ctx, span := s.tracer.Start(ctx, "sqlite.DeleteRelationship", trace.WithSpanKind(trace.SpanKindClient))
	defer span.End()
	span.SetAttributes(attribute.String("relationship.id", id))

	_, err := s.db.ExecContext(ctx, "DELETE FROM semantic_relationships WHERE id = ?", id)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("failed to delete relationship: %w", err)
	}
	return nil
}

func (s *Store) TraverseGraph(ctx context.Context, query semantic.GraphQuery) (semantic.GraphResult, error) {
	ctx, span := s.tracer.Start(ctx, "sqlite.TraverseGraph", trace.WithSpanKind(trace.SpanKindClient))
	defer span.End()
	span.SetAttributes(attribute.String("graph.start_entity", query.StartEntity), attribute.Int("graph.max_depth", query.MaxDepth))

	result := semantic.GraphResult{
		Nodes: []semantic.GraphNode{},
		Edges: []semantic.GraphEdge{},
	}

	visited := make(map[string]bool)
	queue := []string{query.StartEntity}
	depth := 0
	maxDepth := query.MaxDepth
	if maxDepth <= 0 {
		maxDepth = 3
	}
	maxResults := query.MaxResults
	if maxResults <= 0 {
		maxResults = 50
	}

	predicateFilter := len(query.Predicates) > 0

	for len(queue) > 0 && depth < maxDepth && len(result.Nodes) < maxResults {
		nextLevel := []string{}

		for _, entity := range queue {
			if visited[entity] {
				continue
			}
			visited[entity] = true

			facts, err := s.searchFactsByEntity(ctx, entity)
			if err == nil && len(facts) > 0 {
				result.Nodes = append(result.Nodes, semantic.GraphNode{
					Entity: entity,
					Facts:  facts,
				})
			}

			rels, err := s.getRelationshipsBySubject(ctx, entity)
			if err != nil {
				continue
			}

			for _, rel := range rels {
				if predicateFilter && !contains(query.Predicates, rel.Predicate) {
					continue
				}

				result.Edges = append(result.Edges, semantic.GraphEdge{
					Subject:   rel.Subject,
					Predicate: rel.Predicate,
					Object:    rel.Object,
				})

				if !visited[rel.Object] {
					nextLevel = append(nextLevel, rel.Object)
				}
			}
		}

		queue = nextLevel
		depth++
	}

	if len(result.Nodes) > maxResults {
		result.Nodes = result.Nodes[:maxResults]
	}

	return result, nil
}

func (s *Store) searchFactsByEntity(ctx context.Context, entity string) ([]semantic.Fact, error) {
	query := `SELECT id, type, entity, attribute, value, statement, sources, tags, constraints, created_at, validated_at, last_accessed, access_count, access_score FROM semantic_facts WHERE entity = ?`
	rows, err := s.db.QueryContext(ctx, query, entity)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var facts []semantic.Fact
	for rows.Next() {
		fact, err := scanFact(rows)
		if err != nil {
			return nil, err
		}
		facts = append(facts, fact)
	}
	return facts, rows.Err()
}

func (s *Store) getRelationshipsBySubject(ctx context.Context, subject string) ([]semantic.Relationship, error) {
	query := `SELECT id, subject, predicate, object, created_at, metadata FROM semantic_relationships WHERE subject = ?`
	rows, err := s.db.QueryContext(ctx, query, subject)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var rels []semantic.Relationship
	for rows.Next() {
		rel, err := scanRelationship(rows)
		if err != nil {
			return nil, err
		}
		rels = append(rels, rel)
	}
	return rels, rows.Err()
}

func (s *Store) Close() error {
	return s.db.Close()
}

type scanner interface {
	Scan(dest ...interface{}) error
}

func scanFact(s scanner) (semantic.Fact, error) {
	var id, factType string
	var entity, attribute, value, statement sql.NullString
	var sourcesJSON, tagsJSON, constraintsJSON string
	var createdAt, validatedAt, lastAccessed string
	var accessCount int
	var accessScore float64

	err := s.Scan(&id, &factType, &entity, &attribute, &value, &statement, &sourcesJSON, &tagsJSON, &constraintsJSON, &createdAt, &validatedAt, &lastAccessed, &accessCount, &accessScore)
	if err != nil {
		return nil, err
	}

	var sources []semantic.Source
	_ = json.Unmarshal([]byte(sourcesJSON), &sources)

	var tags []string
	_ = json.Unmarshal([]byte(tagsJSON), &tags)

	var constraintsMap map[string]string
	_ = json.Unmarshal([]byte(constraintsJSON), &constraintsMap)

	createdTime, _ := time.Parse(time.RFC3339, createdAt)
	validatedTime, _ := time.Parse(time.RFC3339, validatedAt)
	accessedTime, _ := time.Parse(time.RFC3339, lastAccessed)

	metadata := semantic.FactMetadata{
		CreatedAt:    createdTime,
		ValidatedAt:  validatedTime,
		LastAccessed: accessedTime,
		AccessCount:  accessCount,
		AccessScore:  accessScore,
		Constraints:  constraintsMap,
	}

	switch semantic.FactType(factType) {
	case semantic.FactTypeTriple:
		return semantic.TripleFact{
			ID:        id,
			Entity:    entity.String,
			Attribute: attribute.String,
			Value:     value.String,
			Sources:   sources,
			Tags:      tags,
			Metadata:  metadata,
		}, nil
	case semantic.FactTypeStatement:
		return semantic.StatementFact{
			ID:        id,
			Statement: statement.String,
			Sources:   sources,
			Tags:      tags,
			Metadata:  metadata,
		}, nil
	default:
		return nil, fmt.Errorf("unknown fact type: %s", factType)
	}
}

func scanRelationship(s scanner) (semantic.Relationship, error) {
	var id, subject, predicate, object, createdAt string
	var metadataJSON sql.NullString

	err := s.Scan(&id, &subject, &predicate, &object, &createdAt, &metadataJSON)
	if err != nil {
		return semantic.Relationship{}, err
	}

	var metadata semantic.RelationshipMetadata
	if metadataJSON.Valid {
		_ = json.Unmarshal([]byte(metadataJSON.String), &metadata)
	}

	createdTime, _ := time.Parse(time.RFC3339, createdAt)
	metadata.CreatedAt = createdTime

	return semantic.Relationship{
		ID:        id,
		Subject:   subject,
		Predicate: predicate,
		Object:    object,
		Metadata:  metadata,
	}, nil
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
