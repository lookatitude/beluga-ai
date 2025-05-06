// Package vectorstores provides implementations of the rag.VectorStore interface.
package vectorstores

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lookatitude/beluga-ai/core"
	"github.com/lookatitude/beluga-ai/rag"
	"github.com/lookatitude/beluga-ai/schema"
	"github.com/pgvector/pgvector-go"
)

// PgVectorStore implements the rag.VectorStore interface using PostgreSQL with the pgvector extension.
// It requires a pgx connection pool and an Embedder.
type PgVectorStore struct {
	pool           *pgxpool.Pool
	Embedder       rag.Embedder // Mandatory embedder
	TableName      string       // Name of the table storing vectors
	VectorColumn   string       // Name of the vector column
	ContentColumn  string       // Name of the document content column
	MetadataColumn string       // Name of the JSONB metadata column
	IDColumn       string       // Name of the primary key ID column (usually UUID)
	preDeleteStmt  string       // Prepared statement name for deletion
	preInsertStmt  string       // Prepared statement name for insertion

	initOnce sync.Once
	initErr  error
}

// PgVectorStoreOptions contains configuration options for creating a PgVectorStore.
type PgVectorStoreOptions struct {
	Pool           *pgxpool.Pool // Mandatory: Existing pgx connection pool
	Embedder       rag.Embedder  // Mandatory: Embedder for documents and queries
	TableName      string        // Optional: Defaults to "beluga_documents"
	VectorColumn   string        // Optional: Defaults to "embedding"
	ContentColumn  string        // Optional: Defaults to "content"
	MetadataColumn string        // Optional: Defaults to "metadata"
	IDColumn       string        // Optional: Defaults to "id"
}

// NewPgVectorStore creates a new PgVectorStore.
func NewPgVectorStore(ctx context.Context, opts PgVectorStoreOptions) (*PgVectorStore, error) {
	if opts.Pool == nil {
		return nil, errors.New("pgx connection pool is required")
	}
	if opts.Embedder == nil {
		return nil, errors.New("embedder is required")
	}

	store := &PgVectorStore{
		pool:           opts.Pool,
		Embedder:       opts.Embedder,
		TableName:      "beluga_documents",
		VectorColumn:   "embedding",
		ContentColumn:  "content",
		MetadataColumn: "metadata",
		IDColumn:       "id",
	}

	if opts.TableName != "" {
		store.TableName = opts.TableName
	}
	if opts.VectorColumn != "" {
		store.VectorColumn = opts.VectorColumn
	}
	if opts.ContentColumn != "" {
		store.ContentColumn = opts.ContentColumn
	}
	if opts.MetadataColumn != "" {
		store.MetadataColumn = opts.MetadataColumn
	}
	if opts.IDColumn != "" {
		store.IDColumn = opts.IDColumn
	}

	// Initialize lazily on first use to ensure table exists etc.
	// _, err := store.initialize(ctx)
	// if err != nil {
	// 	 return nil, err
	// }

	return store, nil
}

