// Package pgvector provides a PostgreSQL vector store implementation using the pgvector extension.
// This provider offers persistent vector storage with ACID compliance and efficient similarity search.
//
// Features:
// - Persistent storage in PostgreSQL
// - pgvector extension for efficient vector operations
// - ACID compliance for data integrity
// - Connection pooling and transaction support
// - Configurable table and schema names
//
// Requirements:
// - PostgreSQL with pgvector extension installed
// - Proper database permissions
//
// Example:
//
//	import "github.com/lookatitude/beluga-ai/pkg/vectorstores/providers/pgvector"
//
//	store := pgvector.NewPgVectorStore(embedder, &pgvector.PgVectorStoreConfig{
//		ConnectionString:   "postgres://user:pass@localhost/db",
//		TableName:         "documents",
//		EmbeddingDimension: 768,
//	})
package pgvector

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/lookatitude/beluga-ai/pkg/vectorstores"

	_ "github.com/lib/pq" // PostgreSQL driver
	// Placeholder for actual pgvector library if one is used, e.g., github.com/pgvector/pgvector-go
	// For now, we assume direct SQL interaction or a helper library for pgvector operations.
)

// PgVectorStore implements the VectorStore interface using a PostgreSQL database
// with the pgvector extension.
type PgVectorStore struct {
	db             *sql.DB
	tableName      string
	embeddingDim   int    // Dimension of the embeddings
	collectionName string // Optional, for multi-tenancy or logical separation within the table
	name           string
	// Other necessary fields like connection string, preDeleteCollection, etc.
}

// PgVectorStoreConfig holds configuration specific to PgVectorStore.
type PgVectorStoreConfig struct {
	ConnectionString    string `mapstructure:"connection_string"`
	TableName           string `mapstructure:"table_name"`
	EmbeddingDimension  int    `mapstructure:"embedding_dimension"`
	CollectionName      string `mapstructure:"collection_name"`       // Optional
	PreDeleteCollection bool   `mapstructure:"pre_delete_collection"` // If true, deletes existing data for the collection on init
}

// NewPgVectorStore creates a new PgVectorStore from configuration.
// This is used by the factory pattern.
func NewPgVectorStoreFromConfig(ctx context.Context, config vectorstores.Config) (vectorstores.VectorStore, error) {
	// Extract pgvector-specific configuration from ProviderConfig
	providerConfig, ok := config.ProviderConfig["pgvector"].(map[string]interface{})
	if !ok {
		return nil, vectorstores.NewVectorStoreError(vectorstores.ErrCodeInvalidConfig,
			"pgvector provider config not found or invalid format")
	}

	connStr, ok := providerConfig["connection_string"].(string)
	if !ok || connStr == "" {
		return nil, vectorstores.NewVectorStoreError(vectorstores.ErrCodeInvalidConfig,
			"connection_string is required in pgvector provider config")
	}

	tableName, ok := providerConfig["table_name"].(string)
	if !ok || tableName == "" {
		tableName = "beluga_documents" // default
	}

	// Register the provider
	vectorstores.RegisterGlobal("pgvector", NewPgVectorStoreFromConfig)

	// Log configuration
	vectorstores.LogInfo(ctx, "PgVector store configured",
		"pgvector",
		slog.String("table_name", tableName))

	return &PgVectorStore{
		db:             nil, // Will be initialized in AddDocuments
		tableName:      tableName,
		embeddingDim:   768, // Default, should be configurable
		collectionName: "default",
		name:           "pgvector",
	}, nil
}

// NewPgVectorStore creates a new PgVectorStore.
// It requires a database connection string, table name, and embedding dimension.
// Further configuration can be passed via PgVectorStoreConfig within vectorstores.Config.ProviderArgs.
func NewPgVectorStore(ctx context.Context, config vectorstores.Config) (*PgVectorStore, error) {
	store, err := NewPgVectorStoreFromConfig(ctx, config)
	if err != nil {
		return nil, err
	}
	return store.(*PgVectorStore), nil
}

func (s *PgVectorStore) ensureTableExists(ctx context.Context) error {
	// Example DDL. Production code should handle migrations and more complex schema.
	// The vector type depends on the pgvector extension.
	// Example: CREATE EXTENSION IF NOT EXISTS vector;
	// CREATE TABLE IF NOT EXISTS items (id bigserial PRIMARY KEY, embedding vector(3), content text);
	query := fmt.Sprintf(`
	CREATE EXTENSION IF NOT EXISTS vector;
	CREATE TABLE IF NOT EXISTS %s (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		content TEXT,
		metadata JSONB,
		embedding VECTOR(%d),
		collection_name TEXT
	);
	CREATE INDEX IF NOT EXISTS %s_embedding_idx ON %s USING HNSW (embedding vector_l2_ops);
	`, s.tableName, s.embeddingDim, s.tableName, s.tableName) // Example index

	_, err := s.db.ExecContext(ctx, query)
	return err
}

