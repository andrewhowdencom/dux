package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/andrewhowdencom/dux/pkg/memory/semantic"
	_ "github.com/mattn/go-sqlite3" // sqlite driver
)

// Store implements semantic.Store using an SQLite backend.
type Store struct {
	db *sql.DB
}

// NewStore creates a new SQLite-backed semantic store and initializes the schema.
func NewStore(dbParams string) (*Store, error) {
	db, err := sql.Open("sqlite3", dbParams)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	if err := migrate(db); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return &Store{db: db}, nil
}

func migrate(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS semmem_facts (
		entity TEXT NOT NULL,
		attribute TEXT NOT NULL,
		value TEXT NOT NULL,
		PRIMARY KEY (entity, attribute)
	);
	CREATE INDEX IF NOT EXISTS idx_semmem_facts_attr_val ON semmem_facts (attribute, value);
	`
	_, err := db.Exec(schema)
	return err
}

// Write implements semantic.Store
func (s *Store) Write(ctx context.Context, fact semantic.Fact) error {
	query := `
		INSERT INTO semmem_facts (entity, attribute, value)
		VALUES (?, ?, ?)
		ON CONFLICT(entity, attribute) DO UPDATE SET value=excluded.value;
	`
	_, err := s.db.ExecContext(ctx, query, fact.Entity, fact.Attribute, fact.Value)
	if err != nil {
		return fmt.Errorf("failed to write fact: %w", err)
	}
	return nil
}

// Read implements semantic.Store
func (s *Store) Read(ctx context.Context, entity, attribute string) (semantic.Fact, error) {
	query := `SELECT entity, attribute, value FROM semmem_facts WHERE entity = ? AND attribute = ?`
	row := s.db.QueryRowContext(ctx, query, entity, attribute)

	var fact semantic.Fact
	err := row.Scan(&fact.Entity, &fact.Attribute, &fact.Value)
	if err == sql.ErrNoRows {
		return semantic.Fact{}, fmt.Errorf("fact not found")
	} else if err != nil {
		return semantic.Fact{}, fmt.Errorf("failed to read fact: %w", err)
	}

	return fact, nil
}

// Search implements semantic.Store
func (s *Store) Search(ctx context.Context, attribute, value string) ([]semantic.Fact, error) {
	query := `SELECT entity, attribute, value FROM semmem_facts WHERE attribute = ? AND value = ?`
	rows, err := s.db.QueryContext(ctx, query, attribute, value)
	if err != nil {
		return nil, fmt.Errorf("failed to search facts: %w", err)
	}
	defer rows.Close()

	var facts []semantic.Fact
	for rows.Next() {
		var fact semantic.Fact
		if err := rows.Scan(&fact.Entity, &fact.Attribute, &fact.Value); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		facts = append(facts, fact)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return facts, nil
}

// Delete implements semantic.Store
func (s *Store) Delete(ctx context.Context, entity, attribute string) error {
	query := `DELETE FROM semmem_facts WHERE entity = ? AND attribute = ?`
	_, err := s.db.ExecContext(ctx, query, entity, attribute)
	if err != nil {
		return fmt.Errorf("failed to delete fact: %w", err)
	}
	return nil
}

// Close closes the underlying database connection.
func (s *Store) Close() error {
	return s.db.Close()
}
