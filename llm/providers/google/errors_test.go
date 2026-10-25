package google

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/lookatitude/beluga-ai/v2/core"
	"github.com/lookatitude/beluga-ai/v2/schema"
	"google.golang.org/genai"
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
		{"value 429", genai.APIError{Code: 429}, core.ErrRateLimit},
		{"pointer 403", &genai.APIError{Code: 403}, core.ErrAuth},
		{"wrapped 503", fmt.Errorf("x: %w", genai.APIError{Code: 503}), core.ErrProviderDown},
		{"value 400", genai.APIError{Code: 400}, core.ErrInvalidInput},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := codeOf(classifyError("google: generate failed", tt.err)); got != tt.want {
				t.Fatalf("code = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGenerate_AuthErrorNotRetryable(t *testing.T) {
	ts, m := newTestModel(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `{"error":{"code":401,"message":"bad key","status":"UNAUTHENTICATED"}}`)
	})
	defer ts.Close()
	msgs := []schema.Message{schema.NewHumanMessage("Hi")}
	_, err := m.Generate(context.Background(), msgs)
	if codeOf(err) != core.ErrAuth || core.IsRetryable(err) {
		t.Fatalf("Generate: want non-retryable auth error, got %v", err)
	}
	var streamErr error
	for _, err := range m.Stream(context.Background(), msgs) {
		if err != nil {
			streamErr = err
		}
	}
	if codeOf(streamErr) != core.ErrAuth {
		t.Fatalf("Stream: want auth error, got %v", streamErr)
	}
}
