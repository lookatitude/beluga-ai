package mongodb

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/rag/vectorstore"
	"github.com/lookatitude/beluga-ai/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestServer(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *Store) {
	t.Helper()
	srv := httptest.NewServer(handler)
	store := New(srv.URL,
		WithCollection("test_col"),
		WithDatabase("test_db"),
		WithIndex("test_index"),
		WithHTTPClient(srv.Client()),
	)
	return srv, store
}

func TestNew(t *testing.T) {
	store := New("http://localhost:8080",
		WithCollection("my_col"),
		WithDatabase("my_db"),
		WithIndex("my_index"),
	)
	require.NotNil(t, store)
	assert.Equal(t, "http://localhost:8080", store.baseURL)
	assert.Equal(t, "my_col", store.collection)
	assert.Equal(t, "my_db", store.database)
	assert.Equal(t, "my_index", store.index)
}

func TestNew_Defaults(t *testing.T) {
	store := New("http://localhost:8080")
	assert.Equal(t, "beluga", store.database)
	assert.Equal(t, "documents", store.collection)
	assert.Equal(t, "vector_index", store.index)
}

func TestStore_InterfaceCompliance(t *testing.T) {
	var _ vectorstore.VectorStore = (*Store)(nil)
}

func TestStore_Add(t *testing.T) {
	var receivedBody map[string]any
	srv, store := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Contains(t, r.URL.Path, "/action/insertMany")

		json.NewDecoder(r.Body).Decode(&receivedBody)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"insertedIds":["doc1","doc2"]}`))
	})
	defer srv.Close()

	docs := []schema.Document{
		{ID: "doc1", Content: "hello", Metadata: map[string]any{"category": "A"}},
		{ID: "doc2", Content: "world"},
	}
	embeddings := [][]float32{
		{0.1, 0.2, 0.3},
		{0.4, 0.5, 0.6},
	}

	err := store.Add(context.Background(), docs, embeddings)
	require.NoError(t, err)

	documents := receivedBody["documents"].([]any)
	assert.Len(t, documents, 2)
	assert.Equal(t, "test_db", receivedBody["database"])
	assert.Equal(t, "test_col", receivedBody["collection"])
}

func TestStore_Add_MismatchedLength(t *testing.T) {
	store := New("http://localhost:8080")
	err := store.Add(context.Background(),
		[]schema.Document{{ID: "doc1"}},
		[][]float32{{0.1}, {0.2}},
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "docs length")
}

func TestStore_Add_ServerError(t *testing.T) {
	srv, store := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"internal error"}`))
	})
	defer srv.Close()

	err := store.Add(context.Background(),
		[]schema.Document{{ID: "doc1", Content: "test"}},
		[][]float32{{0.1, 0.2, 0.3}},
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func TestStore_Search(t *testing.T) {
	srv, store := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Contains(t, r.URL.Path, "/action/aggregate")

		resp := map[string]any{
			"documents": []map[string]any{
				{
					"_id":     "doc1",
					"content": "hello world",
					"score":   0.95,
					"metadata": map[string]any{
						"category": "A",
					},
				},
				{
					"_id":     "doc2",
					"content": "goodbye",
					"score":   0.80,
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	})
	defer srv.Close()

	results, err := store.Search(context.Background(), []float32{0.1, 0.2, 0.3}, 5)
	require.NoError(t, err)
	require.Len(t, results, 2)

	assert.Equal(t, "doc1", results[0].ID)
	assert.Equal(t, "hello world", results[0].Content)
	assert.Equal(t, 0.95, results[0].Score)
	assert.Equal(t, "A", results[0].Metadata["category"])

	assert.Equal(t, "doc2", results[1].ID)
	assert.Equal(t, 0.80, results[1].Score)
}

func TestStore_Search_WithFilter(t *testing.T) {
	var receivedBody map[string]any
	srv, store := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedBody)
		resp := map[string]any{"documents": []any{}}
		json.NewEncoder(w).Encode(resp)
	})
	defer srv.Close()

	filter := map[string]any{"category": "A"}
	_, err := store.Search(context.Background(), []float32{0.1, 0.2, 0.3}, 5,
		vectorstore.WithFilter(filter))
	require.NoError(t, err)

	// Verify the pipeline contains filter.
	pipeline := receivedBody["pipeline"].([]any)
	require.NotEmpty(t, pipeline)
	vectorSearch := pipeline[0].(map[string]any)["$vectorSearch"].(map[string]any)
	f, ok := vectorSearch["filter"]
	require.True(t, ok, "filter should be in vectorSearch")
	filterMap := f.(map[string]any)
	assert.Equal(t, "A", filterMap["metadata.category"])
}

func TestStore_Search_WithThreshold(t *testing.T) {
	srv, store := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"documents": []map[string]any{
				{"_id": "doc1", "content": "hello", "score": 0.95},
				{"_id": "doc2", "content": "world", "score": 0.50},
			},
		}
		json.NewEncoder(w).Encode(resp)
	})
	defer srv.Close()

	results, err := store.Search(context.Background(), []float32{0.1, 0.2, 0.3}, 5,
		vectorstore.WithThreshold(0.7))
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "doc1", results[0].ID)
}

func TestStore_Search_Empty(t *testing.T) {
	srv, store := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{"documents": []any{}}
		json.NewEncoder(w).Encode(resp)
	})
	defer srv.Close()

	results, err := store.Search(context.Background(), []float32{0.1, 0.2, 0.3}, 5)
	require.NoError(t, err)
	assert.Len(t, results, 0)
}

