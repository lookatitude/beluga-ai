package chroma

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
		WithCollectionID("col-123"),
		WithHTTPClient(srv.Client()),
	)
	return srv, store
}

func TestNew(t *testing.T) {
	store := New("http://localhost:8000", WithCollection("my_col"))
	require.NotNil(t, store)
	assert.Equal(t, "http://localhost:8000", store.baseURL)
	assert.Equal(t, "my_col", store.collection)
	assert.Equal(t, "default_tenant", store.tenant)
	assert.Equal(t, "default_database", store.database)
}

func TestNew_Defaults(t *testing.T) {
	store := New("http://localhost:8000")
	assert.Equal(t, "default_tenant", store.tenant)
	assert.Equal(t, "default_database", store.database)
}

func TestStore_InterfaceCompliance(t *testing.T) {
	var _ vectorstore.VectorStore = (*Store)(nil)
}

func TestStore_Add(t *testing.T) {
	var receivedBody map[string]any
	srv, store := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Contains(t, r.URL.Path, "/collections/col-123/upsert")

		json.NewDecoder(r.Body).Decode(&receivedBody)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`true`))
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

	ids := receivedBody["ids"].([]any)
	assert.Len(t, ids, 2)
	assert.Equal(t, "doc1", ids[0])
	assert.Equal(t, "doc2", ids[1])
}

func TestStore_Add_MismatchedLength(t *testing.T) {
	store := New("http://localhost:8000")
	store.collectionID = "col-123"
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
		assert.Contains(t, r.URL.Path, "/collections/col-123/query")

		resp := map[string]any{
			"ids":       [][]string{{"doc1", "doc2"}},
			"documents": [][]string{{"hello world", "goodbye"}},
			"metadatas": [][]map[string]any{
				{{"category": "A"}, {}},
			},
			"distances": [][]float64{{0.05, 0.25}},
		}
		json.NewEncoder(w).Encode(resp)
	})
	defer srv.Close()

	results, err := store.Search(context.Background(), []float32{0.1, 0.2, 0.3}, 5)
	require.NoError(t, err)
	require.Len(t, results, 2)

	// Lower distance = higher score. 1/(1+0.05) ≈ 0.952, 1/(1+0.25) = 0.8
	assert.Equal(t, "doc1", results[0].ID)
	assert.Equal(t, "hello world", results[0].Content)
	assert.Greater(t, results[0].Score, results[1].Score)
	assert.Equal(t, "A", results[0].Metadata["category"])

	assert.Equal(t, "doc2", results[1].ID)
}

func TestStore_Search_WithFilter(t *testing.T) {
	var receivedBody map[string]any
	srv, store := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedBody)
		resp := map[string]any{
			"ids":       [][]string{},
			"documents": [][]string{},
			"metadatas": [][]map[string]any{},
			"distances": [][]float64{},
		}
		json.NewEncoder(w).Encode(resp)
	})
	defer srv.Close()

	filter := map[string]any{"category": "A"}
	_, err := store.Search(context.Background(), []float32{0.1, 0.2, 0.3}, 5,
		vectorstore.WithFilter(filter))
	require.NoError(t, err)

	// Verify where clause was sent.
	where, ok := receivedBody["where"]
	require.True(t, ok, "where should be in request body")
	whereMap := where.(map[string]any)
	catFilter := whereMap["category"].(map[string]any)
	assert.Equal(t, "A", catFilter["$eq"])
}