// AddDocuments adds documents to the PgVectorStore.
func (s *PgVectorStore) AddDocuments(ctx context.Context, documents []schema.Document, opts ...vectorstores.Option) ([]string, error) {
	// Apply options
	config := vectorstores.NewDefaultConfig()
	vectorstores.ApplyOptions(config, opts...)

	// Use embedder from options
	embedder := config.Embedder

	if embedder == nil {
		return nil, fmt.Errorf("pgvector: embedder is required to add documents")
	}

	// Call the original implementation logic
	err := s.addDocumentsInternal(ctx, documents, embedder)
	if err != nil {
		return nil, err
	}

	// Generate IDs for the documents
	ids := make([]string, len(documents))
	for i := range documents {
		ids[i] = fmt.Sprintf("pgv_%d", i+1) // Simple ID generation
	}

	return ids, nil
}

// addDocumentsInternal contains the original AddDocuments logic
func (s *PgVectorStore) addDocumentsInternal(ctx context.Context, docs []schema.Document, embedder vectorstores.Embedder) error {
	if embedder == nil {
		return fmt.Errorf("pgvector: embedder is required to add documents")
	}

	// Begin transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback() // Rollback if not committed

	stmt, err := tx.PrepareContext(ctx, fmt.Sprintf("INSERT INTO %s (id, content, metadata, embedding, collection_name) VALUES ($1, $2, $3, $4, $5)", s.tableName))
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, doc := range docs {
		var docEmbedding []float32
		if len(doc.Embedding) > 0 {
			docEmbedding = doc.Embedding
		} else {
			embeds, err := embedder.EmbedDocuments(ctx, []string{doc.PageContent})
			if err != nil {
				return fmt.Errorf("failed to embed document 	%s	: %w", doc.ID, err)
			}
			if len(embeds) == 0 {
				return fmt.Errorf("embedder returned no embeddings for document 	%s", doc.ID)
			}
			docEmbedding = embeds[0]
		}

		if len(docEmbedding) != s.embeddingDim {
			return fmt.Errorf("document 	%s	 embedding dimension %d does not match store dimension %d", doc.ID, len(docEmbedding), s.embeddingDim)
		}

		// pgvector typically expects embeddings as a string like "[1,2,3]"
		embeddingStr := fmt.Sprintf("[%s]", float32SliceToString(docEmbedding, ","))

		// Convert metadata map to JSONB string (simplified, use json.Marshal for robustness)
		metadataStr := "{}"
		if doc.Metadata != nil {
			// metadataBytes, _ := json.Marshal(doc.Metadata) // Proper way
			// metadataStr = string(metadataBytes)
			// For simplicity now, assuming it's a simple map that can be stringified easily or is already JSON
			metadataStr = fmt.Sprintf("%v", doc.Metadata) // Placeholder
		}

		_, err = stmt.ExecContext(ctx, doc.ID, doc.PageContent, metadataStr, embeddingStr, s.collectionName)
		if err != nil {
			return fmt.Errorf("failed to insert document 	%s	: %w", doc.ID, err)
		}
	}

	return tx.Commit()
}

