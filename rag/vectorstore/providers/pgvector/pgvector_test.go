package pgvector

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/rag/vectorstore"
	"github.com/lookatitude/beluga-ai/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// poolAdapter wraps pgxmock to implement Pool.
type poolAdapter struct {
	mock pgxmock.PgxPoolIface
}

func (a *poolAdapter) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	return a.mock.Exec(ctx, sql, args...)
}

func (a *poolAdapter) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return a.mock.Query(ctx, sql, args...)
}

func newTestStore(t *testing.T) (*Store, pgxmock.PgxPoolIface) {
	t.Helper()
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	store := New(&poolAdapter{mock: mock}, WithTable("test_docs"), WithDimension(3))
	return store, mock
}

func TestNew(t *testing.T) {
	store, _ := newTestStore(t)
	require.NotNil(t, store)
	assert.Equal(t, "test_docs", store.table)
	assert.Equal(t, 3, store.dimension)
}

func TestNew_Defaults(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	store := New(&poolAdapter{mock: mock})
	assert.Equal(t, "documents", store.table)
	assert.Equal(t, 1536, store.dimension)
}

func TestStore_InterfaceCompliance(t *testing.T) {
	var _ vectorstore.VectorStore = (*Store)(nil)
}

func TestStore_Add(t *testing.T) {
	store, mock := newTestStore(t)
	defer mock.Close()

	docs := []schema.Document{
		{ID: "doc1", Content: "hello world", Metadata: map[string]any{"category": "test"}},
		{ID: "doc2", Content: "goodbye world"},
	}
	embeddings := [][]float32{
		{0.1, 0.2, 0.3},
		{0.4, 0.5, 0.6},
	}

	for _, doc := range docs {
		meta, _ := json.Marshal(doc.Metadata)
		mock.ExpectExec("INSERT INTO test_docs").
			WithArgs(doc.ID, pgxmock.AnyArg(), doc.Content, meta).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))
	}

	err := store.Add(context.Background(), docs, embeddings)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestStore_Add_MismatchedLength(t *testing.T) {
	store, mock := newTestStore(t)
	defer mock.Close()

	docs := []schema.Document{{ID: "doc1"}}
	embeddings := [][]float32{{0.1}, {0.2}}

	err := store.Add(context.Background(), docs, embeddings)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "docs length")
}

func TestStore_Add_ExecError(t *testing.T) {
	store, mock := newTestStore(t)
	defer mock.Close()

	docs := []schema.Document{{ID: "doc1", Content: "test"}}
	embeddings := [][]float32{{0.1, 0.2, 0.3}}

	meta, _ := json.Marshal(docs[0].Metadata)
	mock.ExpectExec("INSERT INTO test_docs").
		WithArgs("doc1", pgxmock.AnyArg(), "test", meta).
		WillReturnError(fmt.Errorf("connection refused"))

	err := store.Add(context.Background(), docs, embeddings)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "connection refused")
}

func TestStore_Search(t *testing.T) {
	store, mock := newTestStore(t)
	defer mock.Close()

	meta := map[string]any{"category": "A"}
	metaJSON, _ := json.Marshal(meta)

	rows := pgxmock.NewRows([]string{"id", "content", "metadata", "score"}).
		AddRow("doc1", "hello", metaJSON, 0.95).
		AddRow("doc2", "world", []byte("{}"), 0.80)

	mock.ExpectQuery("SELECT id, content, metadata").
		WithArgs(pgxmock.AnyArg(), 5).
		WillReturnRows(rows)

	results, err := store.Search(context.Background(), []float32{0.1, 0.2, 0.3}, 5)
	require.NoError(t, err)
	require.Len(t, results, 2)

	assert.Equal(t, "doc1", results[0].ID)
	assert.Equal(t, "hello", results[0].Content)
	assert.Equal(t, 0.95, results[0].Score)
	assert.Equal(t, "A", results[0].Metadata["category"])

	assert.Equal(t, "doc2", results[1].ID)
	assert.Equal(t, 0.80, results[1].Score)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestStore_Search_WithFilter(t *testing.T) {
	store, mock := newTestStore(t)
	defer mock.Close()

	metaJSON, _ := json.Marshal(map[string]any{"category": "A"})
	rows := pgxmock.NewRows([]string{"id", "content", "metadata", "score"}).
		AddRow("doc1", "hello", metaJSON, 0.95)

	mock.ExpectQuery("SELECT id, content, metadata").
		WithArgs(pgxmock.AnyArg(), 5, pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnRows(rows)

	filter := map[string]any{"category": "A"}
	results, err := store.Search(context.Background(), []float32{0.1, 0.2, 0.3}, 5,
		vectorstore.WithFilter(filter))
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "doc1", results[0].ID)
}

func TestStore_Search_WithThreshold(t *testing.T) {
	store, mock := newTestStore(t)
	defer mock.Close()

	rows := pgxmock.NewRows([]string{"id", "content", "metadata", "score"}).
		AddRow("doc1", "hello", []byte("{}"), 0.95).
		AddRow("doc2", "world", []byte("{}"), 0.30)

	mock.ExpectQuery("SELECT id, content, metadata").
		WithArgs(pgxmock.AnyArg(), 5).
		WillReturnRows(rows)

	results, err := store.Search(context.Background(), []float32{0.1, 0.2, 0.3}, 5,
		vectorstore.WithThreshold(0.5))
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "doc1", results[0].ID)
	assert.GreaterOrEqual(t, results[0].Score, 0.5)
}

