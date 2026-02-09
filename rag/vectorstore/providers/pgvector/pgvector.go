// Package pgvector provides a VectorStore backed by PostgreSQL with the
// pgvector extension. It uses pgx for connection management and supports
// cosine, dot-product, and Euclidean distance strategies.
//
// Registration:
//
//	import _ "github.com/lookatitude/beluga-ai/rag/vectorstore/providers/pgvector"
//
//	store, err := vectorstore.New("pgvector", config.ProviderConfig{
//	    BaseURL: "postgres://user:pass@localhost:5432/db",
//	    Options: map[string]any{
//	        "table":     "documents",
//	        "dimension": float64(1536),
//	    },
//	})
package pgvector

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/rag/vectorstore"
	"github.com/lookatitude/beluga-ai/schema"
	pgvec "github.com/pgvector/pgvector-go"
)

func init() {
	vectorstore.Register("pgvector", func(cfg config.ProviderConfig) (vectorstore.VectorStore, error) {
		return NewFromConfig(cfg)
	})
}

// Pool abstracts pgx pool operations for testability.
type Pool interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

// Store is a VectorStore backed by PostgreSQL with the pgvector extension.
type Store struct {
	pool      Pool
	table     string
	dimension int
}

// Compile-time interface check.
var _ vectorstore.VectorStore = (*Store)(nil)

// Option configures a Store.
type Option func(*Store)

// WithTable sets the table name. Defaults to "documents".
func WithTable(table string) Option {
	return func(s *Store) {
		s.table = table
	}
}

// WithDimension sets the vector dimension. Required for table creation.
func WithDimension(dim int) Option {
	return func(s *Store) {
		s.dimension = dim
	}
}

// New creates a new pgvector Store with the given pool and options.
func New(pool Pool, opts ...Option) *Store {
	s := &Store{
		pool:      pool,
		table:     "documents",
		dimension: 1536,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// NewFromConfig creates a Store from a ProviderConfig. The base_url field
// should contain the PostgreSQL connection string.
func NewFromConfig(cfg config.ProviderConfig) (*Store, error) {
	if cfg.BaseURL == "" {
		return nil, fmt.Errorf("pgvector: base_url (connection string) is required")
	}

	pool, err := pgx.Connect(context.Background(), cfg.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("pgvector: connect: %w", err)
	}

	var opts []Option
	if table, ok := config.GetOption[string](cfg, "table"); ok {
		opts = append(opts, WithTable(table))
	}
	if dim, ok := config.GetOption[float64](cfg, "dimension"); ok {
		opts = append(opts, WithDimension(int(dim)))
	}

	return New(&connWrapper{conn: pool}, opts...), nil
}

// connWrapper wraps pgx.Conn to implement Pool.
type connWrapper struct {
	conn *pgx.Conn
}

func (w *connWrapper) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	return w.conn.Exec(ctx, sql, args...)
}

func (w *connWrapper) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return w.conn.Query(ctx, sql, args...)
}

