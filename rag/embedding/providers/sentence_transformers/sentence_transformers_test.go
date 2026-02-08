package sentencetransformers

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

func TestNew_MissingAPIKey(t *testing.T) {
	_, err := New(config.ProviderConfig{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "api_key")
}

func TestNew_CustomModel(t *testing.T) {
	emb, err := New(config.ProviderConfig{
		APIKey: "test-key",
		Model:  "sentence-transformers/all-mpnet-base-v2",
	})
	require.NoError(t, err)
	assert.Equal(t, "sentence-transformers/all-mpnet-base-v2", emb.model)
	assert.Equal(t, 768, emb.Dimensions())
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

func TestDimensions_KnownModels(t *testing.T) {
	tests := []struct {
		name     string
		model    string
		expected int
	}{
		{"default", "", defaultDimensions},
		{"all-MiniLM-L6-v2", "sentence-transformers/all-MiniLM-L6-v2", 384},
		{"all-MiniLM-L12-v2", "sentence-transformers/all-MiniLM-L12-v2", 384},
		{"all-mpnet-base-v2", "sentence-transformers/all-mpnet-base-v2", 768},
		{"bge-small-en", "BAAI/bge-small-en-v1.5", 384},
		{"bge-base-en", "BAAI/bge-base-en-v1.5", 768},
		{"bge-large-en", "BAAI/bge-large-en-v1.5", 1024},
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

func TestEmbed_Batch(t *testing.T) {
	ts := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Contains(t, r.URL.Path, "pipeline/feature-extraction")

		w.Header().Set("Content-Type", "application/json")
		resp := [][]float32{
			{0.1, 0.2, 0.3},
			{0.4, 0.5, 0.6},
		}
		json.NewEncoder(w).Encode(resp)
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
		resp := [][]float32{{0.7, 0.8, 0.9}}
		json.NewEncoder(w).Encode(resp)
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

func TestEmbed_RequestBody(t *testing.T) {
	var receivedBody embedRequest
	ts := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedBody)

		assert.Contains(t, r.Header.Get("Authorization"), "Bearer")
		assert.Contains(t, r.URL.Path, defaultModel)

		w.Header().Set("Content-Type", "application/json")
		resp := [][]float32{{0.1}}
		json.NewEncoder(w).Encode(resp)
	})

	emb, err := New(config.ProviderConfig{
		APIKey:  "test-key",
		BaseURL: ts.URL,
	})
	require.NoError(t, err)

	_, err = emb.Embed(context.Background(), []string{"test input"})
	require.NoError(t, err)

	assert.Equal(t, []string{"test input"}, receivedBody.Inputs)
}

func TestEmbed_ContextCancelled(t *testing.T) {
	ts := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := [][]float32{{0.1}}
		json.NewEncoder(w).Encode(resp)
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
		fmt.Fprint(w, `{"error":"invalid api key"}`)
	})

	emb, err := New(config.ProviderConfig{
		APIKey:  "bad-key",
		BaseURL: ts.URL,
	})
	require.NoError(t, err)

	_, err = emb.Embed(context.Background(), []string{"hello"})
	assert.Error(t, err)
}

func TestEmbed_AuthorizationHeader(t *testing.T) {
	var authHeader string
	ts := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		authHeader = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		resp := [][]float32{{0.1}}
		json.NewEncoder(w).Encode(resp)
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

func TestEmbed_CustomModelPath(t *testing.T) {
	var requestPath string
	ts := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		requestPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		resp := [][]float32{{0.1}}
		json.NewEncoder(w).Encode(resp)
	})

	emb, err := New(config.ProviderConfig{
		APIKey:  "test-key",
		BaseURL: ts.URL,
		Model:   "BAAI/bge-large-en-v1.5",
	})
	require.NoError(t, err)

	_, err = emb.Embed(context.Background(), []string{"hello"})
	require.NoError(t, err)
	assert.Contains(t, requestPath, "pipeline/feature-extraction/BAAI/bge-large-en-v1.5")
}

func TestRegistry_Integration(t *testing.T) {
	names := embedding.List()
	found := false
	for _, name := range names {
		if name == "sentence_transformers" {
			found = true
			break
		}
	}
	assert.True(t, found, "sentence_transformers provider should be registered")

	emb, err := embedding.New("sentence_transformers", config.ProviderConfig{APIKey: "test-key"})
	require.NoError(t, err)
	assert.Equal(t, defaultDimensions, emb.Dimensions())
}

func TestInterfaceCompliance(t *testing.T) {
	var _ embedding.Embedder = (*Embedder)(nil)
}