func TestStore_Search_DotProduct(t *testing.T) {
	store, mock := newTestStore(t)
	defer mock.Close()

	rows := pgxmock.NewRows([]string{"id", "content", "metadata", "score"}).
		AddRow("doc1", "hello", []byte("{}"), 14.0)

	mock.ExpectQuery("SELECT id, content, metadata").
		WithArgs(pgxmock.AnyArg(), 2).
		WillReturnRows(rows)

	results, err := store.Search(context.Background(), []float32{1.0, 2.0, 3.0}, 2,
		vectorstore.WithStrategy(vectorstore.DotProduct))
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, 14.0, results[0].Score)
}

func TestStore_Search_Euclidean(t *testing.T) {
	store, mock := newTestStore(t)
	defer mock.Close()

	rows := pgxmock.NewRows([]string{"id", "content", "metadata", "score"}).
		AddRow("doc1", "hello", []byte("{}"), -1.0)

	mock.ExpectQuery("SELECT id, content, metadata").
		WithArgs(pgxmock.AnyArg(), 2).
		WillReturnRows(rows)

	results, err := store.Search(context.Background(), []float32{1.0, 0.0, 0.0}, 2,
		vectorstore.WithStrategy(vectorstore.Euclidean))
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, -1.0, results[0].Score)
}

func TestStore_Search_Empty(t *testing.T) {
	store, mock := newTestStore(t)
	defer mock.Close()

	rows := pgxmock.NewRows([]string{"id", "content", "metadata", "score"})
	mock.ExpectQuery("SELECT id, content, metadata").
		WithArgs(pgxmock.AnyArg(), 5).
		WillReturnRows(rows)

	results, err := store.Search(context.Background(), []float32{0.1, 0.2, 0.3}, 5)
	require.NoError(t, err)
	assert.Len(t, results, 0)
}

func TestStore_Search_QueryError(t *testing.T) {
	store, mock := newTestStore(t)
	defer mock.Close()

	mock.ExpectQuery("SELECT id, content, metadata").
		WithArgs(pgxmock.AnyArg(), 5).
		WillReturnError(fmt.Errorf("connection lost"))

	_, err := store.Search(context.Background(), []float32{0.1, 0.2, 0.3}, 5)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "connection lost")
}

