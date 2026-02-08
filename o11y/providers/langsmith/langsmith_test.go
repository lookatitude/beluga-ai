package langsmith

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/o11y"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		exp, err := New(WithAPIKey("lsv2-test"))
		require.NoError(t, err)
		assert.NotNil(t, exp)
	})

	t.Run("missing api key", func(t *testing.T) {
		_, err := New()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "API key")
	})

	t.Run("with options", func(t *testing.T) {
		exp, err := New(
			WithBaseURL("https://custom.langsmith.com"),
			WithAPIKey("lsv2-test"),
			WithProject("my-project"),
			WithTimeout(5*time.Second),
		)
		require.NoError(t, err)
		assert.NotNil(t, exp)
		assert.Equal(t, "my-project", exp.project)
	})
}

func TestExportLLMCall(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		var receivedBatch batchRequest
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/runs/batch", r.URL.Path)
			assert.Equal(t, "POST", r.Method)
			assert.Equal(t, "lsv2-test", r.Header.Get("x-api-key"))

			json.NewDecoder(r.Body).Decode(&receivedBatch)

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(batchResponse{})
		}))
		defer srv.Close()

		exp, err := New(
			WithBaseURL(srv.URL),
			WithAPIKey("lsv2-test"),
			WithProject("test-project"),
		)
		require.NoError(t, err)

		data := o11y.LLMCallData{
			Model:        "gpt-4",
			Provider:     "openai",
			InputTokens:  100,
			OutputTokens: 50,
			Duration:     500 * time.Millisecond,
			Cost:         0.01,
			Messages:     []map[string]any{{"role": "user", "content": "hello"}},
			Response:     map[string]any{"content": "hi"},
			Metadata:     map[string]any{"session_id": "s1"},
		}

		err = exp.ExportLLMCall(context.Background(), data)
		require.NoError(t, err)

		assert.Len(t, receivedBatch.Post, 1)
		assert.Equal(t, "llm", receivedBatch.Post[0].RunType)
		assert.Equal(t, "openai/gpt-4", receivedBatch.Post[0].Name)
		assert.Equal(t, "test-project", receivedBatch.Post[0].SessionName)
	})

	t.Run("error call", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var batch batchRequest
			json.NewDecoder(r.Body).Decode(&batch)

			assert.Equal(t, "rate limited", batch.Post[0].Error)

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(batchResponse{})
		}))
		defer srv.Close()

		exp, err := New(WithBaseURL(srv.URL), WithAPIKey("lsv2-test"))
		require.NoError(t, err)

		data := o11y.LLMCallData{
			Model:    "gpt-4",
			Provider: "openai",
			Error:    "rate limited",
		}

		err = exp.ExportLLMCall(context.Background(), data)
		require.NoError(t, err)
	})

	t.Run("server error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error":{"message":"internal error"}}`))
		}))
		defer srv.Close()

		exp, err := New(WithBaseURL(srv.URL), WithAPIKey("lsv2-test"))
		require.NoError(t, err)

		err = exp.ExportLLMCall(context.Background(), o11y.LLMCallData{Model: "gpt-4"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "langsmith")
	})

	t.Run("model only name", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var batch batchRequest
			json.NewDecoder(r.Body).Decode(&batch)

			assert.Equal(t, "gpt-4", batch.Post[0].Name)

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(batchResponse{})
		}))
		defer srv.Close()

		exp, err := New(WithBaseURL(srv.URL), WithAPIKey("lsv2-test"))
		require.NoError(t, err)

		err = exp.ExportLLMCall(context.Background(), o11y.LLMCallData{Model: "gpt-4"})
		require.NoError(t, err)
	})

	t.Run("context cancellation", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(5 * time.Second)
		}))
		defer srv.Close()

		exp, err := New(WithBaseURL(srv.URL), WithAPIKey("lsv2-test"), WithTimeout(30*time.Second))
		require.NoError(t, err)

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err = exp.ExportLLMCall(ctx, o11y.LLMCallData{Model: "gpt-4"})
		assert.Error(t, err)
	})
}

func TestFlush(t *testing.T) {
	exp, err := New(WithAPIKey("lsv2-test"))
	require.NoError(t, err)
	assert.NoError(t, exp.Flush(context.Background()))
}

func TestInterfaceCompliance(t *testing.T) {
	var _ o11y.TraceExporter = (*Exporter)(nil)
}
