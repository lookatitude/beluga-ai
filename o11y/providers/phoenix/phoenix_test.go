package phoenix

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
	t.Run("default config", func(t *testing.T) {
		exp, err := New()
		require.NoError(t, err)
		assert.NotNil(t, exp)
	})

	t.Run("with options", func(t *testing.T) {
		exp, err := New(
			WithBaseURL("http://phoenix:6006"),
			WithAPIKey("test-key"),
			WithTimeout(5*time.Second),
		)
		require.NoError(t, err)
		assert.NotNil(t, exp)
	})
}

func TestExportLLMCall(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		var receivedReq phoenixTraceRequest
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/v1/traces", r.URL.Path)
			assert.Equal(t, "POST", r.Method)

			err := json.NewDecoder(r.Body).Decode(&receivedReq)
			require.NoError(t, err)

			resp := phoenixTraceResponse{
				Data: []struct {
					TraceID string `json:"trace_id"`
				}{
					{TraceID: "test-trace"},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		}))
		defer srv.Close()

		exp, err := New(WithBaseURL(srv.URL))
		require.NoError(t, err)

		data := o11y.LLMCallData{
			Model:        "claude-3",
			Provider:     "anthropic",
			InputTokens:  200,
			OutputTokens: 100,
			Duration:     1 * time.Second,
			Cost:         0.05,
			Messages:     []map[string]any{{"role": "user", "content": "test"}},
			Response:     map[string]any{"content": "response"},
			Metadata:     map[string]any{"trace_id": "t1"},
		}

		err = exp.ExportLLMCall(context.Background(), data)
		require.NoError(t, err)

		require.Len(t, receivedReq.Data, 1)
		span := receivedReq.Data[0]
		assert.Equal(t, "anthropic.claude-3", span.Name)
		assert.Equal(t, "LLM", span.Kind)
		assert.Equal(t, "OK", span.Status.StatusCode)
	})

	t.Run("error status", func(t *testing.T) {
		var receivedReq phoenixTraceRequest
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			json.NewDecoder(r.Body).Decode(&receivedReq)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(phoenixTraceResponse{})
		}))
		defer srv.Close()

		exp, err := New(WithBaseURL(srv.URL))
		require.NoError(t, err)

		data := o11y.LLMCallData{
			Model:    "gpt-4",
			Provider: "openai",
			Error:    "context length exceeded",
		}

		err = exp.ExportLLMCall(context.Background(), data)
		require.NoError(t, err)

		require.Len(t, receivedReq.Data, 1)
		assert.Equal(t, "ERROR", receivedReq.Data[0].Status.StatusCode)
		assert.Equal(t, "context length exceeded", receivedReq.Data[0].Status.Message)
	})

	t.Run("server error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error":{"message":"server error"}}`))
		}))
		defer srv.Close()

		exp, err := New(WithBaseURL(srv.URL))
		require.NoError(t, err)

		err = exp.ExportLLMCall(context.Background(), o11y.LLMCallData{Model: "gpt-4"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "phoenix")
	})

	t.Run("context cancellation", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(5 * time.Second)
		}))
		defer srv.Close()

		exp, err := New(WithBaseURL(srv.URL), WithTimeout(30*time.Second))
		require.NoError(t, err)

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err = exp.ExportLLMCall(ctx, o11y.LLMCallData{Model: "gpt-4"})
		assert.Error(t, err)
	})

	t.Run("with api key", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(phoenixTraceResponse{})
		}))
		defer srv.Close()

		exp, err := New(WithBaseURL(srv.URL), WithAPIKey("test-key"))
		require.NoError(t, err)

		err = exp.ExportLLMCall(context.Background(), o11y.LLMCallData{Model: "gpt-4"})
		require.NoError(t, err)
	})
}

func TestFlush(t *testing.T) {
	exp, err := New()
	require.NoError(t, err)
	assert.NoError(t, exp.Flush(context.Background()))
}

func TestInterfaceCompliance(t *testing.T) {
	var _ o11y.TraceExporter = (*Exporter)(nil)
}
