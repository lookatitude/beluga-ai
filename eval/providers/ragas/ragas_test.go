package ragas

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
		assert.Equal(t, "ragas_faithfulness", m.Name())
	})

	t.Run("with options", func(t *testing.T) {
		m, err := New(
			WithBaseURL("http://ragas:8080"),
			WithAPIKey("test-key"),
			WithMetricName("answer_relevancy"),
			WithTimeout(5*time.Second),
		)
		require.NoError(t, err)
		assert.Equal(t, "ragas_answer_relevancy", m.Name())
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

			err := json.NewDecoder(r.Body).Decode(&receivedReq)
			require.NoError(t, err)

			resp := evaluateResponse{
				Scores: []scoreResult{
					{MetricName: "faithfulness", Score: 0.85},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		}))
		defer srv.Close()

		m, err := New(
			WithBaseURL(srv.URL),
			WithMetricName("faithfulness"),
		)
		require.NoError(t, err)

		sample := eval.EvalSample{
			Input:          "What is Go?",
			Output:         "Go is a programming language.",
			ExpectedOutput: "Go is a statically typed language.",
			RetrievedDocs: []schema.Document{
				{Content: "Go is a programming language created by Google."},
				{Content: "It was released in 2009."},
			},
		}

		score, err := m.Score(context.Background(), sample)
		require.NoError(t, err)
		assert.InDelta(t, 0.85, score, 0.001)

		assert.Equal(t, "faithfulness", receivedReq.MetricName)
		assert.Len(t, receivedReq.Data, 1)
		assert.Equal(t, "What is Go?", receivedReq.Data[0].Question)
		assert.Len(t, receivedReq.Data[0].Contexts, 2)
	})

	t.Run("score clamping high", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			resp := evaluateResponse{
				Scores: []scoreResult{
					{MetricName: "relevancy", Score: 1.5},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		}))
		defer srv.Close()

		m, err := New(WithBaseURL(srv.URL), WithMetricName("relevancy"))
		require.NoError(t, err)

		score, err := m.Score(context.Background(), eval.EvalSample{Input: "test"})
		require.NoError(t, err)
		assert.Equal(t, 1.0, score)
	})

	t.Run("score clamping low", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			resp := evaluateResponse{
				Scores: []scoreResult{
					{MetricName: "relevancy", Score: -0.5},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		}))
		defer srv.Close()

		m, err := New(WithBaseURL(srv.URL), WithMetricName("relevancy"))
		require.NoError(t, err)

		score, err := m.Score(context.Background(), eval.EvalSample{Input: "test"})
		require.NoError(t, err)
		assert.Equal(t, 0.0, score)
	})

	t.Run("no scores returned", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			resp := evaluateResponse{Scores: []scoreResult{}}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		}))
		defer srv.Close()

		m, err := New(WithBaseURL(srv.URL))
		require.NoError(t, err)

		_, err = m.Score(context.Background(), eval.EvalSample{Input: "test"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no scores returned")
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
		assert.Contains(t, err.Error(), "ragas")
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

	t.Run("with api key", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "Bearer my-key", r.Header.Get("Authorization"))
			resp := evaluateResponse{
				Scores: []scoreResult{{MetricName: "faithfulness", Score: 0.9}},
			}
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
}

func TestInterfaceCompliance(t *testing.T) {
	var _ eval.Metric = (*Metric)(nil)
}
