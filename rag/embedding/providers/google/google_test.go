package google

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/lookatitude/beluga-ai/v2/config"
	"github.com/lookatitude/beluga-ai/v2/core"
	"github.com/lookatitude/beluga-ai/v2/rag/embedding"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func batchResponse(embeddings [][]float32) string {
	values := make([]map[string]any, len(embeddings))
	for i, emb := range embeddings {
		values[i] = map[string]any{
			"values": emb,
		}
	}
	resp := map[string]any{
		"embeddings": values,
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

	assert.Equal(t, "gemini-embedding-001", emb.model)
	assert.Equal(t, 3072, emb.Dimensions())
}

func TestNew_CustomModel(t *testing.T) {
	emb, err := New(config.ProviderConfig{
		APIKey: "test-key",
		Model:  "embedding-001",
	})
	require.NoError(t, err)
	assert.Equal(t, "embedding-001", emb.model)
	assert.Equal(t, 768, emb.Dimensions())
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

func TestEmbed_Batch(t *testing.T) {
	ts := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Contains(t, r.URL.Path, "batchEmbedContents")

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, batchResponse([][]float32{
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
		fmt.Fprint(w, batchResponse([][]float32{
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
		{"gemini", "gemini-embedding-001", 3072},
		{"004", "text-embedding-004", 768},
		{"001", "embedding-001", 768},
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
		fmt.Fprint(w, batchResponse([][]float32{{0.1}}))
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
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `{"error":{"message":"Invalid request","status":"INVALID_ARGUMENT"}}`)
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
	var receivedBody batchEmbedRequest
	ts := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedBody)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, batchResponse([][]float32{{0.1}, {0.2}}))
	})

	emb, err := New(config.ProviderConfig{
		APIKey:  "test-key",
		BaseURL: ts.URL,
	})
	require.NoError(t, err)

	_, err = emb.Embed(context.Background(), []string{"hello", "world"})
	require.NoError(t, err)

	require.Len(t, receivedBody.Requests, 2)
	assert.Equal(t, "models/"+defaultModel, receivedBody.Requests[0].Model)
	assert.Equal(t, "hello", receivedBody.Requests[0].Content.Parts[0].Text)
	assert.Equal(t, "world", receivedBody.Requests[1].Content.Parts[0].Text)
}

func TestRegistry_Integration(t *testing.T) {
	names := embedding.List()
	found := false
	for _, name := range names {
		if name == "google" {
			found = true
			break
		}
	}
	assert.True(t, found, "google provider should be registered")

	emb, err := embedding.New("google", config.ProviderConfig{APIKey: "test-key"})
	require.NoError(t, err)
	assert.Equal(t, defaultDimensions, emb.Dimensions())
}

func TestInterfaceCompliance(t *testing.T) {
	var _ embedding.Embedder = (*Embedder)(nil)
}

func TestEmbed_APIKeyInHeaderNotURL(t *testing.T) {
	var (
		receivedQuery  string
		receivedHeader string
	)
	ts := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		receivedQuery = r.URL.RawQuery
		receivedHeader = r.Header.Get("x-goog-api-key")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, batchResponse([][]float32{{0.1}}))
	})

	emb, err := New(config.ProviderConfig{
		APIKey:  "my-api-key",
		BaseURL: ts.URL,
	})
	require.NoError(t, err)

	_, err = emb.Embed(context.Background(), []string{"hello"})
	require.NoError(t, err)
	assert.Empty(t, receivedQuery, "API key must not be sent as a query parameter")
	assert.NotContains(t, receivedQuery, "key")
	assert.Equal(t, "my-api-key", receivedHeader)
}

func TestEmbed_UnreachableHostErrorDoesNotLeakKey(t *testing.T) {
	const secret = "super-secret-google-key"
	emb, err := New(config.ProviderConfig{
		APIKey:  secret,
		BaseURL: "http://127.0.0.1:1",
	})
	require.NoError(t, err)

	_, err = emb.Embed(context.Background(), []string{"hello"})
	require.Error(t, err)
	assert.NotContains(t, err.Error(), secret)
}

func TestEmbed_ModelPathEscaped(t *testing.T) {
	tests := []struct {
		name  string
		model string
	}{
		{"traversal", "../../admin"},
		{"query", "evil?key=x"},
		{"fragment", "m#frag"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				rawPath  string
				rawQuery string
			)
			ts := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
				rawPath = r.URL.EscapedPath()
				rawQuery = r.URL.RawQuery
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprint(w, batchResponse([][]float32{{0.1}}))
			})

			emb, err := New(config.ProviderConfig{
				APIKey:  "test-key",
				BaseURL: ts.URL,
				Model:   tt.model,
			})
			require.NoError(t, err)

			_, err = emb.Embed(context.Background(), []string{"hello"})
			require.NoError(t, err)
			assert.Equal(t, "/models/"+url.PathEscape(tt.model)+":batchEmbedContents", rawPath)
			assert.Empty(t, rawQuery)
		})
	}
}

func TestEmbed_MalformedResponse(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{"too_few", batchResponse([][]float32{{0.1}})},
		{"too_many", batchResponse([][]float32{{0.1}, {0.2}, {0.3}})},
		{"missing_values", `{"embeddings":[{"values":[0.1]},{}]}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprint(w, tt.body)
			})
			emb, err := New(config.ProviderConfig{APIKey: "test-key", BaseURL: ts.URL})
			require.NoError(t, err)

			_, err = emb.Embed(context.Background(), []string{"a", "b"})
			require.Error(t, err)
			var cerr *core.Error
			require.ErrorAs(t, err, &cerr)
			assert.Equal(t, core.ErrProviderDown, cerr.Code)
		})
	}
}

func TestEmbedSingle_ErrorFromEmbed(t *testing.T) {
	ts := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, `{"error":{"message":"Internal server error"}}`)
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

func TestEmbed_InvalidJSONResponse(t *testing.T) {
	ts := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, "invalid json")
	})

	emb, err := New(config.ProviderConfig{
		APIKey:  "test-key",
		BaseURL: ts.URL,
	})
	require.NoError(t, err)

	_, err = emb.Embed(context.Background(), []string{"hello"})
	assert.Error(t, err)
}

func TestEmbed_OutputDimensionalitySent(t *testing.T) {
	tests := []struct {
		name string
		opts map[string]any
		want any
	}{
		{"configured", map[string]any{"dimensions": float64(256)}, float64(256)},
		{"not configured", nil, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got any
			ts := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
				var body map[string][]map[string]any
				_ = json.NewDecoder(r.Body).Decode(&body)
				got = body["requests"][0]["outputDimensionality"]
				_, _ = w.Write([]byte(`{"embeddings":[{"values":[0.1]}]}`))
			})
			emb, err := New(config.ProviderConfig{APIKey: "k", BaseURL: ts.URL, Options: tt.opts})
			require.NoError(t, err)
			_, err = emb.Embed(context.Background(), []string{"a"})
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
