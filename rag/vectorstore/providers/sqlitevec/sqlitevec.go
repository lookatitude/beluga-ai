//go:build cgo

// Package sqlitevec provides a VectorStore backed by SQLite with the
// sqlite-vec extension for vector similarity search.
//
// This provider requires CGO and the sqlite-vec extension. Build with:
//
//	CGO_ENABLED=1 go build
//
// Registration:
//
//	import _ "github.com/lookatitude/beluga-ai/rag/vectorstore/providers/sqlitevec"
//
//	store, err := vectorstore.New("sqlitevec", config.ProviderConfig{
//	    BaseURL: "/path/to/database.db",
//	    Options: map[string]any{
//	        "table":     "documents",
//	        "dimension": float64(1536),
//	    },
//	})
package sqlitevec

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	sqlite_vec "github.com/asg017/sqlite-vec-go-bindings/cgo"
	_ "github.com/mattn/go-sqlite3"

	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/rag/vectorstore"
	"github.com/lookatitude/beluga-ai/schema"
)

func init() {
	sqlite_vec.Auto()
	vectorstore.Register("sqlitevec", func(cfg config.ProviderConfig) (vectorstore.VectorStore, error) {
		return NewFromConfig(cfg)
	})
}

// DB abstracts the database interface for testability.
type DB interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

// Store is a VectorStore backed by SQLite with sqlite-vec.
type Store struct {
	db        DB
	table     string
	dimension int
}

// Compile-time interface check.
var _ vectorstore.VectorStore = (*Store)(nil)

// Option configures a Store.
type Option func(*Store)

// WithTable sets the table name.
func WithTable(name string) Option {
	return func(s *Store) { s.table = name }
}

// WithDimension sets the vector dimension.
func WithDimension(dim int) Option {
	return func(s *Store) { s.dimension = dim }
}

// WithDB sets a custom database connection.
func WithDB(db DB) Option {
	return func(s *Store) { s.db = db }
}

// New creates a new SQLite-vec Store.
func New(opts ...Option) (*Store, error) {
	s := &Store{
		table:     "documents",
		dimension: 1536,
	}
	for _, opt := range opts {
		opt(s)
	}
	if s.db == nil {
		return nil, fmt.Errorf("sqlitevec: database connection is required")
	}
	return s, nil
}

// NewFromConfig creates a Store from a ProviderConfig.
func NewFromConfig(cfg config.ProviderConfig) (*Store, error) {
	if cfg.BaseURL == "" {
		return nil, fmt.Errorf("sqlitevec: base_url (database path) is required")
	}

	db, err := sql.Open("sqlite3", cfg.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("sqlitevec: open database: %w", err)
	}

	var opts []Option
	opts = append(opts, WithDB(db))

	if tbl, ok := config.GetOption[string](cfg, "table"); ok {
		opts = append(opts, WithTable(tbl))
	}
	if dim, ok := config.GetOption[float64](cfg, "dimension"); ok {
		opts = append(opts, WithDimension(int(dim)))
	}

	return New(opts...)
}

