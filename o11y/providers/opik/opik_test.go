package opik

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
		exp, err := New(WithAPIKey("opik-test"))
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
			WithBaseURL("https://custom.opik.com"),
			WithAPIKey("opik-test"),
			WithWorkspace("my-workspace"),
			WithTimeout(5*time.Second),
		)
		require.NoError(t, err)
		assert.NotNil(t, exp)
		assert.Equal(t, "my-workspace", exp.workspace)
	})
}

func TestExportLLMCall(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		var receivedTrace traceCreate
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/v1/private/traces", r.URL.Path)
			assert.Equal(t, "POST", r.Method)
			assert.Contains(t, r.Header.Get("Authorization"), "Bearer ")
			assert.Equal(t, "test-ws", r.Header.Get("Comet-Workspace"))

			json.NewDecoder(r.Body).Decode(&receivedTrace)

			resp := traceResponse{ID: receivedTrace.ID}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		}))
		defer srv.Close()

		exp, err := New(
			WithBaseURL(srv.URL),
			WithAPIKey("opik-test"),
			WithWorkspace("test-ws"),
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
		}

		err = exp.ExportLLMCall(context.Background(), data)
		require.NoError(t, err)

		assert.Equal(t, "openai/gpt-4", receivedTrace.Name)
		assert.NotEmpty(t, receivedTrace.ID)
	})

	t.Run("server error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error":{"message":"internal error"}}`))
		}))
		defer srv.Close()

		exp, err := New(WithBaseURL(srv.URL), WithAPIKey("opik-test"))
		require.NoError(t, err)

		err = exp.ExportLLMCall(context.Background(), o11y.LLMCallData{Model: "gpt-4"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "opik")
	})

	t.Run("context cancellation", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(5 * time.Second)
		}))
		defer srv.Close()

		exp, err := New(WithBaseURL(srv.URL), WithAPIKey("opik-test"), WithTimeout(30*time.Second))
		require.NoError(t, err)

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err = exp.ExportLLMCall(ctx, o11y.LLMCallData{Model: "gpt-4"})
		assert.Error(t, err)
	})
}

func TestFlush(t *testing.T) {
	exp, err := New(WithAPIKey("opik-test"))
	require.NoError(t, err)
	assert.NoError(t, exp.Flush(context.Background()))
}

func TestInterfaceCompliance(t *testing.T) {
	var _ o11y.TraceExporter = (*Exporter)(nil)
}
