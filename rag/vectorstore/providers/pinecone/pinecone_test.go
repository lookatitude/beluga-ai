package pinecone

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
	store := New(srv.URL, "test-api-key",
		WithNamespace("test-ns"),
		WithHTTPClient(srv.Client()),
	)
	return srv, store
}

func TestNew(t *testing.T) {
	store := New("https://example.pinecone.io", "my-key", WithNamespace("ns1"))
	require.NotNil(t, store)
	assert.Equal(t, "https://example.pinecone.io", store.baseURL)
	assert.Equal(t, "my-key", store.apiKey)
	assert.Equal(t, "ns1", store.namespace)
}

func TestStore_InterfaceCompliance(t *testing.T) {
	var _ vectorstore.VectorStore = (*Store)(nil)
}

func TestStore_Add(t *testing.T) {
	var receivedBody map[string]any
	srv, store := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/vectors/upsert", r.URL.Path)
		assert.Equal(t, "test-api-key", r.Header.Get("Api-Key"))

		json.NewDecoder(r.Body).Decode(&receivedBody)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"upsertedCount":2}`))
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

	vectors := receivedBody["vectors"].([]any)
	assert.Len(t, vectors, 2)
	assert.Equal(t, "test-ns", receivedBody["namespace"])
}

func TestStore_Add_MismatchedLength(t *testing.T) {
	store := New("http://localhost", "key")
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
		assert.Equal(t, "/query", r.URL.Path)

		resp := map[string]any{
			"matches": []map[string]any{
				{
					"id":    "doc1",
					"score": 0.95,
					"metadata": map[string]any{
						"content":  "hello world",
						"category": "A",
					},
				},
				{
					"id":    "doc2",
					"score": 0.80,
					"metadata": map[string]any{
						"content": "goodbye",
					},
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
		resp := map[string]any{"matches": []any{}}
		json.NewEncoder(w).Encode(resp)
	})
	defer srv.Close()

	filter := map[string]any{"category": "A"}
	_, err := store.Search(context.Background(), []float32{0.1, 0.2, 0.3}, 5,
		vectorstore.WithFilter(filter))
	require.NoError(t, err)

	// Verify filter was sent in Pinecone format.
	f, ok := receivedBody["filter"]
	require.True(t, ok, "filter should be in request body")
	filterMap := f.(map[string]any)
	catFilter := filterMap["category"].(map[string]any)
	assert.Equal(t, "A", catFilter["$eq"])
}

func TestStore_Search_WithThreshold(t *testing.T) {
	srv, store := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"matches": []map[string]any{
				{"id": "doc1", "score": 0.95, "metadata": map[string]any{"content": "hello"}},
				{"id": "doc2", "score": 0.30, "metadata": map[string]any{"content": "world"}},
			},
		}
		json.NewEncoder(w).Encode(resp)
	})
	defer srv.Close()

	results, err := store.Search(context.Background(), []float32{0.1, 0.2, 0.3}, 5,
		vectorstore.WithThreshold(0.5))
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "doc1", results[0].ID)
}

func TestStore_Search_Empty(t *testing.T) {
	srv, store := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{"matches": []any{}}
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

func TestStore_Search_Namespace(t *testing.T) {
	var receivedBody map[string]any
	srv, store := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedBody)
		resp := map[string]any{"matches": []any{}}
		json.NewEncoder(w).Encode(resp)
	})
	defer srv.Close()

	_, err := store.Search(context.Background(), []float32{0.1, 0.2, 0.3}, 5)
	require.NoError(t, err)
	assert.Equal(t, "test-ns", receivedBody["namespace"])
}

func TestStore_Delete(t *testing.T) {
	var receivedBody map[string]any
	srv, store := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/vectors/delete", r.URL.Path)

		json.NewDecoder(r.Body).Decode(&receivedBody)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	})
	defer srv.Close()

	err := store.Delete(context.Background(), []string{"doc1", "doc2"})
	require.NoError(t, err)

	ids := receivedBody["ids"].([]any)
	assert.Len(t, ids, 2)
	assert.Equal(t, "test-ns", receivedBody["namespace"])
}

func TestStore_Delete_Empty(t *testing.T) {
	store := New("http://localhost", "key")
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

func TestStore_ContextCancelled(t *testing.T) {
	srv, store := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{"matches": []any{}}
		json.NewEncoder(w).Encode(resp)
	})
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := store.Search(ctx, []float32{0.1, 0.2, 0.3}, 5)
	require.Error(t, err)
}

func TestRegistry_Integration(t *testing.T) {
	names := vectorstore.List()
	assert.Contains(t, names, "pinecone")
}

func TestNewFromConfig_MissingBaseURL(t *testing.T) {
	_, err := NewFromConfig(config.ProviderConfig{APIKey: "key"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "base_url")
}

func TestNewFromConfig_MissingAPIKey(t *testing.T) {
	_, err := NewFromConfig(config.ProviderConfig{BaseURL: "http://example.com"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "api_key")
}

func TestNewFromConfig(t *testing.T) {
	store, err := NewFromConfig(config.ProviderConfig{
		BaseURL: "https://example.pinecone.io",
		APIKey:  "my-key",
		Options: map[string]any{
			"namespace": "my_ns",
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "https://example.pinecone.io", store.baseURL)
	assert.Equal(t, "my-key", store.apiKey)
	assert.Equal(t, "my_ns", store.namespace)
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
	store, err := vectorstore.New("pinecone", config.ProviderConfig{
		BaseURL: "https://example.pinecone.io",
		APIKey:  "test-key",
		Options: map[string]any{
			"namespace": "test_ns",
		},
	})
	require.NoError(t, err)
	require.NotNil(t, store)

	// Verify it's actually a Pinecone store.
	pineconeStore, ok := store.(*Store)
	require.True(t, ok)
	assert.Equal(t, "https://example.pinecone.io", pineconeStore.baseURL)
	assert.Equal(t, "test-key", pineconeStore.apiKey)
	assert.Equal(t, "test_ns", pineconeStore.namespace)
}

func TestStore_Add_WithoutNamespace(t *testing.T) {
	var receivedBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedBody)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"upsertedCount":1}`))
	}))
	defer srv.Close()

	store := New(srv.URL, "key", WithHTTPClient(srv.Client()))
	// No namespace set.

	docs := []schema.Document{{ID: "doc1", Content: "test"}}
	embeddings := [][]float32{{0.1, 0.2}}

	err := store.Add(context.Background(), docs, embeddings)
	require.NoError(t, err)

	_, hasNamespace := receivedBody["namespace"]
	assert.False(t, hasNamespace, "empty namespace should not be included")
}

func TestStore_Search_NoContentInMetadata(t *testing.T) {
	srv, store := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"matches": []map[string]any{
				{
					"id":       "doc1",
					"score":    0.95,
					"metadata": map[string]any{"category": "A"},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	})
	defer srv.Close()

	results, err := store.Search(context.Background(), []float32{0.1, 0.2, 0.3}, 5)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "", results[0].Content)
	assert.Equal(t, "A", results[0].Metadata["category"])
}