// initialize ensures the vector extension is registered, the table exists, and prepares statements.
func (s *PgVectorStore) initialize(ctx context.Context) (bool, error) {
	s.initOnce.Do(func() {
		// Register the vector type with pgx
		conn, err := s.pool.Acquire(ctx)
		if err != nil {
			s.initErr = fmt.Errorf("failed to acquire connection for type registration: %w", err)
			return
		}
		defer conn.Release()

		pgvector.Register(conn.Conn().TypeMap())

		// Determine vector dimensions from embedder (embed a dummy query)
		dummyEmbedding, err := s.Embedder.EmbedQuery(ctx, "dimension_check")
		if err != nil {
			s.initErr = fmt.Errorf("failed to determine embedding dimension: %w", err)
			return
		}
		vectorDim := len(dummyEmbedding)
		if vectorDim == 0 {
			s.initErr = errors.New("embedder returned zero-dimension vector")
			return
		}

		// Create table if not exists
		createTableSQL := fmt.Sprintf(`
	 	 	 CREATE EXTENSION IF NOT EXISTS vector;
	 	 	 CREATE TABLE IF NOT EXISTS %s (
	 	 	 	 %s UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	 	 	 	 %s TEXT,
	 	 	 	 %s JSONB,
	 	 	 	 %s vector(%d)
	 	 	 );
	 	 `, pgx.Identifier{s.TableName}.Sanitize(),
			pgx.Identifier{s.IDColumn}.Sanitize(),
			pgx.Identifier{s.ContentColumn}.Sanitize(),
			pgx.Identifier{s.MetadataColumn}.Sanitize(),
			pgx.Identifier{s.VectorColumn}.Sanitize(),
			vectorDim)

		_, err = s.pool.Exec(ctx, createTableSQL)
		if err != nil {
			s.initErr = fmt.Errorf("failed to create table %s: %w", s.TableName, err)
			return
		}

		// Prepare statements
		s.preDeleteStmt = fmt.Sprintf("pgvector_delete_%s", s.TableName)
		deleteSQL := fmt.Sprintf(`DELETE FROM %s WHERE %s = $1`,
			pgx.Identifier{s.TableName}.Sanitize(),
			pgx.Identifier{s.IDColumn}.Sanitize())
		_, err = conn.Conn().Prepare(ctx, s.preDeleteStmt, deleteSQL)
		if err != nil {
			// Ignore if already prepared (code 26000)
			var pgErr *pgconn.PgError
			if !errors.As(err, &pgErr) || pgErr.Code != "42P07" { // 42P07 = duplicate_prepared_statement
				s.initErr = fmt.Errorf("failed to prepare delete statement: %w", err)
				return
			}
		}

		s.preInsertStmt = fmt.Sprintf("pgvector_insert_%s", s.TableName)
		insertSQL := fmt.Sprintf(`INSERT INTO %s (%s, %s, %s) VALUES ($1, $2, $3) RETURNING %s`,
			pgx.Identifier{s.TableName}.Sanitize(),
			pgx.Identifier{s.ContentColumn}.Sanitize(),
			pgx.Identifier{s.MetadataColumn}.Sanitize(),
			pgx.Identifier{s.VectorColumn}.Sanitize(),
			pgx.Identifier{s.IDColumn}.Sanitize())
		_, err = conn.Conn().Prepare(ctx, s.preInsertStmt, insertSQL)
		if err != nil {
			var pgErr *pgconn.PgError
			if !errors.As(err, &pgErr) || pgErr.Code != "42P07" {
				s.initErr = fmt.Errorf("failed to prepare insert statement: %w", err)
				return
			}
		}
	})
	return s.initErr == nil, s.initErr
}

// getEmbedder resolves the embedder to use.
func (s *PgVectorStore) getEmbedder(options ...core.Option) (rag.Embedder, error) {
	config := make(map[string]any)
	for _, opt := range options {
		opt.Apply(&config)
	}
	if embedderOpt, ok := config["embedder"].(rag.Embedder); ok {
		return embedderOpt, nil
	}
	if s.Embedder != nil {
		return s.Embedder, nil
	}
	return nil, errors.New("embedder must be provided either during PgVectorStore creation or via WithEmbedder option")
}

// AddDocuments embeds and stores documents in the PostgreSQL table.
func (s *PgVectorStore) AddDocuments(ctx context.Context, documents []schema.Document, options ...core.Option) ([]string, error) {
	ok, err := s.initialize(ctx)
	if !ok {
		return nil, fmt.Errorf("initialization failed: %w", err)
	}

	embedder, err := s.getEmbedder(options...)
	if err != nil {
		return nil, err
	}

	texts := make([]string, len(documents))
	for i, doc := range documents {
		texts[i] = doc.GetContent()
	}

	embeddings, err := embedder.EmbedDocuments(ctx, texts)
	if err != nil {
		return nil, fmt.Errorf("failed to embed documents: %w", err)
	}

	if len(embeddings) != len(documents) {
		return nil, fmt.Errorf("number of embeddings (%d) does not match number of documents (%d)", len(embeddings), len(documents))
	}

	ids := make([]string, len(documents))
	batch := &pgx.Batch{}
	results := make([]pgx.BatchResult, len(documents))

	for i, doc := range documents {
		metadataJSON, err := json.Marshal(doc.GetMetadata())
		if err != nil {
			return nil, fmt.Errorf("failed to marshal metadata for document %d: %w", i, err)
		}
		batch.Queue(s.preInsertStmt, doc.GetContent(), metadataJSON, pgvector.NewVector(embeddings[i]))
	}

	br := s.pool.SendBatch(ctx, batch)
	defer br.Close()

	for i := 0; i < len(documents); i++ {
		var returnedID pgtype.UUID
		err = br.QueryRow().Scan(&returnedID)
		if err != nil {
			return nil, fmt.Errorf("failed to insert document %d: %w", i, err)
		}
		if returnedID.Valid {
			// Convert [16]byte UUID to string
			ids[i] = fmt.Sprintf("%x-%x-%x-%x-%x",
				returnedID.Bytes[0:4], returnedID.Bytes[4:6], returnedID.Bytes[6:8], returnedID.Bytes[8:10], returnedID.Bytes[10:16])
		} else {
			return nil, fmt.Errorf("insert for document %d did not return a valid ID", i)
		}
	}

	return ids, nil
}