func TestStore_Search_WithThreshold(t *testing.T) {
	srv, store := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"ids":       [][]string{{"doc1", "doc2"}},
			"documents": [][]string{{"hello", "world"}},
			"metadatas": [][]map[string]any{{{"k": "v"}, {"k": "v2"}}},
			"distances": [][]float64{{0.05, 5.0}}, // scores: ~0.95 and ~0.167
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
		resp := map[string]any{
			"ids":       [][]string{},
			"documents": [][]string{},
			"metadatas": [][]map[string]any{},
			"distances": [][]float64{},
		}
		json.NewEncoder(w).Encode(resp)
	})
	defer srv.Close()

	results, err := store.Search(context.Background(), []float32{0.1, 0.2, 0.3}, 5)
	require.NoError(t, err)
	assert.Nil(t, results)
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
		assert.Contains(t, r.URL.Path, "/collections/col-123/delete")

		json.NewDecoder(r.Body).Decode(&receivedBody)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[]`))
	})
	defer srv.Close()

	err := store.Delete(context.Background(), []string{"doc1", "doc2"})
	require.NoError(t, err)

	ids := receivedBody["ids"].([]any)
	assert.Len(t, ids, 2)
}

func TestStore_Delete_Empty(t *testing.T) {
	store := New("http://localhost:8000")
	store.collectionID = "col-123"
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

func TestStore_EnsureCollection(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/tenants/default_tenant/databases/default_database/collections")
		resp := map[string]any{"id": "resolved-col-id", "name": "test_col"}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	store := New(srv.URL,
		WithCollection("test_col"),
		WithHTTPClient(srv.Client()),
	)

	err := store.EnsureCollection(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "resolved-col-id", store.collectionID)
}

func TestStore_ContextCancelled(t *testing.T) {
	srv, store := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := store.Search(ctx, []float32{0.1, 0.2, 0.3}, 5)
	require.Error(t, err)
}

func TestRegistry_Integration(t *testing.T) {
	names := vectorstore.List()
	assert.Contains(t, names, "chroma")
}

func TestNewFromConfig_MissingBaseURL(t *testing.T) {
	_, err := NewFromConfig(config.ProviderConfig{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "base_url")
}

func TestNewFromConfig(t *testing.T) {
	store, err := NewFromConfig(config.ProviderConfig{
		BaseURL: "http://localhost:8000",
		Options: map[string]any{
			"collection": "my_col",
			"tenant":     "my_tenant",
			"database":   "my_db",
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "http://localhost:8000", store.baseURL)
	assert.Equal(t, "my_col", store.collection)
	assert.Equal(t, "my_tenant", store.tenant)
	assert.Equal(t, "my_db", store.database)
}

func TestStore_CustomTenantDatabase(t *testing.T) {
	store := New("http://localhost:8000",
		WithTenant("custom_tenant"),
		WithDatabase("custom_db"),
	)
	assert.Equal(t, "custom_tenant", store.tenant)
	assert.Equal(t, "custom_db", store.database)
}

func TestStore_ResolveCollectionID_NoCollectionNameOrID(t *testing.T) {
	store := New("http://localhost:8000")
	// No collection name or ID set, should error
	_, err := store.resolveCollectionID(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "collection name or ID is required")
}

func TestStore_ResolveCollectionID_AlreadyResolved(t *testing.T) {
	store := New("http://localhost:8000", WithCollectionID("existing-id"))
	colID, err := store.resolveCollectionID(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "existing-id", colID)
}

func TestStore_ResolveCollectionID_CallsEnsureCollection(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{"id": "new-col-id", "name": "test_col"}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	store := New(srv.URL,
		WithCollection("test_col"),
		WithHTTPClient(srv.Client()),
	)

	colID, err := store.resolveCollectionID(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "new-col-id", colID)
	assert.Equal(t, "new-col-id", store.collectionID)
}

func TestStore_EnsureCollection_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"failed to create collection"}`))
	}))
	defer srv.Close()

	store := New(srv.URL,
		WithCollection("test_col"),
		WithHTTPClient(srv.Client()),
	)

	err := store.EnsureCollection(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func TestStore_EnsureCollection_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"invalid json`))
	}))
	defer srv.Close()

	store := New(srv.URL,
		WithCollection("test_col"),
		WithHTTPClient(srv.Client()),
	)

	err := store.EnsureCollection(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshal")
}

