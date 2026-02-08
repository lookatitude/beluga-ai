package deepeval

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/eval"
	"github.com/lookatitude/beluga-ai/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Run("default config", func(t *testing.T) {
		m, err := New()
		require.NoError(t, err)
		assert.NotNil(t, m)
		assert.Equal(t, "deepeval_faithfulness", m.Name())
	})

	t.Run("with options", func(t *testing.T) {
		m, err := New(
			WithBaseURL("http://deepeval:8080"),
			WithAPIKey("test-key"),
			WithMetricName("hallucination"),
			WithTimeout(5*time.Second),
		)
		require.NoError(t, err)
		assert.Equal(t, "deepeval_hallucination", m.Name())
	})

	t.Run("empty metric name", func(t *testing.T) {
		_, err := New(WithMetricName(""))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "metric name")
	})
}

func TestScore(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		var receivedReq evaluateRequest
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/api/v1/evaluate", r.URL.Path)
			assert.Equal(t, "POST", r.Method)

			json.NewDecoder(r.Body).Decode(&receivedReq)

			resp := evaluateResponse{
				Score:   0.92,
				Success: true,
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		}))
		defer srv.Close()

		m, err := New(WithBaseURL(srv.URL), WithMetricName("faithfulness"))
		require.NoError(t, err)

		sample := eval.EvalSample{
			Input:          "What is Go?",
			Output:         "Go is a programming language.",
			ExpectedOutput: "Go is a statically typed language.",
			RetrievedDocs: []schema.Document{
				{Content: "Go is a language created by Google."},
			},
		}

		score, err := m.Score(context.Background(), sample)
		require.NoError(t, err)
		assert.InDelta(t, 0.92, score, 0.001)

		assert.Equal(t, "faithfulness", receivedReq.Metric)
		assert.Equal(t, "What is Go?", receivedReq.Input)
		assert.Equal(t, "Go is a programming language.", receivedReq.ActualOutput)
		assert.Len(t, receivedReq.Context, 1)
	})

	t.Run("evaluation failed", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			resp := evaluateResponse{
				Score:   0,
				Success: false,
				Reason:  "insufficient context",
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		}))
		defer srv.Close()

		m, err := New(WithBaseURL(srv.URL))
		require.NoError(t, err)

		_, err = m.Score(context.Background(), eval.EvalSample{Input: "test"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "insufficient context")
	})

	t.Run("score clamping high", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			resp := evaluateResponse{Score: 1.5, Success: true}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		}))
		defer srv.Close()

		m, err := New(WithBaseURL(srv.URL))
		require.NoError(t, err)

		score, err := m.Score(context.Background(), eval.EvalSample{Input: "test"})
		require.NoError(t, err)
		assert.Equal(t, 1.0, score)
	})

	t.Run("score clamping low", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			resp := evaluateResponse{Score: -0.5, Success: true}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		}))
		defer srv.Close()

		m, err := New(WithBaseURL(srv.URL))
		require.NoError(t, err)

		score, err := m.Score(context.Background(), eval.EvalSample{Input: "test"})
		require.NoError(t, err)
		assert.Equal(t, 0.0, score)
	})

	t.Run("server error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error":{"message":"server error"}}`))
		}))
		defer srv.Close()

		m, err := New(WithBaseURL(srv.URL))
		require.NoError(t, err)

		_, err = m.Score(context.Background(), eval.EvalSample{Input: "test"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "deepeval")
	})

	t.Run("with api key", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "Bearer my-key", r.Header.Get("Authorization"))
			resp := evaluateResponse{Score: 0.9, Success: true}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		}))
		defer srv.Close()

		m, err := New(WithBaseURL(srv.URL), WithAPIKey("my-key"))
		require.NoError(t, err)

		score, err := m.Score(context.Background(), eval.EvalSample{Input: "test"})
		require.NoError(t, err)
		assert.InDelta(t, 0.9, score, 0.001)
	})

	t.Run("context cancellation", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(5 * time.Second)
		}))
		defer srv.Close()

		m, err := New(WithBaseURL(srv.URL), WithTimeout(30*time.Second))
		require.NoError(t, err)

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err = m.Score(ctx, eval.EvalSample{Input: "test"})
		assert.Error(t, err)
	})
}

func TestInterfaceCompliance(t *testing.T) {
	var _ eval.Metric = (*Metric)(nil)
}
