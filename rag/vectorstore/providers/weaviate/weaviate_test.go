package weaviate

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
		WithClass("TestDoc"),
		WithHTTPClient(srv.Client()),
	)
	return srv, store
}

func TestNew(t *testing.T) {
	store := New("http://localhost:8080", WithClass("MyDoc"))
	require.NotNil(t, store)
	assert.Equal(t, "http://localhost:8080", store.baseURL)
	assert.Equal(t, "MyDoc", store.class)
}

func TestNew_Defaults(t *testing.T) {
	store := New("http://localhost:8080")
	assert.Equal(t, "Document", store.class)
}

func TestStore_InterfaceCompliance(t *testing.T) {
	var _ vectorstore.VectorStore = (*Store)(nil)
}

func TestStore_Add(t *testing.T) {
	var receivedBody map[string]any
	srv, store := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Contains(t, r.URL.Path, "/v1/batch/objects")

		json.NewDecoder(r.Body).Decode(&receivedBody)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[]`))
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

	objects := receivedBody["objects"].([]any)
	assert.Len(t, objects, 2)
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
		assert.Contains(t, r.URL.Path, "/v1/graphql")

		resp := map[string]any{
			"data": map[string]any{
				"Get": map[string]any{
					"TestDoc": []map[string]any{
						{
							"content":    "hello world",
							"_beluga_id": "doc1",
							"_additional": map[string]any{
								"id":       "some-uuid",
								"distance": 0.05,
							},
						},
						{
							"content":    "goodbye",
							"_beluga_id": "doc2",
							"_additional": map[string]any{
								"id":       "some-uuid-2",
								"distance": 0.2,
							},
						},
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
	assert.InDelta(t, 0.95, results[0].Score, 0.001)

	assert.Equal(t, "doc2", results[1].ID)
	assert.InDelta(t, 0.8, results[1].Score, 0.001)
}

func TestStore_Search_Empty(t *testing.T) {
	srv, store := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"data": map[string]any{
				"Get": map[string]any{
					"TestDoc": []any{},
				},
			},
		}
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

func TestStore_Search_NoResults(t *testing.T) {
	srv, store := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"data": map[string]any{
				"Get": map[string]any{},
			},
		}
		json.NewEncoder(w).Encode(resp)
	})
	defer srv.Close()

	results, err := store.Search(context.Background(), []float32{0.1, 0.2, 0.3}, 5)
	require.NoError(t, err)
	assert.Nil(t, results)
}

func TestStore_Delete(t *testing.T) {
	deletedPaths := make([]string, 0)
	srv, store := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method)
		deletedPaths = append(deletedPaths, r.URL.Path)
		w.WriteHeader(http.StatusNoContent)
	})
	defer srv.Close()

	err := store.Delete(context.Background(), []string{"doc1", "doc2"})
	require.NoError(t, err)
	assert.Len(t, deletedPaths, 2)
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
		assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))
		resp := map[string]any{
			"data": map[string]any{
				"Get": map[string]any{
					"TestDoc": []any{},
				},
			},
		}
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
		w.Write([]byte(`{}`))
	})
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := store.Search(ctx, []float32{0.1, 0.2, 0.3}, 5)
	require.Error(t, err)
}

func TestRegistry_Integration(t *testing.T) {
	names := vectorstore.List()
	assert.Contains(t, names, "weaviate")
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
			"class": "CustomDoc",
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "http://localhost:8080", store.baseURL)
	assert.Equal(t, "my-key", store.apiKey)
	assert.Equal(t, "CustomDoc", store.class)
}

func TestUUIDFromID(t *testing.T) {
	uuid1 := uuidFromID("doc1")
	uuid2 := uuidFromID("doc2")
	assert.NotEqual(t, uuid1, uuid2, "different IDs should produce different UUIDs")

	// Same ID should produce same UUID.
	assert.Equal(t, uuid1, uuidFromID("doc1"))

	// Should be valid UUID format: 8-4-4-4-12 hex chars.
	assert.Regexp(t, `^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`, uuid1)
}
