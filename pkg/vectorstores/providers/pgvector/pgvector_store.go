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
	vectorstoresiface "github.com/lookatitude/beluga-ai/pkg/vectorstores/iface"

	_ "github.com/lib/pq" // PostgreSQL driver
	// Placeholder for actual pgvector library if one is used, e.g., github.com/pgvector/pgvector-go
	// For now, we assume direct SQL interaction or a helper library for pgvector operations.
)

// PgVectorStore implements the VectorStore interface using a PostgreSQL database
// with the pgvector extension.
type PgVectorStore struct {
	db             *sql.DB
	tableName      string
	collectionName string
	name           string
	embeddingDim   int
}

// PgVectorStoreConfig holds configuration specific to PgVectorStore.
type PgVectorStoreConfig struct {
	ConnectionString    string `mapstructure:"connection_string"`
	TableName           string `mapstructure:"table_name"`
	CollectionName      string `mapstructure:"collection_name"`
	EmbeddingDimension  int    `mapstructure:"embedding_dimension"`
	PreDeleteCollection bool   `mapstructure:"pre_delete_collection"`
}

// NewPgVectorStoreFromConfig creates a new PgVectorStore from configuration.
// This is used by the factory pattern.
func NewPgVectorStoreFromConfig(ctx context.Context, config vectorstoresiface.Config) (vectorstores.VectorStore, error) {
	// Extract pgvector-specific configuration from ProviderConfig
	providerConfig, ok := config.ProviderConfig["pgvector"]
	if !ok {
		providerConfig = make(map[string]any)
	}

	// Extract connection parameters with defaults
	connStr, _ := providerConfig.(map[string]any)["connection_string"].(string)
	if connStr == "" {
		return nil, vectorstores.NewVectorStoreError(vectorstores.ErrCodeInvalidConfig,
			"connection_string is required in pgvector provider config")
	}

	tableName, _ := providerConfig.(map[string]any)["table_name"].(string)
	if tableName == "" {
		tableName = "beluga_documents"
	}

	embeddingDim, _ := providerConfig.(map[string]any)["embedding_dimension"].(int)
	if embeddingDim == 0 {
		embeddingDim = 768 // default
	}

	// Initialize database connection
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, vectorstores.WrapError(err, vectorstores.ErrCodeConnectionFailed,
			"failed to connect to PostgreSQL database")
	}

	// Test the connection
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, vectorstores.WrapError(err, vectorstores.ErrCodeConnectionFailed,
			"failed to ping PostgreSQL database")
	}

	// Ensure table exists
	store := &PgVectorStore{
		db:             db,
		tableName:      tableName,
		embeddingDim:   embeddingDim,
		collectionName: "default",
		name:           "pgvector",
	}

	if err := store.ensureTableExists(ctx); err != nil {
		_ = db.Close()
		return nil, vectorstores.WrapError(err, vectorstores.ErrCodeStorageFailed,
			"failed to ensure table exists")
	}

	// Log configuration
	vectorstores.LogInfo(ctx, "PgVector store configured",
		"pgvector",
		slog.String("table_name", tableName),
		slog.Int("embedding_dimension", embeddingDim))

	return store, nil
}

// Note: Provider registration is handled externally to avoid import cycles.
// Applications should import this package for side effects to register the provider.
// import _ "github.com/lookatitude/beluga-ai/pkg/vectorstores/providers/pgvector"

// NewPgVectorStore creates a new PgVectorStore.
// It requires a database connection string, table name, and embedding dimension.
// Further configuration can be passed via PgVectorStoreConfig within vectorstoresiface.Config.ProviderArgs.
func NewPgVectorStore(ctx context.Context, config vectorstoresiface.Config) (*PgVectorStore, error) {
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
	if err != nil {
		return fmt.Errorf("failed to create table and index: %w", err)
	}
	return nil
}

