package guardrailsai

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
	t.Run("default config", func(t *testing.T) {
		g, err := New()
		require.NoError(t, err)
		assert.NotNil(t, g)
		assert.Equal(t, "guardrails_ai", g.Name())
	})

	t.Run("with options", func(t *testing.T) {
		g, err := New(
			WithBaseURL("http://guardrails:8000"),
			WithAPIKey("test-key"),
			WithGuardName("my-guard"),
			WithTimeout(5*time.Second),
		)
		require.NoError(t, err)
		assert.NotNil(t, g)
	})

	t.Run("empty guard name", func(t *testing.T) {
		_, err := New(WithGuardName(""))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "guard name")
	})
}

func TestValidate(t *testing.T) {
	t.Run("pass", func(t *testing.T) {
		var receivedReq validateRequest
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/guards/my-guard/validate", r.URL.Path)
			assert.Equal(t, "POST", r.Method)

			err := json.NewDecoder(r.Body).Decode(&receivedReq)
			require.NoError(t, err)

			resp := validateResponse{
				Result: "pass",
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		}))
		defer srv.Close()

		g, err := New(WithBaseURL(srv.URL), WithGuardName("my-guard"))
		require.NoError(t, err)

		result, err := g.Validate(context.Background(), guard.GuardInput{
			Content: "Hello, how are you?",
			Role:    "input",
		})
		require.NoError(t, err)
		assert.True(t, result.Allowed)
		assert.Equal(t, "guardrails_ai", result.GuardName)
		assert.Equal(t, "Hello, how are you?", receivedReq.Prompt)
	})

	t.Run("fail", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			resp := validateResponse{
				Result: "fail",
				Failed: []validationItem{
					{
						ValidatorName: "toxicity",
						Result:        "fail",
						Message:       "toxic content detected",
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		}))
		defer srv.Close()

		g, err := New(WithBaseURL(srv.URL))
		require.NoError(t, err)

		result, err := g.Validate(context.Background(), guard.GuardInput{
			Content: "toxic content",
			Role:    "output",
		})
		require.NoError(t, err)
		assert.False(t, result.Allowed)
		assert.Equal(t, "toxic content detected", result.Reason)
	})

	t.Run("fail without message", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			resp := validateResponse{
				Result: "fail",
				Failed: []validationItem{
					{
						ValidatorName: "pii_detector",
						Result:        "fail",
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		}))
		defer srv.Close()

		g, err := New(WithBaseURL(srv.URL))
		require.NoError(t, err)

		result, err := g.Validate(context.Background(), guard.GuardInput{
			Content: "my ssn is 123-45-6789",
			Role:    "input",
		})
		require.NoError(t, err)
		assert.False(t, result.Allowed)
		assert.Contains(t, result.Reason, "pii_detector")
	})

	t.Run("fail no validators", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			resp := validateResponse{Result: "fail"}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		}))
		defer srv.Close()

		g, err := New(WithBaseURL(srv.URL))
		require.NoError(t, err)

		result, err := g.Validate(context.Background(), guard.GuardInput{
			Content: "test",
			Role:    "input",
		})
		require.NoError(t, err)
		assert.False(t, result.Allowed)
		assert.Equal(t, "validation failed", result.Reason)
	})

	t.Run("modified output", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			resp := validateResponse{
				Result:          "pass",
				ValidatedOutput: "redacted content",
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		}))
		defer srv.Close()

		g, err := New(WithBaseURL(srv.URL))
		require.NoError(t, err)

		result, err := g.Validate(context.Background(), guard.GuardInput{
			Content: "original with PII",
			Role:    "output",
		})
		require.NoError(t, err)
		assert.True(t, result.Allowed)
		assert.Equal(t, "redacted content", result.Modified)
	})

	t.Run("output role uses llmOutput field", func(t *testing.T) {
		var receivedReq validateRequest
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			json.NewDecoder(r.Body).Decode(&receivedReq)
			resp := validateResponse{Result: "pass"}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		}))
		defer srv.Close()

		g, err := New(WithBaseURL(srv.URL))
		require.NoError(t, err)

		_, err = g.Validate(context.Background(), guard.GuardInput{
			Content: "model response",
			Role:    "output",
		})
		require.NoError(t, err)
		assert.Equal(t, "model response", receivedReq.LLMOutput)
		assert.Empty(t, receivedReq.Prompt)
	})

	t.Run("server error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error":{"message":"server error"}}`))
		}))
		defer srv.Close()

		g, err := New(WithBaseURL(srv.URL))
		require.NoError(t, err)

		_, err = g.Validate(context.Background(), guard.GuardInput{
			Content: "test",
			Role:    "input",
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "guardrailsai")
	})

	t.Run("context cancellation", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(5 * time.Second)
		}))
		defer srv.Close()

		g, err := New(WithBaseURL(srv.URL), WithTimeout(30*time.Second))
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