func TestStore_Add_ResolveCollectionError(t *testing.T) {
	store := New("http://localhost:8000")
	// No collection name or ID, should fail during resolve
	err := store.Add(context.Background(),
		[]schema.Document{{ID: "doc1", Content: "test"}},
		[][]float32{{0.1, 0.2, 0.3}},
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "collection name or ID is required")
}

func TestStore_Search_ResolveCollectionError(t *testing.T) {
	store := New("http://localhost:8000")
	// No collection name or ID, should fail during resolve
	_, err := store.Search(context.Background(), []float32{0.1, 0.2, 0.3}, 5)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "collection name or ID is required")
}

func TestStore_Search_InvalidJSON(t *testing.T) {
	srv, store := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"invalid json`))
	})
	defer srv.Close()

	_, err := store.Search(context.Background(), []float32{0.1, 0.2, 0.3}, 5)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshal")
}

func TestStore_Delete_ResolveCollectionError(t *testing.T) {
	store := New("http://localhost:8000")
	// No collection name or ID, should fail during resolve
	err := store.Delete(context.Background(), []string{"doc1"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "collection name or ID is required")
}

func TestStore_DoJSON_NilBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"result":"ok"}`))
	}))
	defer srv.Close()

	store := New(srv.URL, WithHTTPClient(srv.Client()))
	resp, err := store.doJSON(context.Background(), http.MethodGet, "/test", nil)
	require.NoError(t, err)
	assert.Contains(t, string(resp), "ok")
}

func TestRegistry_Factory(t *testing.T) {
	// Test the factory function from registry
	store, err := vectorstore.New("chroma", config.ProviderConfig{
		BaseURL: "http://localhost:8000",
		Options: map[string]any{
			"collection": "test",
		},
	})
	require.NoError(t, err)
	require.NotNil(t, store)
	_, ok := store.(*Store)
	assert.True(t, ok, "should return *chroma.Store")
}

func TestRegistry_FactoryError(t *testing.T) {
	// Test factory error path (missing base_url)
	_, err := vectorstore.New("chroma", config.ProviderConfig{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "base_url")
}

func TestStore_ResolveCollectionID_EnsureCollectionError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"collection creation failed"}`))
	}))
	defer srv.Close()

	store := New(srv.URL,
		WithCollection("test_col"),
		WithHTTPClient(srv.Client()),
	)

	_, err := store.resolveCollectionID(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func TestStore_DoJSON_MarshalError(t *testing.T) {
	store := New("http://localhost:8000")
	// Create an unmarshalable body (channels can't be marshaled)
	badBody := map[string]any{
		"channel": make(chan int),
	}
	_, err := store.doJSON(context.Background(), http.MethodPost, "/test", badBody)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "marshal")
}

func TestStore_DoJSON_InvalidURL(t *testing.T) {
	store := New("http://[::1]:namedport") // Invalid URL
	_, err := store.doJSON(context.Background(), http.MethodGet, "/test", nil)
	require.Error(t, err)
}

// errorReadCloser is a test io.ReadCloser that always returns an error
type errorReadCloser struct{}

func (errorReadCloser) Read(p []byte) (n int, err error) {
	return 0, assert.AnError
}

func (errorReadCloser) Close() error { return nil }

// mockHTTPClient implements HTTPClient for testing
type mockHTTPClient struct {
	DoFunc func(req *http.Request) (*http.Response, error)
}

func (m *mockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return m.DoFunc(req)
}

func TestStore_DoJSON_ReadBodyError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	// Custom HTTP client that returns a response with unreadable body
	customClient := &mockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       errorReadCloser{},
				Header:     make(http.Header),
			}, nil
		},
	}

	store := New(srv.URL, WithHTTPClient(customClient))
	_, err := store.doJSON(context.Background(), http.MethodGet, "/test", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "read response")
}