// DeleteDocuments removes documents by ID.
func (s *PgVectorStore) DeleteDocuments(ctx context.Context, ids []string, options ...core.Option) error {
	ok, err := s.initialize(ctx)
	if !ok {
		return fmt.Errorf("initialization failed: %w", err)
	}

	batch := &pgx.Batch{}
	for _, idStr := range ids {
		// Parse string ID to UUID bytes
		var uuidBytes [16]byte
		_, err := fmt.Sscan(strings.ReplaceAll(idStr, "-", ""), fmt.Sprintf("%%32x", &uuidBytes))
		if err != nil {
			return fmt.Errorf("invalid UUID format for ID 	%s	: %w", idStr, err)
		}
		pgUUID := pgtype.UUID{Bytes: uuidBytes, Valid: true}
		batch.Queue(s.preDeleteStmt, pgUUID)
	}

	br := s.pool.SendBatch(ctx, batch)
	defer br.Close()

	var firstErr error
	for i := 0; i < len(ids); i++ {
		ct, err := br.Exec()
		if err != nil && firstErr == nil {
			firstErr = fmt.Errorf("failed to delete document with id %s: %w", ids[i], err)
		} else if err == nil && ct.RowsAffected() == 0 {
			// Optionally warn or error if ID wasn't found
			fmt.Printf("Warning: Document with ID %s not found for deletion.\n", ids[i])
		}
	}

	return firstErr
}

// similaritySearchInternal performs the core similarity search logic.
func (s *PgVectorStore) similaritySearchInternal(ctx context.Context, queryEmbedding []float32, k int, options ...core.Option) ([]schema.Document, error) {
	ok, err := s.initialize(ctx)
	if !ok {
		return nil, fmt.Errorf("initialization failed: %w", err)
	}

	config := make(map[string]any)
	for _, opt := range options {
		opt.Apply(&config)
	}
	// TODO: Implement metadata filtering
	// metadataFilter, _ := config["metadata_filter"].(map[string]any)
	// TODO: Implement score threshold (needs distance calculation in SQL)
	// scoreThreshold, _ := config["score_threshold"].(float32)

	// Using cosine distance (<=>) - smaller is better (0=identical, 1=orthogonal, 2=opposite)
	// Other options: L2 distance (<->), inner product (<#>) - larger is better
	distanceOperator := "<=>"

	querySQL := fmt.Sprintf(`
	 	 SELECT %s, %s, %s %s $1 AS distance
	 	 FROM %s
	 	 ORDER BY distance ASC
	 	 LIMIT $2
	 `, pgx.Identifier{s.IDColumn}.Sanitize(),
		pgx.Identifier{s.ContentColumn}.Sanitize(),
		pgx.Identifier{s.MetadataColumn}.Sanitize(),
		pgx.Identifier{s.VectorColumn}.Sanitize(),
		pgx.Identifier{s.TableName}.Sanitize())

	rows, err := s.pool.Query(ctx, querySQL, pgvector.NewVector(queryEmbedding), k)
	if err != nil {
		return nil, fmt.Errorf("failed to execute similarity search query: %w", err)
	}
	defer rows.Close()

	docs := make([]schema.Document, 0, k)
	for rows.Next() {
		var id pgtype.UUID
		var content string
		var metadataJSON []byte
		var distance float64 // pgvector distance operators return float64

		err := rows.Scan(&id, &content, &metadataJSON, &distance)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		metadata := make(map[string]any)
		if len(metadataJSON) > 0 {
			err = json.Unmarshal(metadataJSON, &metadata)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
			}
		}

		// Optionally add distance/score to metadata
		// Cosine distance = 1 - cosine similarity. Score = 1 - distance.
		metadata["score"] = 1.0 - distance

		docs = append(docs, schema.NewDocument(content, metadata))
	}

	if rows.Err() != nil {
		return nil, fmt.Errorf("error iterating rows: %w", rows.Err())
	}

	return docs, nil
}

// SimilaritySearch performs search using a query string.
func (s *PgVectorStore) SimilaritySearch(ctx context.Context, query string, k int, options ...core.Option) ([]schema.Document, error) {
	embedder, err := s.getEmbedder(options...)
	if err != nil {
		return nil, err
	}

	queryEmbedding, err := embedder.EmbedQuery(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to embed query: %w", err)
	}

	return s.similaritySearchInternal(ctx, queryEmbedding, k, options...)
}

// SimilaritySearchByVector performs search using a pre-computed vector.
func (s *PgVectorStore) SimilaritySearchByVector(ctx context.Context, embedding []float32, k int, options ...core.Option) ([]schema.Document, error) {
	return s.similaritySearchInternal(ctx, embedding, k, options...)
}

// AsRetriever returns a Retriever instance based on this VectorStore.
func (s *PgVectorStore) AsRetriever(options ...core.Option) rag.Retriever {
	return NewVectorStoreRetriever(s, options...)
}

// Ensure PgVectorStore implements the interface.
var _ rag.VectorStore = (*PgVectorStore)(nil)
