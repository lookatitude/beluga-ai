package cohere

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	coherego "github.com/cohere-ai/cohere-go/v2"
	coherecore "github.com/cohere-ai/cohere-go/v2/core"
	"github.com/lookatitude/beluga-ai/v2/config"
	"github.com/lookatitude/beluga-ai/v2/core"
	"github.com/lookatitude/beluga-ai/v2/schema"
)

func codeOf(err error) core.ErrorCode {
	var ce *core.Error
	if errors.As(err, &ce) {
		return ce.Code
	}
	return ""
}

func TestClassifyError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want core.ErrorCode
	}{
		{"api 429", coherecore.NewAPIError(429, nil, errors.New("slow down")), core.ErrRateLimit},
		{"typed 400", &coherego.BadRequestError{APIError: coherecore.NewAPIError(400, nil, errors.New("bad"))}, core.ErrInvalidInput},
		{"typed 401", &coherego.UnauthorizedError{APIError: coherecore.NewAPIError(401, nil, errors.New("key"))}, core.ErrAuth},
		{"api 503", coherecore.NewAPIError(503, nil, errors.New("down")), core.ErrProviderDown},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := codeOf(classifyError("cohere: generate failed", tt.err)); got != tt.want {
				t.Fatalf("code = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGenerateAndStream_AuthError(t *testing.T) {
	var hits atomic.Int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `{"message":"invalid api token"}`)
	}))
	defer ts.Close()
	m, err := New(config.ProviderConfig{Model: "command-a-03-2025", APIKey: "test", BaseURL: ts.URL})
	if err != nil {
		t.Fatal(err)
	}
	msgs := []schema.Message{schema.NewHumanMessage("Hi")}
	_, err = m.Generate(context.Background(), msgs)
	if codeOf(err) != core.ErrAuth || core.IsRetryable(err) {
		t.Fatalf("Generate: want non-retryable auth, got %v", err)
	}

	before := hits.Load()
	seq := m.Stream(context.Background(), msgs)
	if hits.Load() != before {
		t.Fatal("Stream sent a request before iteration")
	}
	var streamErr error
	for _, err := range seq {
		if err != nil {
			streamErr = err
		}
	}
	if codeOf(streamErr) != core.ErrAuth {
		t.Fatalf("Stream: want auth, got %v", streamErr)
	}
}