func TestStore_Search_ServerError(t *testing.T) {
	srv, store := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"search failed"}`))
	})
	defer srv.Close()

	_, err := store.Search(context.Background(), []float32{0.1, 0.2, 0.3}, 5)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func TestStore_Delete(t *testing.T) {
	var receivedBody map[string]any
	srv, store := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Contains(t, r.URL.Path, "/action/deleteMany")

		json.NewDecoder(r.Body).Decode(&receivedBody)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"deletedCount":2}`))
	})
	defer srv.Close()

	err := store.Delete(context.Background(), []string{"doc1", "doc2"})
	require.NoError(t, err)

	filter := receivedBody["filter"].(map[string]any)
	in := filter["_id"].(map[string]any)["$in"].([]any)
	assert.Len(t, in, 2)
}

func TestStore_Delete_Empty(t *testing.T) {
	store := New("http://localhost:8080")
	err := store.Delete(context.Background(), []string{})
	require.NoError(t, err)
}

func TestStore_Delete_ServerError(t *testing.T) {
	srv, store := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"delete failed"}`))
	})
	defer srv.Close()

	err := store.Delete(context.Background(), []string{"doc1"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func TestStore_APIKey(t *testing.T) {
	srv, store := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "test-key", r.Header.Get("api-key"))
		resp := map[string]any{"documents": []any{}}
		json.NewEncoder(w).Encode(resp)
	})
	defer srv.Close()
	store.apiKey = "test-key"

	_, err := store.Search(context.Background(), []float32{0.1, 0.2, 0.3}, 5)
	require.NoError(t, err)
}

func TestStore_ContextCancelled(t *testing.T) {
	srv, store := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"documents":[]}`))
	})
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := store.Search(ctx, []float32{0.1, 0.2, 0.3}, 5)
	require.Error(t, err)
}

func TestRegistry_Integration(t *testing.T) {
	names := vectorstore.List()
	assert.Contains(t, names, "mongodb")
}

func TestNewFromConfig_MissingBaseURL(t *testing.T) {
	_, err := NewFromConfig(config.ProviderConfig{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "base_url")
}

func TestNewFromConfig(t *testing.T) {
	store, err := NewFromConfig(config.ProviderConfig{
		BaseURL: "http://localhost:8080",
		APIKey:  "my-key",
		Options: map[string]any{
			"database":   "my_db",
			"collection": "my_col",
			"index":      "my_index",
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "http://localhost:8080", store.baseURL)
	assert.Equal(t, "my-key", store.apiKey)
	assert.Equal(t, "my_db", store.database)
	assert.Equal(t, "my_col", store.collection)
	assert.Equal(t, "my_index", store.index)
}

func TestStore_Search_PipelineStructure(t *testing.T) {
	var receivedBody map[string]any
	srv, store := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedBody)
		resp := map[string]any{"documents": []any{}}
		json.NewEncoder(w).Encode(resp)
	})
	defer srv.Close()

	_, err := store.Search(context.Background(), []float32{0.1, 0.2, 0.3}, 10)
	require.NoError(t, err)

	pipeline := receivedBody["pipeline"].([]any)
	assert.Len(t, pipeline, 2) // $vectorSearch + $addFields

	vectorSearch := pipeline[0].(map[string]any)["$vectorSearch"].(map[string]any)
	assert.Equal(t, "test_index", vectorSearch["index"])
	assert.Equal(t, "embedding", vectorSearch["path"])
	assert.Equal(t, float64(10), vectorSearch["limit"])
	assert.Equal(t, float64(100), vectorSearch["numCandidates"])
}

func TestStore_Search_InvalidJSON(t *testing.T) {
	srv, store := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{invalid json`))
	})
	defer srv.Close()

	_, err := store.Search(context.Background(), []float32{0.1, 0.2, 0.3}, 5)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshal")
}

func TestRegistry_Factory(t *testing.T) {
	// Test that the init() registered factory works.
	store, err := vectorstore.New("mongodb", config.ProviderConfig{
		BaseURL: "http://localhost:8080",
		APIKey:  "test-key",
		Options: map[string]any{
			"database":   "test_db",
			"collection": "test_col",
			"index":      "test_idx",
		},
	})
	require.NoError(t, err)
	require.NotNil(t, store)

	// Verify it's actually a MongoDB store.
	mongoStore, ok := store.(*Store)
	require.True(t, ok)
	assert.Equal(t, "http://localhost:8080", mongoStore.baseURL)
	assert.Equal(t, "test-key", mongoStore.apiKey)
	assert.Equal(t, "test_db", mongoStore.database)
	assert.Equal(t, "test_col", mongoStore.collection)
	assert.Equal(t, "test_idx", mongoStore.index)
}

func TestStore_Add_WithNilMetadata(t *testing.T) {
	var receivedBody map[string]any
	srv, store := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedBody)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"insertedIds":["doc1"]}`))
	})
	defer srv.Close()

	docs := []schema.Document{
		{ID: "doc1", Content: "hello", Metadata: nil},
	}
	embeddings := [][]float32{{0.1, 0.2, 0.3}}

	err := store.Add(context.Background(), docs, embeddings)
	require.NoError(t, err)

	documents := receivedBody["documents"].([]any)
	doc := documents[0].(map[string]any)
	_, hasMetadata := doc["metadata"]
	assert.False(t, hasMetadata, "nil metadata should not be included")
}