// AddDocuments adds documents to the PgVectorStore.
func (s *PgVectorStore) AddDocuments(ctx context.Context, documents []schema.Document, opts ...vectorstores.Option) ([]string, error) {
	// Apply options
	config := vectorstores.NewDefaultConfig()
	vectorstores.ApplyOptions(config, opts...)

	// Use embedder from options
	embedder := config.Embedder
	if embedder == nil {
		return nil, vectorstores.NewVectorStoreError(vectorstores.ErrCodeEmbeddingFailed,
			"embedder is required to add documents")
	}

	// Generate embeddings if needed
	ids := make([]string, len(documents))
	docsToEmbed := make([]string, 0, len(documents))
	docIndices := make([]int, 0, len(documents))

	for i, doc := range documents {
		if len(doc.Embedding) == 0 {
			docsToEmbed = append(docsToEmbed, doc.GetContent())
			docIndices = append(docIndices, i)
		}
	}

	var embeddings [][]float32
	if len(docsToEmbed) > 0 {
		embeds, err := embedder.EmbedDocuments(ctx, docsToEmbed)
		if err != nil {
			return nil, vectorstores.WrapError(err, vectorstores.ErrCodeEmbeddingFailed,
				"failed to embed documents")
		}
		embeddings = embeds
	}

	// Begin transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, vectorstores.WrapError(err, vectorstores.ErrCodeStorageFailed,
			"failed to begin transaction")
	}
	defer func() { _ = tx.Rollback() }()

	// Prepare statement for inserting documents
	stmt, err := tx.PrepareContext(ctx, fmt.Sprintf(
		"INSERT INTO %s (content, metadata, embedding, collection_name) VALUES ($1, $2, $3, $4) RETURNING id",
		s.tableName))
	if err != nil {
		return nil, vectorstores.WrapError(err, vectorstores.ErrCodeStorageFailed,
			"failed to prepare insert statement")
	}
	defer func() { _ = stmt.Close() }()

	embedIndex := 0
	for i, doc := range documents {
		var docEmbedding []float32
		if len(doc.Embedding) > 0 {
			docEmbedding = doc.Embedding
		} else {
			docEmbedding = embeddings[embedIndex]
			embedIndex++
		}

		if len(docEmbedding) != s.embeddingDim {
			return nil, vectorstores.NewVectorStoreError(vectorstores.ErrCodeInvalidParameters,
				"document embedding dimension %d does not match store dimension %d", len(docEmbedding), s.embeddingDim)
		}

		// Convert embedding to pgvector format
		embeddingStr := fmt.Sprintf("[%s]", float32SliceToString(docEmbedding, ","))

		// Convert metadata to JSON
		var metadataBytes []byte
		if doc.Metadata != nil {
			metadataBytes, err = json.Marshal(doc.Metadata)
			if err != nil {
				return nil, vectorstores.WrapError(err, vectorstores.ErrCodeStorageFailed,
					"failed to marshal document metadata")
			}
		} else {
			metadataBytes = []byte("{}")
		}

		// Insert document and get the generated ID
		var id string
		err = stmt.QueryRowContext(ctx, doc.GetContent(), string(metadataBytes), embeddingStr, s.collectionName).Scan(&id)
		if err != nil {
			return nil, vectorstores.WrapError(err, vectorstores.ErrCodeStorageFailed,
				"failed to insert document")
		}

		ids[i] = id
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return nil, vectorstores.WrapError(err, vectorstores.ErrCodeStorageFailed,
			"failed to commit transaction")
	}

	return ids, nil
}

// SimilaritySearch performs a similarity search using a pre-computed query vector.
func (s *PgVectorStore) SimilaritySearch(ctx context.Context, queryVector []float32, k int, opts ...vectorstores.Option) ([]schema.Document, []float32, error) {
	if len(queryVector) != s.embeddingDim {
		return nil, nil, vectorstores.NewVectorStoreError(vectorstores.ErrCodeInvalidParameters,
			"query vector dimension %d does not match store dimension %d", len(queryVector), s.embeddingDim)
	}

	if k <= 0 {
		return nil, nil, vectorstores.NewVectorStoreError(vectorstores.ErrCodeInvalidParameters,
			"k must be greater than 0")
	}

	// Apply options
	config := vectorstores.NewDefaultConfig()
	vectorstores.ApplyOptions(config, opts...)
	if k == 0 {
		k = config.SearchK
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
		return nil, nil, vectorstores.WrapError(err, vectorstores.ErrCodeRetrievalFailed,
			"failed to execute similarity search query")
	}
	defer func() { _ = rows.Close() }()

	var resultDocs []schema.Document
	var resultScores []float32

	for rows.Next() {
		var doc schema.Document
		var distance float32
		var metadataStr sql.NullString

		if err := rows.Scan(&doc.ID, &doc.PageContent, &metadataStr, &distance); err != nil {
			return nil, nil, vectorstores.WrapError(err, vectorstores.ErrCodeRetrievalFailed,
				"failed to scan row")
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

	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("error iterating query results: %w", err)
	}
	return resultDocs, resultScores, nil
}

// SimilaritySearchByQuery generates an embedding for the query and then performs a similarity search.
func (s *PgVectorStore) SimilaritySearchByQuery(ctx context.Context, query string, k int, embedder vectorstores.Embedder, opts ...vectorstores.Option) ([]schema.Document, []float32, error) {
	if embedder == nil {
		return nil, nil, vectorstores.NewVectorStoreError(vectorstores.ErrCodeEmbeddingFailed,
			"embedder is required for SimilaritySearchByQuery")
	}

	queryEmbedding, err := embedder.EmbedQuery(ctx, query)
	if err != nil {
		return nil, nil, vectorstores.WrapError(err, vectorstores.ErrCodeEmbeddingFailed,
			"failed to embed query")
	}

	return s.SimilaritySearch(ctx, queryEmbedding, k, opts...)
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
	args := make([]any, len(ids))
	for i, id := range ids {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}

	query := fmt.Sprintf("DELETE FROM %s WHERE id IN (%s)", s.tableName, strings.Join(placeholders, ","))

	_, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return vectorstores.WrapError(err, vectorstores.ErrCodeStorageFailed,
			"failed to delete documents")
	}

	return nil
}

// AsRetriever returns a Retriever instance based on this VectorStore.
func (s *PgVectorStore) AsRetriever(opts ...vectorstores.Option) vectorstores.Retriever {
	return &PgVectorRetriever{
		store: s,
		opts:  opts,
	}
}

// PgVectorRetriever implements the Retriever interface for PgVectorStore.
type PgVectorRetriever struct {
	store *PgVectorStore
	opts  []vectorstores.Option
}

// GetRelevantDocuments retrieves relevant documents for a query.
func (r *PgVectorRetriever) GetRelevantDocuments(ctx context.Context, query string) ([]schema.Document, error) {
	// Apply default search options
	opts := append(r.opts, vectorstores.WithSearchK(5))

	docs, _, err := r.store.SimilaritySearchByQuery(ctx, query, 5, nil, opts...)
	return docs, err
}

// Ensure PgVectorStore implements the VectorStore interface.
var _ vectorstores.VectorStore = (*PgVectorStore)(nil)