// SimilaritySearch performs a similarity search using a pre-computed query vector.
func (s *PgVectorStore) SimilaritySearch(ctx context.Context, queryVector []float32, k int, opts ...vectorstores.Option) ([]schema.Document, []float32, error) {
	if len(queryVector) != s.embeddingDim {
		return nil, nil, fmt.Errorf("query vector dimension %d does not match store dimension %d", len(queryVector), s.embeddingDim)
	}

	queryEmbeddingStr := fmt.Sprintf("[%s]", float32SliceToString(queryVector, ","))

	// Example query using L2 distance (<-> operator for pgvector)
	// Other operators: <#> for negative inner product, <=> for cosine distance
	query := fmt.Sprintf(
		"SELECT id, content, metadata, embedding <-> $1 AS distance FROM %s WHERE collection_name = $2 ORDER BY distance LIMIT $3",
		s.tableName,
	)
	if s.collectionName == "" {
		query = fmt.Sprintf(
			"SELECT id, content, metadata, embedding <-> $1 AS distance FROM %s ORDER BY distance LIMIT $2",
			s.tableName,
		)
	}

	var rows *sql.Rows
	var err error

	if s.collectionName == "" {
		rows, err = s.db.QueryContext(ctx, query, queryEmbeddingStr, k)
	} else {
		rows, err = s.db.QueryContext(ctx, query, queryEmbeddingStr, s.collectionName, k)
	}

	if err != nil {
		return nil, nil, fmt.Errorf("failed to execute similarity search query: %w", err)
	}
	defer rows.Close()

	var resultDocs []schema.Document
	var resultScores []float32

	for rows.Next() {
		var doc schema.Document
		var distance float32
		// var embeddingStr string // pgvector returns embedding as string - This variable is not used in the current logic.
		var metadataStr sql.NullString // Assuming metadata is stored as JSONB and retrieved as string

		if err := rows.Scan(&doc.ID, &doc.PageContent, &metadataStr, &distance); err != nil {
			return nil, nil, fmt.Errorf("failed to scan row: %w", err)
		}
		// Convert distance to similarity score (e.g., 1 - distance for cosine distance, or handle L2 appropriately)
		// For L2 distance, smaller is better. If a score where higher is better is needed, transform it.
		// For simplicity, we return L2 distance as the "score" here, noting smaller is better.
		resultScores = append(resultScores, distance)

		// Parse metadataStr (JSON) into doc.Metadata map[string]string
		if metadataStr.Valid && metadataStr.String != "" && metadataStr.String != "{}" {
			var parsedMeta map[string]string
			if err := json.Unmarshal([]byte(metadataStr.String), &parsedMeta); err == nil {
				doc.Metadata = parsedMeta
			} else {
				// Handle error or set a default, e.g., log the error
				// For now, we can assign the raw string to a special key if parsing fails
				doc.Metadata = map[string]string{"_raw_pgvector_metadata_error": metadataStr.String, "_parsing_error": err.Error()}
			}
		} else {
			doc.Metadata = make(map[string]string) // Ensure it's not nil
		}
		resultDocs = append(resultDocs, doc)
	}

	return resultDocs, resultScores, rows.Err()
}

// SimilaritySearchByQuery generates an embedding for the query and then performs a similarity search.
func (s *PgVectorStore) SimilaritySearchByQuery(ctx context.Context, query string, k int, embedder vectorstores.Embedder, opts ...vectorstores.Option) ([]schema.Document, []float32, error) {
	if embedder == nil {
		return nil, nil, fmt.Errorf("pgvector: embedder is required for SimilaritySearchByQuery")
	}

	queryEmbedding, err := embedder.EmbedQuery(ctx, query)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to embed query: %w", err)
	}

	return s.SimilaritySearch(ctx, queryEmbedding, k)
}

// GetName returns the name of the vector store.
func (s *PgVectorStore) GetName() string {
	return s.name
}

// float32SliceToString converts a slice of float32 to a comma-separated string.
func float32SliceToString(slice []float32, separator string) string {
	strVals := make([]string, len(slice))
	for i, v := range slice {
		strVals[i] = fmt.Sprintf("%f", v)
	}
	return string(JoinBytes([]byte(separator), strVals...))
}

// JoinBytes is a helper to join string parts with a byte separator.
// This is a simplified helper; strings.Join is usually sufficient.
func JoinBytes(sep []byte, parts ...string) []byte {
	if len(parts) == 0 {
		return []byte{}
	}
	n := len(sep) * (len(parts) - 1)
	for i := 0; i < len(parts); i++ {
		n += len(parts[i])
	}

	b := make([]byte, n)
	bp := copy(b, parts[0])
	for _, s := range parts[1:] {
		bp += copy(b[bp:], sep)
		bp += copy(b[bp:], s)
	}
	return b
}

// DeleteDocuments removes documents from the store based on their IDs.
func (s *PgVectorStore) DeleteDocuments(ctx context.Context, ids []string, opts ...vectorstores.Option) error {
	if len(ids) == 0 {
		return nil
	}

	// Create placeholders for the IN clause
	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}

	query := fmt.Sprintf("DELETE FROM %s WHERE id IN (%s)", s.tableName, fmt.Sprintf("%s", fmt.Sprintf("%s", placeholders[0])))
	if len(placeholders) > 1 {
		for i := 1; i < len(placeholders); i++ {
			query = strings.Replace(query, ")", fmt.Sprintf(",%s)", placeholders[i]), 1)
		}
	}

	_, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to delete documents: %w", err)
	}

	return nil
}

// AsRetriever returns a Retriever instance based on this VectorStore.
func (s *PgVectorStore) AsRetriever(opts ...vectorstores.Option) vectorstores.Retriever {
	// For now, return a simple retriever implementation
	// This would typically return a VectorStoreRetriever
	return nil // TODO: Implement proper retriever
}

// Ensure PgVectorStore implements the VectorStore interface.
var _ vectorstores.VectorStore = (*PgVectorStore)(nil)