// EnsureTable creates the documents and vec_documents tables.
func (s *Store) EnsureTable(ctx context.Context) error {
	// Create the metadata table.
	metaSQL := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
		id TEXT PRIMARY KEY,
		content TEXT NOT NULL,
		metadata TEXT
	)`, s.table)
	if _, err := s.db.ExecContext(ctx, metaSQL); err != nil {
		return fmt.Errorf("sqlitevec: create metadata table: %w", err)
	}

	// Create the virtual vector table.
	vecSQL := fmt.Sprintf(`CREATE VIRTUAL TABLE IF NOT EXISTS vec_%s USING vec0(
		id TEXT PRIMARY KEY,
		embedding float[%d]
	)`, s.table, s.dimension)
	if _, err := s.db.ExecContext(ctx, vecSQL); err != nil {
		return fmt.Errorf("sqlitevec: create vec table: %w", err)
	}

	return nil
}

// Add inserts documents with their embeddings into the store.
func (s *Store) Add(ctx context.Context, docs []schema.Document, embeddings [][]float32) error {
	if len(docs) != len(embeddings) {
		return fmt.Errorf("sqlitevec: docs length (%d) != embeddings length (%d)", len(docs), len(embeddings))
	}

	for i, doc := range docs {
		// Serialize metadata.
		var metaJSON string
		if doc.Metadata != nil {
			b, err := json.Marshal(doc.Metadata)
			if err != nil {
				return fmt.Errorf("sqlitevec: marshal metadata: %w", err)
			}
			metaJSON = string(b)
		}

		// Upsert into metadata table.
		metaSQL := fmt.Sprintf(
			`INSERT OR REPLACE INTO %s (id, content, metadata) VALUES (?, ?, ?)`,
			s.table,
		)
		if _, err := s.db.ExecContext(ctx, metaSQL, doc.ID, doc.Content, metaJSON); err != nil {
			return fmt.Errorf("sqlitevec: insert metadata: %w", err)
		}

		// Serialize embedding as binary blob for sqlite-vec.
		embBlob, err := sqlite_vec.SerializeFloat32(embeddings[i])
		if err != nil {
			return fmt.Errorf("sqlitevec: serialize embedding: %w", err)
		}

		// Delete existing vector entry (vec0 doesn't support INSERT OR REPLACE).
		delSQL := fmt.Sprintf(`DELETE FROM vec_%s WHERE id = ?`, s.table)
		if _, err := s.db.ExecContext(ctx, delSQL, doc.ID); err != nil {
			return fmt.Errorf("sqlitevec: delete old embedding: %w", err)
		}

		// Insert into vector table.
		vecSQL := fmt.Sprintf(
			`INSERT INTO vec_%s (id, embedding) VALUES (?, ?)`,
			s.table,
		)
		if _, err := s.db.ExecContext(ctx, vecSQL, doc.ID, embBlob); err != nil {
			return fmt.Errorf("sqlitevec: insert embedding: %w", err)
		}
	}

	return nil
}

// Search finds the k most similar documents to the query vector.
func (s *Store) Search(ctx context.Context, query []float32, k int, opts ...vectorstore.SearchOption) ([]schema.Document, error) {
	cfg := &vectorstore.SearchConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	// Serialize query vector as binary blob for sqlite-vec.
	queryBlob, err := sqlite_vec.SerializeFloat32(query)
	if err != nil {
		return nil, fmt.Errorf("sqlitevec: serialize query: %w", err)
	}

	// sqlite-vec uses L2 distance by default; lower = more similar.
	// Score = 1 / (1 + distance) to convert to similarity.
	// vec0 requires 'k = ?' constraint instead of LIMIT.
	searchSQL := fmt.Sprintf(`
		SELECT v.id, v.distance, d.content, d.metadata
		FROM vec_%s v
		JOIN %s d ON v.id = d.id
		WHERE v.embedding MATCH ?
		AND k = ?
		ORDER BY v.distance
	`, s.table, s.table)

	rows, err := s.db.QueryContext(ctx, searchSQL, queryBlob, k)
	if err != nil {
		return nil, fmt.Errorf("sqlitevec: search: %w", err)
	}
	defer rows.Close()

	var docs []schema.Document
	for rows.Next() {
		var (
			id       string
			distance float64
			content  string
			metaJSON sql.NullString
		)
		if err := rows.Scan(&id, &distance, &content, &metaJSON); err != nil {
			return nil, fmt.Errorf("sqlitevec: scan result: %w", err)
		}

		score := 1.0 / (1.0 + distance)
		if cfg.Threshold > 0 && score < cfg.Threshold {
			continue
		}

		doc := schema.Document{
			ID:      id,
			Content: content,
			Score:   score,
		}

		if metaJSON.Valid && metaJSON.String != "" {
			var meta map[string]any
			if err := json.Unmarshal([]byte(metaJSON.String), &meta); err == nil {
				doc.Metadata = meta
			}
		}

		// Apply metadata filter.
		if len(cfg.Filter) > 0 && !matchesFilter(doc.Metadata, cfg.Filter) {
			continue
		}

		docs = append(docs, doc)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("sqlitevec: rows iteration: %w", err)
	}

	return docs, nil
}

// Delete removes documents with the given IDs from the store.
func (s *Store) Delete(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	placeholders := make([]string, len(ids))
	args := make([]any, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}
	ph := strings.Join(placeholders, ",")

	// Delete from metadata table.
	metaSQL := fmt.Sprintf("DELETE FROM %s WHERE id IN (%s)", s.table, ph)
	if _, err := s.db.ExecContext(ctx, metaSQL, args...); err != nil {
		return fmt.Errorf("sqlitevec: delete metadata: %w", err)
	}

	// Delete from vector table.
	vecSQL := fmt.Sprintf("DELETE FROM vec_%s WHERE id IN (%s)", s.table, ph)
	if _, err := s.db.ExecContext(ctx, vecSQL, args...); err != nil {
		return fmt.Errorf("sqlitevec: delete embeddings: %w", err)
	}

	return nil
}

// matchesFilter checks if metadata matches all filter key-value pairs.
func matchesFilter(metadata map[string]any, filter map[string]any) bool {
	if metadata == nil {
		return false
	}
	for k, v := range filter {
		if mv, ok := metadata[k]; !ok || fmt.Sprintf("%v", mv) != fmt.Sprintf("%v", v) {
			return false
		}
	}
	return true
}
