package jina

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/rag/embedding"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func jinaResponse(embeddings [][]float32) string {
	data := make([]map[string]any, len(embeddings))
	for i, emb := range embeddings {
		data[i] = map[string]any{
			"object":    "embedding",
			"embedding": emb,
			"index":     i,
		}
	}
	resp := map[string]any{
		"object": "list",
		"data":   data,
		"model":  "jina-embeddings-v2-base-en",
		"usage": map[string]any{
			"total_tokens":  10,
			"prompt_tokens": 10,
		},
	}
	b, _ := json.Marshal(resp)
	return string(b)
}

func mockServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	ts := httptest.NewServer(handler)
	t.Cleanup(ts.Close)
	return ts
}

func TestNew_Defaults(t *testing.T) {
	emb, err := New(config.ProviderConfig{APIKey: "test-key"})
	require.NoError(t, err)
	require.NotNil(t, emb)

	assert.Equal(t, defaultModel, emb.model)
	assert.Equal(t, defaultDimensions, emb.Dimensions())
}

func TestNew_CustomModel(t *testing.T) {
	emb, err := New(config.ProviderConfig{
		APIKey: "test-key",
		Model:  "jina-embeddings-v3",
	})
	require.NoError(t, err)
	assert.Equal(t, "jina-embeddings-v3", emb.model)
	assert.Equal(t, 1024, emb.Dimensions())
}

func TestNew_CustomDimensions(t *testing.T) {
	emb, err := New(config.ProviderConfig{
		APIKey: "test-key",
		Options: map[string]any{
			"dimensions": float64(256),
		},
	})
	require.NoError(t, err)
	assert.Equal(t, 256, emb.Dimensions())
}

func TestEmbed_Batch(t *testing.T) {
	ts := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Contains(t, r.URL.Path, "embeddings")

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, jinaResponse([][]float32{
			{0.1, 0.2, 0.3},
			{0.4, 0.5, 0.6},
		}))
	})

	emb, err := New(config.ProviderConfig{
		APIKey:  "test-key",
		BaseURL: ts.URL,
	})
	require.NoError(t, err)

	vecs, err := emb.Embed(context.Background(), []string{"hello", "world"})
	require.NoError(t, err)
	require.Len(t, vecs, 2)

	assert.InDelta(t, float32(0.1), vecs[0][0], 0.001)
	assert.InDelta(t, float32(0.4), vecs[1][0], 0.001)
}

func TestEmbedSingle(t *testing.T) {
	ts := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, jinaResponse([][]float32{
			{0.7, 0.8, 0.9},
		}))
	})

	emb, err := New(config.ProviderConfig{
		APIKey:  "test-key",
		BaseURL: ts.URL,
	})
	require.NoError(t, err)

	vec, err := emb.EmbedSingle(context.Background(), "hello")
	require.NoError(t, err)
	require.Len(t, vec, 3)
	assert.InDelta(t, float32(0.7), vec[0], 0.001)
}

func TestEmbed_Empty(t *testing.T) {
	emb, err := New(config.ProviderConfig{APIKey: "test-key"})
	require.NoError(t, err)

	vecs, err := emb.Embed(context.Background(), []string{})
	require.NoError(t, err)
	assert.Len(t, vecs, 0)
}

func TestDimensions(t *testing.T) {
	tests := []struct {
		name     string
		model    string
		expected int
	}{
		{"default", "", defaultDimensions},
		{"v2-base-en", "jina-embeddings-v2-base-en", 768},
		{"v2-small-en", "jina-embeddings-v2-small-en", 512},
		{"v2-base-de", "jina-embeddings-v2-base-de", 768},
		{"v3", "jina-embeddings-v3", 1024},
		{"unknown", "custom-model", defaultDimensions},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			emb, err := New(config.ProviderConfig{
				APIKey: "test-key",
				Model:  tt.model,
			})
			require.NoError(t, err)
			assert.Equal(t, tt.expected, emb.Dimensions())
		})
	}
}

func TestEmbed_ContextCancelled(t *testing.T) {
	ts := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, jinaResponse([][]float32{{0.1}}))
	})

	emb, err := New(config.ProviderConfig{
		APIKey:  "test-key",
		BaseURL: ts.URL,
	})
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = emb.Embed(ctx, []string{"hello"})
	assert.Error(t, err)
}

