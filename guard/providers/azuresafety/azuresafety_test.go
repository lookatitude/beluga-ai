package azuresafety

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/guard"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		g, err := New(
			WithEndpoint("https://test.cognitiveservices.azure.com"),
			WithAPIKey("test-key"),
		)
		require.NoError(t, err)
		assert.NotNil(t, g)
		assert.Equal(t, "azure_content_safety", g.Name())
	})

	t.Run("missing endpoint", func(t *testing.T) {
		_, err := New(WithAPIKey("test-key"))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "endpoint")
	})

	t.Run("missing api key", func(t *testing.T) {
		_, err := New(WithEndpoint("https://test.cognitiveservices.azure.com"))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "API key")
	})

	t.Run("with options", func(t *testing.T) {
		g, err := New(
			WithEndpoint("https://test.cognitiveservices.azure.com"),
			WithAPIKey("test-key"),
			WithThreshold(4),
			WithTimeout(5*time.Second),
		)
		require.NoError(t, err)
		assert.NotNil(t, g)
		assert.Equal(t, 4, g.threshold)
	})
}

func TestValidate(t *testing.T) {
	t.Run("allowed", func(t *testing.T) {
		var receivedReq analyzeRequest
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Contains(t, r.URL.Path, "/contentsafety/text:analyze")
			assert.Equal(t, "POST", r.Method)
			assert.NotEmpty(t, r.Header.Get("Ocp-Apim-Subscription-Key"))

			json.NewDecoder(r.Body).Decode(&receivedReq)

			resp := analyzeResponse{
				CategoriesAnalysis: []categoryAnalysis{
					{Category: "Hate", Severity: 0},
					{Category: "SelfHarm", Severity: 0},
					{Category: "Sexual", Severity: 0},
					{Category: "Violence", Severity: 0},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		}))
		defer srv.Close()

		g, err := New(WithEndpoint(srv.URL), WithAPIKey("test-key"))
		require.NoError(t, err)

		result, err := g.Validate(context.Background(), guard.GuardInput{
			Content: "Hello world",
			Role:    "input",
		})
		require.NoError(t, err)
		assert.True(t, result.Allowed)
		assert.Equal(t, "azure_content_safety", result.GuardName)
		assert.Equal(t, "Hello world", receivedReq.Text)
	})

	t.Run("blocked", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			resp := analyzeResponse{
				CategoriesAnalysis: []categoryAnalysis{
					{Category: "Hate", Severity: 4},
					{Category: "Violence", Severity: 6},
					{Category: "SelfHarm", Severity: 0},
					{Category: "Sexual", Severity: 0},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		}))
		defer srv.Close()

		g, err := New(WithEndpoint(srv.URL), WithAPIKey("test-key"))
		require.NoError(t, err)

		result, err := g.Validate(context.Background(), guard.GuardInput{
			Content: "violent content",
			Role:    "input",
		})
		require.NoError(t, err)
		assert.False(t, result.Allowed)
		assert.Contains(t, result.Reason, "Hate")
		assert.Contains(t, result.Reason, "Violence")
	})

	t.Run("custom threshold", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			resp := analyzeResponse{
				CategoriesAnalysis: []categoryAnalysis{
					{Category: "Hate", Severity: 2},
					{Category: "Violence", Severity: 3},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		}))
		defer srv.Close()

		g, err := New(WithEndpoint(srv.URL), WithAPIKey("test-key"), WithThreshold(4))
		require.NoError(t, err)

		result, err := g.Validate(context.Background(), guard.GuardInput{Content: "test", Role: "input"})
		require.NoError(t, err)
		assert.True(t, result.Allowed)
	})

	t.Run("server error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error":{"message":"server error"}}`))
		}))
		defer srv.Close()

		g, err := New(WithEndpoint(srv.URL), WithAPIKey("test-key"))
		require.NoError(t, err)

		_, err = g.Validate(context.Background(), guard.GuardInput{Content: "test", Role: "input"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "azuresafety")
	})

	t.Run("context cancellation", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(5 * time.Second)
		}))
		defer srv.Close()

		g, err := New(WithEndpoint(srv.URL), WithAPIKey("test-key"), WithTimeout(30*time.Second))
		require.NoError(t, err)

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err = g.Validate(ctx, guard.GuardInput{Content: "test", Role: "input"})
		assert.Error(t, err)
	})
}

func TestInterfaceCompliance(t *testing.T) {
	var _ guard.Guard = (*Guard)(nil)
}
