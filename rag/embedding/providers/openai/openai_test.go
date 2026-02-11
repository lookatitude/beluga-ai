package openai

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

func embeddingResponse(embeddings [][]float64) string {
	data := make([]map[string]any, len(embeddings))
	for i, emb := range embeddings {
		data[i] = map[string]any{
			"object":    "embedding",
			"index":     i,
			"embedding": emb,
		}
	}
	resp := map[string]any{
		"object": "list",
		"data":   data,
		"model":  "text-embedding-3-small",
		"usage": map[string]any{
			"prompt_tokens": 8,
			"total_tokens":  8,
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
	emb, err := New(config.ProviderConfig{APIKey: "sk-test"})
	require.NoError(t, err)
	require.NotNil(t, emb)

	assert.Equal(t, defaultModel, emb.model)
	assert.Equal(t, defaultDimensions, emb.Dimensions())
}

func TestNew_CustomModel(t *testing.T) {
	emb, err := New(config.ProviderConfig{
		APIKey: "sk-test",
		Model:  "text-embedding-3-large",
	})
	require.NoError(t, err)
	assert.Equal(t, "text-embedding-3-large", emb.model)
	assert.Equal(t, 3072, emb.Dimensions())
}

func TestNew_CustomDimensions(t *testing.T) {
	emb, err := New(config.ProviderConfig{
		APIKey: "sk-test",
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
		fmt.Fprint(w, embeddingResponse([][]float64{
			{0.1, 0.2, 0.3},
			{0.4, 0.5, 0.6},
		}))
	})

	emb, err := New(config.ProviderConfig{
		APIKey:  "sk-test",
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
		fmt.Fprint(w, embeddingResponse([][]float64{
			{0.1, 0.2, 0.3},
		}))
	})

	emb, err := New(config.ProviderConfig{
		APIKey:  "sk-test",
		BaseURL: ts.URL,
	})
	require.NoError(t, err)

	vec, err := emb.EmbedSingle(context.Background(), "hello")
	require.NoError(t, err)
	require.Len(t, vec, 3)
	assert.InDelta(t, float32(0.1), vec[0], 0.001)
}

func TestEmbed_Empty(t *testing.T) {
	emb, err := New(config.ProviderConfig{APIKey: "sk-test"})
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
		{"default", "", 1536},
		{"ada-002", "text-embedding-ada-002", 1536},
		{"3-small", "text-embedding-3-small", 1536},
		{"3-large", "text-embedding-3-large", 3072},
		{"unknown model", "custom-model", 1536},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			emb, err := New(config.ProviderConfig{
				APIKey: "sk-test",
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
		fmt.Fprint(w, embeddingResponse([][]float64{{0.1}}))
	})

	emb, err := New(config.ProviderConfig{
		APIKey:  "sk-test",
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
		fmt.Fprint(w, `{"error":{"message":"Invalid API key","type":"invalid_request_error"}}`)
	})

	emb, err := New(config.ProviderConfig{
		APIKey:  "bad-key",
		BaseURL: ts.URL,
	})
	require.NoError(t, err)

	_, err = emb.Embed(context.Background(), []string{"hello"})
	assert.Error(t, err)
}

func TestRegistry_Integration(t *testing.T) {
	names := embedding.List()
	found := false
	for _, name := range names {
		if name == "openai" {
			found = true
			break
		}
	}
	assert.True(t, found, "openai provider should be registered")

	emb, err := embedding.New("openai", config.ProviderConfig{APIKey: "sk-test"})
	require.NoError(t, err)
	assert.Equal(t, defaultDimensions, emb.Dimensions())
}

func TestInterfaceCompliance(t *testing.T) {
	var _ embedding.Embedder = (*Embedder)(nil)
}

func TestEmbed_RequestBody(t *testing.T) {
	var receivedBody map[string]any
	ts := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedBody)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, embeddingResponse([][]float64{{0.1}}))
	})

	emb, err := New(config.ProviderConfig{
		APIKey:  "sk-test",
		BaseURL: ts.URL,
	})
	require.NoError(t, err)

	_, err = emb.Embed(context.Background(), []string{"test input"})
	require.NoError(t, err)

	assert.Equal(t, "text-embedding-3-small", receivedBody["model"])
	assert.Equal(t, "float", receivedBody["encoding_format"])
}

func TestEmbed_MultipleResults(t *testing.T) {
	ts := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, embeddingResponse([][]float64{
			{0.1, 0.2},
			{0.3, 0.4},
			{0.5, 0.6},
		}))
	})

	emb, err := New(config.ProviderConfig{
		APIKey:  "sk-test",
		BaseURL: ts.URL,
	})
	require.NoError(t, err)

	vecs, err := emb.Embed(context.Background(), []string{"a", "b", "c"})
	require.NoError(t, err)
	require.Len(t, vecs, 3)

	assert.InDelta(t, float32(0.1), vecs[0][0], 0.001)
	assert.InDelta(t, float32(0.3), vecs[1][0], 0.001)
	assert.InDelta(t, float32(0.5), vecs[2][0], 0.001)
}

func TestEmbedSingle_ErrorFromEmbed(t *testing.T) {
	ts := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprint(w, `{"error":{"message":"Service temporarily unavailable","type":"server_error"}}`)
	})

	emb, err := New(config.ProviderConfig{
		APIKey:  "sk-test",
		BaseURL: ts.URL,
	})
	require.NoError(t, err)

	_, err = emb.EmbedSingle(context.Background(), "hello")
	assert.Error(t, err)
}

func TestNew_NoAPIKey(t *testing.T) {
	emb, err := New(config.ProviderConfig{
		BaseURL: "https://api.openai.com/v1",
	})
	require.NoError(t, err)
	assert.NotNil(t, emb)
}

func TestNew_CustomTimeout(t *testing.T) {
	emb, err := New(config.ProviderConfig{
		APIKey:  "sk-test",
		Timeout: 30000000000,
	})
	require.NoError(t, err)
	assert.NotNil(t, emb)
}

func TestEmbed_OutOfBoundsIndex(t *testing.T) {
	ts := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := map[string]any{
			"object": "list",
			"data": []map[string]any{
				{"object": "embedding", "index": 0, "embedding": []float64{0.1, 0.2}},
				{"object": "embedding", "index": 99, "embedding": []float64{0.3, 0.4}},
			},
			"model": "text-embedding-3-small",
			"usage": map[string]any{"prompt_tokens": 8, "total_tokens": 8},
		}
		b, _ := json.Marshal(resp)
		fmt.Fprint(w, string(b))
	})

	emb, err := New(config.ProviderConfig{
		APIKey:  "sk-test",
		BaseURL: ts.URL,
	})
	require.NoError(t, err)

	vecs, err := emb.Embed(context.Background(), []string{"first", "second"})
	require.NoError(t, err)
	require.Len(t, vecs, 2)
	assert.NotNil(t, vecs[0])
}
