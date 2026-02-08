package braintrust

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
	t.Run("success", func(t *testing.T) {
		m, err := New(WithAPIKey("bt-test"))
		require.NoError(t, err)
		assert.NotNil(t, m)
		assert.Equal(t, "braintrust_factuality", m.Name())
	})

	t.Run("with options", func(t *testing.T) {
		m, err := New(
			WithBaseURL("http://braintrust:8080"),
			WithAPIKey("bt-test"),
			WithMetricName("relevance"),
			WithProjectName("my-project"),
			WithTimeout(5*time.Second),
		)
		require.NoError(t, err)
		assert.Equal(t, "braintrust_relevance", m.Name())
	})

	t.Run("missing api key", func(t *testing.T) {
		_, err := New()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "API key")
	})

	t.Run("empty metric name", func(t *testing.T) {
		_, err := New(WithAPIKey("bt-test"), WithMetricName(""))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "metric name")
	})
}

func TestScore(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		var receivedReq scoreRequest
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/v1/score", r.URL.Path)
			assert.Equal(t, "POST", r.Method)
			assert.Contains(t, r.Header.Get("Authorization"), "Bearer ")

			json.NewDecoder(r.Body).Decode(&receivedReq)

			resp := scoreResponse{
				Results: []scoreResult{
					{Name: "factuality", Score: 0.88},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		}))
		defer srv.Close()

		m, err := New(
			WithBaseURL(srv.URL),
			WithAPIKey("bt-test"),
			WithMetricName("factuality"),
			WithProjectName("test-project"),
		)
		require.NoError(t, err)

		sample := eval.EvalSample{
			Input:          "What is Go?",
			Output:         "Go is a programming language.",
			ExpectedOutput: "Go is a statically typed language.",
			RetrievedDocs: []schema.Document{
				{Content: "Go was created by Google."},
			},
		}

		score, err := m.Score(context.Background(), sample)
		require.NoError(t, err)
		assert.InDelta(t, 0.88, score, 0.001)

		assert.Equal(t, "test-project", receivedReq.ProjectName)
		assert.Len(t, receivedReq.Scores, 1)
		assert.Equal(t, "factuality", receivedReq.Scores[0].Name)
	})

	t.Run("no results", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			resp := scoreResponse{Results: nil}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		}))
		defer srv.Close()

		m, err := New(WithBaseURL(srv.URL), WithAPIKey("bt-test"))
		require.NoError(t, err)

		_, err = m.Score(context.Background(), eval.EvalSample{Input: "test"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no results")
	})

	t.Run("score clamping high", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			resp := scoreResponse{Results: []scoreResult{{Name: "factuality", Score: 1.5}}}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		}))
		defer srv.Close()

		m, err := New(WithBaseURL(srv.URL), WithAPIKey("bt-test"))
		require.NoError(t, err)

		score, err := m.Score(context.Background(), eval.EvalSample{Input: "test"})
		require.NoError(t, err)
		assert.Equal(t, 1.0, score)
	})

	t.Run("score clamping low", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			resp := scoreResponse{Results: []scoreResult{{Name: "factuality", Score: -0.3}}}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		}))
		defer srv.Close()

		m, err := New(WithBaseURL(srv.URL), WithAPIKey("bt-test"))
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

		m, err := New(WithBaseURL(srv.URL), WithAPIKey("bt-test"))
		require.NoError(t, err)

		_, err = m.Score(context.Background(), eval.EvalSample{Input: "test"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "braintrust")
	})

	t.Run("context cancellation", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(5 * time.Second)
		}))
		defer srv.Close()

		m, err := New(WithBaseURL(srv.URL), WithAPIKey("bt-test"), WithTimeout(30*time.Second))
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