func TestEmbed_ErrorResponse(t *testing.T) {
	ts := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `{"message":"invalid api key"}`)
	})

	emb, err := New(config.ProviderConfig{
		APIKey:  "bad-key",
		BaseURL: ts.URL,
	})
	require.NoError(t, err)

	_, err = emb.Embed(context.Background(), []string{"hello"})
	assert.Error(t, err)
}

func TestEmbed_RequestBody(t *testing.T) {
	var receivedBody embedRequest
	ts := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedBody)

		assert.Contains(t, r.Header.Get("Authorization"), "Bearer")

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, jinaResponse([][]float32{{0.1}}))
	})

	emb, err := New(config.ProviderConfig{
		APIKey:  "test-key",
		BaseURL: ts.URL,
	})
	require.NoError(t, err)

	_, err = emb.Embed(context.Background(), []string{"test input"})
	require.NoError(t, err)

	assert.Equal(t, defaultModel, receivedBody.Model)
	assert.Equal(t, []string{"test input"}, receivedBody.Input)
}

func TestRegistry_Integration(t *testing.T) {
	names := embedding.List()
	found := false
	for _, name := range names {
		if name == "jina" {
			found = true
			break
		}
	}
	assert.True(t, found, "jina provider should be registered")

	emb, err := embedding.New("jina", config.ProviderConfig{APIKey: "test-key"})
	require.NoError(t, err)
	assert.Equal(t, defaultDimensions, emb.Dimensions())
}

func TestInterfaceCompliance(t *testing.T) {
	var _ embedding.Embedder = (*Embedder)(nil)
}

func TestEmbed_AuthorizationHeader(t *testing.T) {
	var authHeader string
	ts := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		authHeader = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, jinaResponse([][]float32{{0.1}}))
	})

	emb, err := New(config.ProviderConfig{
		APIKey:  "my-secret-key",
		BaseURL: ts.URL,
	})
	require.NoError(t, err)

	_, err = emb.Embed(context.Background(), []string{"hello"})
	require.NoError(t, err)
	assert.Equal(t, "Bearer my-secret-key", authHeader)
}

func TestEmbed_OutOfOrderIndices(t *testing.T) {
	ts := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"object": "list",
			"data": []map[string]any{
				{"object": "embedding", "embedding": []float32{0.4, 0.5}, "index": 1},
				{"object": "embedding", "embedding": []float32{0.1, 0.2}, "index": 0},
			},
			"model": "jina-embeddings-v2-base-en",
			"usage": map[string]any{"total_tokens": 5, "prompt_tokens": 5},
		}
		json.NewEncoder(w).Encode(resp)
	})

	emb, err := New(config.ProviderConfig{
		APIKey:  "test-key",
		BaseURL: ts.URL,
	})
	require.NoError(t, err)

	vecs, err := emb.Embed(context.Background(), []string{"first", "second"})
	require.NoError(t, err)
	require.Len(t, vecs, 2)

	assert.InDelta(t, float32(0.1), vecs[0][0], 0.001)
	assert.InDelta(t, float32(0.4), vecs[1][0], 0.001)
}

func TestEmbedSingle_ErrorFromEmbed(t *testing.T) {
	ts := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprint(w, `{"message":"api key invalid"}`)
	})

	emb, err := New(config.ProviderConfig{
		APIKey:  "test-key",
		BaseURL: ts.URL,
	})
	require.NoError(t, err)

	_, err = emb.EmbedSingle(context.Background(), "hello")
	assert.Error(t, err)
}

func TestNew_CustomTimeout(t *testing.T) {
	emb, err := New(config.ProviderConfig{
		APIKey:  "test-key",
		Timeout: 30000000000,
	})
	require.NoError(t, err)
	assert.NotNil(t, emb)
}

func TestEmbed_IndexOutOfBounds(t *testing.T) {
	ts := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"object": "list",
			"data": []map[string]any{
				{"object": "embedding", "embedding": []float32{0.1, 0.2}, "index": 0},
				{"object": "embedding", "embedding": []float32{0.3, 0.4}, "index": 99},
			},
			"model": "jina-embeddings-v2-base-en",
			"usage": map[string]any{"total_tokens": 5, "prompt_tokens": 5},
		}
		json.NewEncoder(w).Encode(resp)
	})

	emb, err := New(config.ProviderConfig{
		APIKey:  "test-key",
		BaseURL: ts.URL,
	})
	require.NoError(t, err)

	vecs, err := emb.Embed(context.Background(), []string{"first", "second"})
	require.NoError(t, err)
	require.Len(t, vecs, 2)
	assert.NotNil(t, vecs[0])
}
