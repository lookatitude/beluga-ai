package cohere

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

func cohereResponse(embeddings [][]float32) string {
	resp := map[string]any{
		"id": "emb-test-123",
		"embeddings": map[string]any{
			"float": embeddings,
		},
		"texts": []string{},
		"meta": map[string]any{
			"api_version": map[string]any{"version": "2"},
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
	assert.Equal(t, "search_document", emb.inputType)
}

func TestNew_CustomModel(t *testing.T) {
	emb, err := New(config.ProviderConfig{
		APIKey: "test-key",
		Model:  "embed-multilingual-v3.0",
	})
	require.NoError(t, err)
	assert.Equal(t, "embed-multilingual-v3.0", emb.model)
	assert.Equal(t, 1024, emb.Dimensions())
}

func TestNew_CustomDimensions(t *testing.T) {
	emb, err := New(config.ProviderConfig{
		APIKey: "test-key",
		Options: map[string]any{
			"dimensions": float64(512),
		},
	})
	require.NoError(t, err)
	assert.Equal(t, 512, emb.Dimensions())
}

func TestNew_InputType(t *testing.T) {
	emb, err := New(config.ProviderConfig{
		APIKey: "test-key",
		Options: map[string]any{
			"input_type": "search_query",
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "search_query", emb.inputType)
}

func TestEmbed_Batch(t *testing.T) {
	ts := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Contains(t, r.URL.Path, "embed")

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, cohereResponse([][]float32{
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
		fmt.Fprint(w, cohereResponse([][]float32{
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
		{"english-v3", "embed-english-v3.0", 1024},
		{"multilingual-v3", "embed-multilingual-v3.0", 1024},
		{"english-light-v3", "embed-english-light-v3.0", 384},
		{"english-v2", "embed-english-v2.0", 4096},
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
		fmt.Fprint(w, cohereResponse([][]float32{{0.1}}))
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
		fmt.Fprint(w, `{"message":"invalid api token"}`)
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

		// Check authorization header
		assert.Contains(t, r.Header.Get("Authorization"), "Bearer")

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, cohereResponse([][]float32{{0.1}}))
	})

	emb, err := New(config.ProviderConfig{
		APIKey:  "test-key",
		BaseURL: ts.URL,
	})
	require.NoError(t, err)

	_, err = emb.Embed(context.Background(), []string{"test input"})
	require.NoError(t, err)

	assert.Equal(t, defaultModel, receivedBody.Model)
	assert.Equal(t, []string{"test input"}, receivedBody.Texts)
	assert.Equal(t, "search_document", receivedBody.InputType)
	assert.Equal(t, []string{"float"}, receivedBody.EmbeddingTypes)
}

func TestRegistry_Integration(t *testing.T) {
	names := embedding.List()
	found := false
	for _, name := range names {
		if name == "cohere" {
			found = true
			break
		}
	}
	assert.True(t, found, "cohere provider should be registered")

	emb, err := embedding.New("cohere", config.ProviderConfig{APIKey: "test-key"})
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
		fmt.Fprint(w, cohereResponse([][]float32{{0.1}}))
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

func TestEmbedSingle_ErrorFromEmbed(t *testing.T) {
	ts := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		fmt.Fprint(w, `{"message":"rate limit exceeded"}`)
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

func TestNew_EmptyInputType(t *testing.T) {
	emb, err := New(config.ProviderConfig{
		APIKey: "test-key",
		Options: map[string]any{
			"input_type": "",
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "search_document", emb.inputType)
}
