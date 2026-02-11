package vespa

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
		WithNamespace("test_ns"),
		WithDocType("test_doc"),
		WithHTTPClient(srv.Client()),
	)
	return srv, store
}

func TestNew(t *testing.T) {
	store := New("http://localhost:8080",
		WithNamespace("my_ns"),
		WithDocType("my_doc"),
	)
	require.NotNil(t, store)
	assert.Equal(t, "http://localhost:8080", store.baseURL)
	assert.Equal(t, "my_ns", store.namespace)
	assert.Equal(t, "my_doc", store.docType)
}

func TestNew_Defaults(t *testing.T) {
	store := New("http://localhost:8080")
	assert.Equal(t, "default", store.namespace)
	assert.Equal(t, "document", store.docType)
}

func TestStore_InterfaceCompliance(t *testing.T) {
	var _ vectorstore.VectorStore = (*Store)(nil)
}

func TestStore_Add(t *testing.T) {
	callCount := 0
	var receivedPaths []string
	srv, store := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPut, r.Method)
		receivedPaths = append(receivedPaths, r.URL.Path)
		callCount++

		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		fields := body["fields"].(map[string]any)
		assert.NotNil(t, fields["content"])
		assert.NotNil(t, fields["embedding"])

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"doc"}`))
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
	assert.Equal(t, 2, callCount)
	assert.Contains(t, receivedPaths[0], "/document/v1/test_ns/test_doc/docid/doc1")
	assert.Contains(t, receivedPaths[1], "/document/v1/test_ns/test_doc/docid/doc2")
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
}

func TestStore_Add_WithMetadata(t *testing.T) {
	var receivedFields map[string]any
	srv, store := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		receivedFields = body["fields"].(map[string]any)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"doc1"}`))
	})
	defer srv.Close()

	docs := []schema.Document{
		{ID: "doc1", Content: "hello", Metadata: map[string]any{"category": "A", "priority": float64(1)}},
	}
	err := store.Add(context.Background(), docs, [][]float32{{0.1, 0.2}})
	require.NoError(t, err)

	assert.Equal(t, "A", receivedFields["category"])
	assert.Equal(t, float64(1), receivedFields["priority"])
}

func TestStore_Search(t *testing.T) {
	srv, store := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Contains(t, r.URL.Path, "/search/")

		// Verify query parameter.
		yql := r.URL.Query().Get("yql")
		assert.Contains(t, yql, "nearestNeighbor")

		resp := map[string]any{
			"root": map[string]any{
				"children": []map[string]any{
					{
						"id":        "doc1",
						"relevance": 0.95,
						"fields": map[string]any{
							"content":  "hello world",
							"category": "A",
						},
					},
					{
						"id":        "doc2",
						"relevance": 0.80,
						"fields": map[string]any{
							"content": "goodbye",
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
	assert.Equal(t, 0.95, results[0].Score)
	assert.Equal(t, "A", results[0].Metadata["category"])

	assert.Equal(t, "doc2", results[1].ID)
	assert.Equal(t, 0.80, results[1].Score)
}

func TestStore_Search_WithThreshold(t *testing.T) {
	srv, store := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"root": map[string]any{
				"children": []map[string]any{
					{"id": "doc1", "relevance": 0.95, "fields": map[string]any{"content": "hello"}},
					{"id": "doc2", "relevance": 0.50, "fields": map[string]any{"content": "world"}},
				},
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
		resp := map[string]any{
			"root": map[string]any{
				"children": []any{},
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

func TestStore_Delete(t *testing.T) {
	callCount := 0
	srv, store := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method)
		assert.Contains(t, r.URL.Path, "/document/v1/test_ns/test_doc/docid/")
		callCount++
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"deleted"}`))
	})
	defer srv.Close()

	err := store.Delete(context.Background(), []string{"doc1", "doc2"})
	require.NoError(t, err)
	assert.Equal(t, 2, callCount)
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
}

func TestStore_ContextCancelled(t *testing.T) {
	srv, store := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"root": map[string]any{"children": []any{}},
		}
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
	assert.Contains(t, names, "vespa")
}

func TestNewFromConfig_MissingBaseURL(t *testing.T) {
	_, err := NewFromConfig(config.ProviderConfig{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "base_url")
}

func TestNewFromConfig(t *testing.T) {
	store, err := NewFromConfig(config.ProviderConfig{
		BaseURL: "http://localhost:8080",
		Options: map[string]any{
			"namespace": "my_ns",
			"doc_type":  "my_doc",
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "http://localhost:8080", store.baseURL)
	assert.Equal(t, "my_ns", store.namespace)
	assert.Equal(t, "my_doc", store.docType)
}

func TestNewFromConfig_RegistryFactory(t *testing.T) {
	// Test that the init() registry factory works
	vs, err := vectorstore.New("vespa", config.ProviderConfig{
		BaseURL: "http://localhost:8080",
	})
	require.NoError(t, err)
	assert.NotNil(t, vs)
}

func TestNewFromConfig_RegistryFactory_Error(t *testing.T) {
	// Test that the init() registry factory returns error for missing base_url
	_, err := vectorstore.New("vespa", config.ProviderConfig{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "base_url")
}

func TestStore_Search_WithDotProductStrategy(t *testing.T) {
	srv, store := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		ranking := r.URL.Query().Get("ranking")
		assert.Equal(t, "dotProduct(embedding)", ranking)

		resp := map[string]any{
			"root": map[string]any{
				"children": []map[string]any{
					{
						"id":        "doc1",
						"relevance": 0.95,
						"fields":    map[string]any{"content": "test"},
					},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	})
	defer srv.Close()

	results, err := store.Search(context.Background(), []float32{0.1, 0.2}, 5,
		vectorstore.WithStrategy(vectorstore.DotProduct))
	require.NoError(t, err)
	assert.Len(t, results, 1)
}

func TestStore_Search_WithEuclideanStrategy(t *testing.T) {
	srv, store := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		ranking := r.URL.Query().Get("ranking")
		assert.Equal(t, "euclidean(embedding)", ranking)

		resp := map[string]any{
			"root": map[string]any{
				"children": []map[string]any{
					{
						"id":        "doc1",
						"relevance": 0.85,
						"fields":    map[string]any{"content": "test"},
					},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	})
	defer srv.Close()

	results, err := store.Search(context.Background(), []float32{0.1, 0.2}, 5,
		vectorstore.WithStrategy(vectorstore.Euclidean))
	require.NoError(t, err)
	assert.Len(t, results, 1)
}

func TestStore_Search_WithFilter(t *testing.T) {
	srv, store := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		yql := r.URL.Query().Get("yql")
		assert.Contains(t, yql, `category = "A"`)
		assert.Contains(t, yql, `priority = "high"`)

		resp := map[string]any{
			"root": map[string]any{
				"children": []map[string]any{
					{
						"id":        "doc1",
						"relevance": 0.95,
						"fields": map[string]any{
							"content":  "filtered",
							"category": "A",
							"priority": "high",
						},
					},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	})
	defer srv.Close()

	results, err := store.Search(context.Background(), []float32{0.1, 0.2}, 5,
		vectorstore.WithFilter(map[string]any{"category": "A", "priority": "high"}))
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "doc1", results[0].ID)
}

func TestStore_Search_InvalidJSON(t *testing.T) {
	srv, store := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{invalid json`))
	})
	defer srv.Close()

	_, err := store.Search(context.Background(), []float32{0.1, 0.2}, 5)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshal")
}

func TestStore_Search_NoMetadata(t *testing.T) {
	srv, store := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"root": map[string]any{
				"children": []map[string]any{
					{
						"id":        "doc1",
						"relevance": 0.95,
						"fields": map[string]any{
							"content":   "only content",
							"embedding": []float64{0.1, 0.2},
						},
					},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	})
	defer srv.Close()

	results, err := store.Search(context.Background(), []float32{0.1, 0.2}, 5)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "only content", results[0].Content)
	assert.Nil(t, results[0].Metadata)
}

func TestStore_doPut_MarshalError(t *testing.T) {
	store := New("http://localhost:8080")
	// Use a value that cannot be marshaled (e.g., a channel)
	invalidBody := map[string]any{
		"fields": map[string]any{
			"invalid": make(chan int),
		},
	}
	err := store.doPut(context.Background(), "/test", invalidBody)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "marshal")
}

func TestFloat32SliceToFloat64(t *testing.T) {
	tests := []struct {
		name  string
		input []float32
	}{
		{name: "empty slice", input: []float32{}},
		{name: "single value", input: []float32{0.5}},
		{name: "multiple values", input: []float32{0.125, 0.25, 0.5}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := float32SliceToFloat64(tt.input)
			assert.Len(t, got, len(tt.input))
			for i, v := range tt.input {
				assert.Equal(t, float64(v), got[i])
			}
		})
	}
}