func TestStore_Delete(t *testing.T) {
	store, mock := newTestStore(t)
	defer mock.Close()

	mock.ExpectExec("DELETE FROM test_docs WHERE id IN").
		WithArgs("doc1", "doc2").
		WillReturnResult(pgxmock.NewResult("DELETE", 2))

	err := store.Delete(context.Background(), []string{"doc1", "doc2"})
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestStore_Delete_Empty(t *testing.T) {
	store, mock := newTestStore(t)
	defer mock.Close()

	err := store.Delete(context.Background(), []string{})
	require.NoError(t, err)
}

func TestStore_Delete_Error(t *testing.T) {
	store, mock := newTestStore(t)
	defer mock.Close()

	mock.ExpectExec("DELETE FROM test_docs WHERE id IN").
		WithArgs("doc1").
		WillReturnError(fmt.Errorf("db error"))

	err := store.Delete(context.Background(), []string{"doc1"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "db error")
}

func TestStore_EnsureTable(t *testing.T) {
	store, mock := newTestStore(t)
	defer mock.Close()

	mock.ExpectExec("CREATE EXTENSION IF NOT EXISTS vector").
		WillReturnResult(pgxmock.NewResult("CREATE EXTENSION", 0))
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS test_docs").
		WillReturnResult(pgxmock.NewResult("CREATE TABLE", 0))

	err := store.EnsureTable(context.Background())
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestStore_EnsureTable_ExtensionError(t *testing.T) {
	store, mock := newTestStore(t)
	defer mock.Close()

	mock.ExpectExec("CREATE EXTENSION IF NOT EXISTS vector").
		WillReturnError(fmt.Errorf("permission denied"))

	err := store.EnsureTable(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "permission denied")
}

func TestDistanceOperator(t *testing.T) {
	assert.Equal(t, "<=>", distanceOperator(vectorstore.Cosine))
	assert.Equal(t, "<#>", distanceOperator(vectorstore.DotProduct))
	assert.Equal(t, "<->", distanceOperator(vectorstore.Euclidean))
}

func TestRegistry_Integration(t *testing.T) {
	names := vectorstore.List()
	assert.Contains(t, names, "pgvector")
}

func TestNewFromConfig_MissingBaseURL(t *testing.T) {
	_, err := NewFromConfig(config.ProviderConfig{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "base_url")
}

func TestNewFromConfig_InvalidConnectionString(t *testing.T) {
	cfg := config.ProviderConfig{
		BaseURL: "invalid://connection/string",
	}
	_, err := NewFromConfig(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "connect")
}

func TestStore_EnsureTable_TableCreationError(t *testing.T) {
	store, mock := newTestStore(t)
	defer mock.Close()

	mock.ExpectExec("CREATE EXTENSION IF NOT EXISTS vector").
		WillReturnResult(pgxmock.NewResult("CREATE EXTENSION", 0))
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS test_docs").
		WillReturnError(fmt.Errorf("disk full"))

	err := store.EnsureTable(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "disk full")
}

func TestStore_Search_ScanError(t *testing.T) {
	store, mock := newTestStore(t)
	defer mock.Close()

	rows := pgxmock.NewRows([]string{"id", "content", "metadata", "score"}).
		AddRow("doc1", "hello", []byte("{}"), 0.95).
		RowError(0, fmt.Errorf("scan failed"))

	mock.ExpectQuery("SELECT id, content, metadata").
		WithArgs(pgxmock.AnyArg(), 5).
		WillReturnRows(rows)

	_, err := store.Search(context.Background(), []float32{0.1, 0.2, 0.3}, 5)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "scan")
}

func TestStore_Search_UnmarshalError(t *testing.T) {
	store, mock := newTestStore(t)
	defer mock.Close()

	invalidJSON := []byte("{invalid json")
	rows := pgxmock.NewRows([]string{"id", "content", "metadata", "score"}).
		AddRow("doc1", "hello", invalidJSON, 0.95)

	mock.ExpectQuery("SELECT id, content, metadata").
		WithArgs(pgxmock.AnyArg(), 5).
		WillReturnRows(rows)

	_, err := store.Search(context.Background(), []float32{0.1, 0.2, 0.3}, 5)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshal metadata")
}

func TestStore_Search_RowsError(t *testing.T) {
	store, mock := newTestStore(t)
	defer mock.Close()

	rows := pgxmock.NewRows([]string{"id", "content", "metadata", "score"}).
		AddRow("doc1", "hello", []byte("{}"), 0.95).
		AddRow("doc2", "world", []byte("{}"), 0.80)

	mock.ExpectQuery("SELECT id, content, metadata").
		WithArgs(pgxmock.AnyArg(), 5).
		WillReturnRows(rows.CloseError(fmt.Errorf("rows error")))

	_, err := store.Search(context.Background(), []float32{0.1, 0.2, 0.3}, 5)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "rows error")
}

func TestStore_Search_MultipleFilters(t *testing.T) {
	store, mock := newTestStore(t)
	defer mock.Close()

	metaJSON, _ := json.Marshal(map[string]any{"category": "A", "lang": "en"})
	rows := pgxmock.NewRows([]string{"id", "content", "metadata", "score"}).
		AddRow("doc1", "hello", metaJSON, 0.95)

	mock.ExpectQuery("SELECT id, content, metadata").
		WithArgs(pgxmock.AnyArg(), 5, pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnRows(rows)

	filter := map[string]any{"category": "A", "lang": "en"}
	results, err := store.Search(context.Background(), []float32{0.1, 0.2, 0.3}, 5,
		vectorstore.WithFilter(filter))
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "doc1", results[0].ID)
}

func TestStore_Search_EmptyMetadata(t *testing.T) {
	store, mock := newTestStore(t)
	defer mock.Close()

	rows := pgxmock.NewRows([]string{"id", "content", "metadata", "score"}).
		AddRow("doc1", "hello", []byte(nil), 0.95)

	mock.ExpectQuery("SELECT id, content, metadata").
		WithArgs(pgxmock.AnyArg(), 5).
		WillReturnRows(rows)

	results, err := store.Search(context.Background(), []float32{0.1, 0.2, 0.3}, 5)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Nil(t, results[0].Metadata)
}

func TestRegistry_FactorySuccess(t *testing.T) {
	// Test that the registered factory can be looked up and called
	_, err := vectorstore.New("pgvector", config.ProviderConfig{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "base_url")
}

// unmarshalableType has a MarshalJSON method that always returns an error.
type unmarshalableType struct{}

func (u unmarshalableType) MarshalJSON() ([]byte, error) {
	return nil, fmt.Errorf("marshal error")
}

func TestStore_Add_MarshalError(t *testing.T) {
	store, mock := newTestStore(t)
	defer mock.Close()

	docs := []schema.Document{
		{ID: "doc1", Content: "test", Metadata: map[string]any{"bad": unmarshalableType{}}},
	}
	embeddings := [][]float32{{0.1, 0.2, 0.3}}

	err := store.Add(context.Background(), docs, embeddings)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "marshal metadata")
}
