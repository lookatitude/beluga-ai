package ollama

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

func ollamaResponse(embeddings [][]float32) string {
	resp := map[string]any{
		"model":      "nomic-embed-text",
		"embeddings": embeddings,
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
	emb, err := New(config.ProviderConfig{})
	require.NoError(t, err)
	require.NotNil(t, emb)

	assert.Equal(t, defaultModel, emb.model)
	assert.Equal(t, defaultDimensions, emb.Dimensions())
}

func TestNew_CustomModel(t *testing.T) {
	emb, err := New(config.ProviderConfig{
		Model: "mxbai-embed-large",
	})
	require.NoError(t, err)
	assert.Equal(t, "mxbai-embed-large", emb.model)
	assert.Equal(t, 1024, emb.Dimensions())
}

func TestNew_CustomDimensions(t *testing.T) {
	emb, err := New(config.ProviderConfig{
		Options: map[string]any{
			"dimensions": float64(384),
		},
	})
	require.NoError(t, err)
	assert.Equal(t, 384, emb.Dimensions())
}

func TestEmbed_Batch(t *testing.T) {
	callCount := 0
	ts := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Contains(t, r.URL.Path, "api/embed")

		var body embedRequest
		json.NewDecoder(r.Body).Decode(&body)

		callCount++
		vec := []float32{float32(callCount) * 0.1, float32(callCount) * 0.2}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, ollamaResponse([][]float32{vec}))
	})

	emb, err := New(config.ProviderConfig{
		BaseURL: ts.URL,
	})
	require.NoError(t, err)

	vecs, err := emb.Embed(context.Background(), []string{"hello", "world"})
	require.NoError(t, err)
	require.Len(t, vecs, 2)
	assert.Equal(t, 2, callCount, "should make one call per text")
}

func TestEmbedSingle(t *testing.T) {
	ts := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, ollamaResponse([][]float32{
			{0.7, 0.8, 0.9},
		}))
	})

	emb, err := New(config.ProviderConfig{
		BaseURL: ts.URL,
	})
	require.NoError(t, err)

	vec, err := emb.EmbedSingle(context.Background(), "hello")
	require.NoError(t, err)
	require.Len(t, vec, 3)
	assert.InDelta(t, float32(0.7), vec[0], 0.001)
}

func TestEmbed_Empty(t *testing.T) {
	emb, err := New(config.ProviderConfig{})
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
		{"nomic", "nomic-embed-text", 768},
		{"mxbai", "mxbai-embed-large", 1024},
		{"all-minilm", "all-minilm", 384},
		{"unknown", "custom-model", defaultDimensions},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			emb, err := New(config.ProviderConfig{Model: tt.model})
			require.NoError(t, err)
			assert.Equal(t, tt.expected, emb.Dimensions())
		})
	}
}

func TestEmbed_ContextCancelled(t *testing.T) {
	ts := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, ollamaResponse([][]float32{{0.1}}))
	})

	emb, err := New(config.ProviderConfig{
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
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, `{"error":"model not found"}`)
	})

	emb, err := New(config.ProviderConfig{
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
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, ollamaResponse([][]float32{{0.1}}))
	})

	emb, err := New(config.ProviderConfig{
		BaseURL: ts.URL,
	})
	require.NoError(t, err)

	_, err = emb.Embed(context.Background(), []string{"test input"})
	require.NoError(t, err)

	assert.Equal(t, defaultModel, receivedBody.Model)
	assert.Equal(t, "test input", receivedBody.Input)
}

func TestRegistry_Integration(t *testing.T) {
	names := embedding.List()
	found := false
	for _, name := range names {
		if name == "ollama" {
			found = true
			break
		}
	}
	assert.True(t, found, "ollama provider should be registered")

	emb, err := embedding.New("ollama", config.ProviderConfig{})
	require.NoError(t, err)
	assert.Equal(t, defaultDimensions, emb.Dimensions())
}

func TestInterfaceCompliance(t *testing.T) {
	var _ embedding.Embedder = (*Embedder)(nil)
}