func TestFormatVectorParam(t *testing.T) {
	tests := []struct {
		name  string
		input []float32
		want  string
	}{
		{
			name:  "empty vector",
			input: []float32{},
			want:  "[]",
		},
		{
			name:  "single value",
			input: []float32{0.5},
			want:  "[0.5]",
		},
		{
			name:  "multiple values",
			input: []float32{0.1, 0.2, 0.3},
			want:  "[0.1,0.2,0.3]",
		},
		{
			name:  "negative values",
			input: []float32{-0.1, 0.2, -0.3},
			want:  "[-0.1,0.2,-0.3]",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatVectorParam(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDocPath(t *testing.T) {
	tests := []struct {
		name      string
		namespace string
		docType   string
		id        string
		want      string
	}{
		{
			name:      "simple id",
			namespace: "ns1",
			docType:   "doc1",
			id:        "abc123",
			want:      "/document/v1/ns1/doc1/docid/abc123",
		},
		{
			name:      "id with special chars",
			namespace: "ns1",
			docType:   "doc1",
			id:        "id with spaces",
			want:      "/document/v1/ns1/doc1/docid/id%20with%20spaces",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := New("http://localhost:8080",
				WithNamespace(tt.namespace),
				WithDocType(tt.docType),
			)
			got := store.docPath(tt.id)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestStore_Search_MissingContentField(t *testing.T) {
	srv, store := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"root": map[string]any{
				"children": []map[string]any{
					{
						"id":        "doc1",
						"relevance": 0.95,
						"fields": map[string]any{
							"other_field": "value",
						},
					},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	})
	defer srv.Close()

	results, err := store.Search(context.Background(), []float32{0.1, 0.2}, 5)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "", results[0].Content)
	assert.Equal(t, "value", results[0].Metadata["other_field"])
}

func TestStore_Delete_HTTPClientError(t *testing.T) {
	// Use a mock client that returns an error
	store := New("http://localhost:8080")
	store.client = &errorHTTPClient{err: assert.AnError}

	err := store.Delete(context.Background(), []string{"doc1"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "delete")
}

func TestStore_Add_HTTPClientError(t *testing.T) {
	// Use a mock client that returns an error
	store := New("http://localhost:8080")
	store.client = &errorHTTPClient{err: assert.AnError}

	err := store.Add(context.Background(),
		[]schema.Document{{ID: "doc1", Content: "test"}},
		[][]float32{{0.1, 0.2}})
	require.Error(t, err)
}

// errorHTTPClient is a mock client that always returns an error
type errorHTTPClient struct {
	err error
}

func (c *errorHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return nil, c.err
}
