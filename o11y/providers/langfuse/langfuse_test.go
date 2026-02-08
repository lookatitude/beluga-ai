package langfuse

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
		exp, err := New(
			WithBaseURL("https://example.com"),
			WithPublicKey("pk-test"),
			WithSecretKey("sk-test"),
		)
		require.NoError(t, err)
		assert.NotNil(t, exp)
	})

	t.Run("missing public key", func(t *testing.T) {
		_, err := New(WithSecretKey("sk-test"))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "public key")
	})

	t.Run("missing secret key", func(t *testing.T) {
		_, err := New(WithPublicKey("pk-test"))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "secret key")
	})
}

func TestExportLLMCall(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		var receivedBatch ingestionRequest
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/api/public/ingestion", r.URL.Path)
			assert.Equal(t, "POST", r.Method)
			assert.Contains(t, r.Header.Get("Authorization"), "Basic ")

			err := json.NewDecoder(r.Body).Decode(&receivedBatch)
			require.NoError(t, err)

			resp := ingestionResponse{
				Successes: []struct {
					ID     string `json:"id"`
					Status int    `json:"status"`
				}{
					{ID: "test-id", Status: 200},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		}))
		defer srv.Close()

		exp, err := New(
			WithBaseURL(srv.URL),
			WithPublicKey("pk-test"),
			WithSecretKey("sk-test"),
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

		assert.Len(t, receivedBatch.Batch, 2)
		assert.Equal(t, "trace-create", receivedBatch.Batch[0].Type)
		assert.Equal(t, "generation-create", receivedBatch.Batch[1].Type)
	})

	t.Run("error call", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var batch ingestionRequest
			json.NewDecoder(r.Body).Decode(&batch)

			resp := ingestionResponse{
				Successes: []struct {
					ID     string `json:"id"`
					Status int    `json:"status"`
				}{
					{ID: "test-id", Status: 200},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		}))
		defer srv.Close()

		exp, err := New(
			WithBaseURL(srv.URL),
			WithPublicKey("pk-test"),
			WithSecretKey("sk-test"),
		)
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

		exp, err := New(
			WithBaseURL(srv.URL),
			WithPublicKey("pk-test"),
			WithSecretKey("sk-test"),
		)
		require.NoError(t, err)

		err = exp.ExportLLMCall(context.Background(), o11y.LLMCallData{Model: "gpt-4"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "langfuse")
	})

	t.Run("ingestion errors", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			resp := ingestionResponse{
				Errors: []struct {
					ID      string `json:"id"`
					Status  int    `json:"status"`
					Message string `json:"message"`
				}{
					{ID: "err-1", Status: 400, Message: "invalid event"},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		}))
		defer srv.Close()

		exp, err := New(
			WithBaseURL(srv.URL),
			WithPublicKey("pk-test"),
			WithSecretKey("sk-test"),
		)
		require.NoError(t, err)

		err = exp.ExportLLMCall(context.Background(), o11y.LLMCallData{Model: "gpt-4"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid event")
	})

	t.Run("context cancellation", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(5 * time.Second)
		}))
		defer srv.Close()

		exp, err := New(
			WithBaseURL(srv.URL),
			WithPublicKey("pk-test"),
			WithSecretKey("sk-test"),
			WithTimeout(30*time.Second),
		)
		require.NoError(t, err)

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err = exp.ExportLLMCall(ctx, o11y.LLMCallData{Model: "gpt-4"})
		assert.Error(t, err)
	})
}

func TestFlush(t *testing.T) {
	exp, err := New(
		WithPublicKey("pk-test"),
		WithSecretKey("sk-test"),
	)
	require.NoError(t, err)
	assert.NoError(t, exp.Flush(context.Background()))
}

func TestInterfaceCompliance(t *testing.T) {
	var _ o11y.TraceExporter = (*Exporter)(nil)
}