// EnsureTable creates the documents table and vector extension if they
// do not exist.
func (s *Store) EnsureTable(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, "CREATE EXTENSION IF NOT EXISTS vector")
	if err != nil {
		return fmt.Errorf("pgvector: create extension: %w", err)
	}

	query := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
		id TEXT PRIMARY KEY,
		embedding vector(%d),
		content TEXT,
		metadata JSONB
	)`, s.table, s.dimension)

	_, err = s.pool.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("pgvector: create table: %w", err)
	}
	return nil
}

// Add inserts documents with their embeddings into the store. The docs and
// embeddings slices must have the same length.
func (s *Store) Add(ctx context.Context, docs []schema.Document, embeddings [][]float32) error {
	if len(docs) != len(embeddings) {
		return fmt.Errorf("pgvector: docs length (%d) != embeddings length (%d)", len(docs), len(embeddings))
	}

	for i, doc := range docs {
		metaJSON, err := json.Marshal(doc.Metadata)
		if err != nil {
			return fmt.Errorf("pgvector: marshal metadata for %s: %w", doc.ID, err)
		}

		query := fmt.Sprintf(
			`INSERT INTO %s (id, embedding, content, metadata) VALUES ($1, $2, $3, $4)
			ON CONFLICT (id) DO UPDATE SET embedding = $2, content = $3, metadata = $4`,
			s.table,
		)

		vec := pgvec.NewVector(embeddings[i])
		_, err = s.pool.Exec(ctx, query, doc.ID, vec, doc.Content, metaJSON)
		if err != nil {
			return fmt.Errorf("pgvector: insert %s: %w", doc.ID, err)
		}
	}
	return nil
}

// Search finds the k most similar documents to the query vector. Results
// are returned in descending order of similarity with their Score field populated.
func (s *Store) Search(ctx context.Context, query []float32, k int, opts ...vectorstore.SearchOption) ([]schema.Document, error) {
	cfg := &vectorstore.SearchConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	op := distanceOperator(cfg.Strategy)

	// Build WHERE clause for metadata filters.
	var conditions []string
	var args []any
	argIdx := 3 // $1=query, $2=k

	for key, val := range cfg.Filter {
		argIdx++
		conditions = append(conditions, fmt.Sprintf("metadata->>$%d = $%d", argIdx-1, argIdx))
		args = append(args, key, fmt.Sprintf("%v", val))
		argIdx++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Build the score expression. For cosine (<=>), the operator returns
	// distance so we convert to similarity: 1 - distance.
	// For dot product (<#>), it returns negative inner product, so: -1 * distance.
	// For euclidean (<->), it returns distance.
	var scoreExpr string
	switch cfg.Strategy {
	case vectorstore.DotProduct:
		scoreExpr = fmt.Sprintf("(embedding %s $1) * -1", op)
	case vectorstore.Euclidean:
		scoreExpr = fmt.Sprintf("(embedding %s $1) * -1", op)
	default: // Cosine
		scoreExpr = fmt.Sprintf("1 - (embedding %s $1)", op)
	}

	sql := fmt.Sprintf(
		`SELECT id, content, metadata, %s AS score FROM %s %s ORDER BY embedding %s $1 LIMIT $2`,
		scoreExpr, s.table, whereClause, op,
	)

	vec := pgvec.NewVector(query)
	allArgs := append([]any{vec, k}, args...)

	rows, err := s.pool.Query(ctx, sql, allArgs...)
	if err != nil {
		return nil, fmt.Errorf("pgvector: search: %w", err)
	}
	defer rows.Close()

	var results []schema.Document
	for rows.Next() {
		var doc schema.Document
		var metaJSON []byte

		if err := rows.Scan(&doc.ID, &doc.Content, &metaJSON, &doc.Score); err != nil {
			return nil, fmt.Errorf("pgvector: scan: %w", err)
		}

		if len(metaJSON) > 0 {
			if err := json.Unmarshal(metaJSON, &doc.Metadata); err != nil {
				return nil, fmt.Errorf("pgvector: unmarshal metadata: %w", err)
			}
		}

		if cfg.Threshold > 0 && doc.Score < cfg.Threshold {
			continue
		}

		results = append(results, doc)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("pgvector: rows: %w", err)
	}

	return results, nil
}

// Delete removes documents with the given IDs from the store.
func (s *Store) Delete(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	placeholders := make([]string, len(ids))
	args := make([]any, len(ids))
	for i, id := range ids {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}

	query := fmt.Sprintf("DELETE FROM %s WHERE id IN (%s)", s.table, strings.Join(placeholders, ", "))
	_, err := s.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("pgvector: delete: %w", err)
	}
	return nil
}

// distanceOperator returns the pgvector SQL operator for the given strategy.
func distanceOperator(strategy vectorstore.SearchStrategy) string {
	switch strategy {
	case vectorstore.DotProduct:
		return "<#>"
	case vectorstore.Euclidean:
		return "<->"
	default: // Cosine
		return "<=>"
	}
}
