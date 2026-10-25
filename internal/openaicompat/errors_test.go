package openaicompat

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/lookatitude/beluga-ai/v2/config"
	"github.com/lookatitude/beluga-ai/v2/core"
	"github.com/lookatitude/beluga-ai/v2/schema"
	"github.com/openai/openai-go/option"
)

func codeOf(err error) core.ErrorCode {
	var ce *core.Error
	if errors.As(err, &ce) {
		return ce.Code
	}
	return ""
}

// TestErrorClassification is the FWC-04 regression: HTTP failures from
// OpenAI-compatible backends must carry core error codes so that
// IsRetryable-driven fallback and retry behave correctly.
func TestErrorClassification(t *testing.T) {
	tests := []struct {
		status    int
		want      core.ErrorCode
		retryable bool
	}{
		{http.StatusTooManyRequests, core.ErrRateLimit, true},
		{http.StatusServiceUnavailable, core.ErrProviderDown, true},
		{http.StatusInternalServerError, core.ErrProviderDown, true},
		{http.StatusUnauthorized, core.ErrAuth, false},
		{http.StatusForbidden, core.ErrAuth, false},
		{http.StatusBadRequest, core.ErrInvalidInput, false},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprint(tt.status), func(t *testing.T) {
			ts, m := newTestServer(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.status)
				fmt.Fprint(w, `{"error":{"message":"nope"}}`)
			})
			defer ts.Close()

			_, err := m.Generate(context.Background(), []schema.Message{schema.NewHumanMessage("Hi")})
			if got := codeOf(err); got != tt.want {
				t.Fatalf("Generate code = %q, want %q (err=%v)", got, tt.want, err)
			}
			if core.IsRetryable(err) != tt.retryable {
				t.Fatalf("Generate IsRetryable = %v, want %v", core.IsRetryable(err), tt.retryable)
			}

			var streamErr error
			for _, err := range m.Stream(context.Background(), []schema.Message{schema.NewHumanMessage("Hi")}) {
				if err != nil {
					streamErr = err
					break
				}
			}
			if got := codeOf(streamErr); got != tt.want {
				t.Fatalf("Stream code = %q, want %q (err=%v)", got, tt.want, streamErr)
			}
		})
	}
}

func TestGenerate_NetworkErrorIsProviderDown(t *testing.T) {
	m, err := NewWithOptions(config.ProviderConfig{Model: "m", APIKey: "k", BaseURL: "http://127.0.0.1:1"}, option.WithMaxRetries(0))
	if err != nil {
		t.Fatal(err)
	}
	_, err = m.Generate(context.Background(), []schema.Message{schema.NewHumanMessage("Hi")})
	if codeOf(err) != core.ErrProviderDown {
		t.Fatalf("code = %q, want provider_unavailable (err=%v)", codeOf(err), err)
	}
}

func TestGenerate_CanceledIsNotRetryable(t *testing.T) {
	ts, m := newTestServer(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	defer ts.Close()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := m.Generate(ctx, []schema.Message{schema.NewHumanMessage("Hi")})
	if err == nil || core.IsRetryable(err) || !errors.Is(err, context.Canceled) {
		t.Fatalf("want non-retryable context.Canceled, got %v", err)
	}
}

// TestStream_IsLazy checks that no request is sent until the stream is ranged.
func TestStream_IsLazy(t *testing.T) {
	var hits atomic.Int32
	ts, m := newTestServer(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	})
	defer ts.Close()

	seq := m.Stream(context.Background(), []schema.Message{schema.NewHumanMessage("Hi")})
	if hits.Load() != 0 {
		t.Fatalf("request sent before iteration: hits=%d", hits.Load())
	}
	for range seq {
		break
	}
	if hits.Load() != 1 {
		t.Fatalf("hits after iteration = %d, want 1", hits.Load())
	}
}
