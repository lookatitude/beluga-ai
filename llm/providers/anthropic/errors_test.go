package anthropic

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync/atomic"
	"testing"

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

// TestErrorClassification is the FWC-04 regression: Anthropic failures must be
// classified by HTTP status instead of always being ErrProviderDown.
func TestErrorClassification(t *testing.T) {
	tests := []struct {
		status    int
		want      core.ErrorCode
		retryable bool
	}{
		{http.StatusUnauthorized, core.ErrAuth, false},
		{http.StatusForbidden, core.ErrAuth, false},
		{http.StatusBadRequest, core.ErrInvalidInput, false},
		{http.StatusTooManyRequests, core.ErrRateLimit, true},
		{http.StatusInternalServerError, core.ErrProviderDown, true},
		{529, core.ErrProviderDown, true},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprint(tt.status), func(t *testing.T) {
			ts, m := newTestModel(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.status)
				fmt.Fprint(w, `{"type":"error","error":{"type":"x","message":"nope"}}`)
			})
			defer ts.Close()
			msgs := []schema.Message{schema.NewHumanMessage("Hi")}

			_, err := m.Generate(context.Background(), msgs)
			if codeOf(err) != tt.want || core.IsRetryable(err) != tt.retryable {
				t.Fatalf("Generate: code=%q retryable=%v, want %q/%v (err=%v)", codeOf(err), core.IsRetryable(err), tt.want, tt.retryable, err)
			}
			var streamErr error
			for _, err := range m.Stream(context.Background(), msgs) {
				if err != nil {
					streamErr = err
				}
			}
			if codeOf(streamErr) != tt.want {
				t.Fatalf("Stream: code=%q, want %q (err=%v)", codeOf(streamErr), tt.want, streamErr)
			}
		})
	}
}

func TestStream_IsLazy(t *testing.T) {
	var hits atomic.Int32
	ts, m := newTestModel(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	})
	defer ts.Close()
	seq := m.Stream(context.Background(), []schema.Message{schema.NewHumanMessage("Hi")})
	if hits.Load() != 0 {
		t.Fatal("request sent before iteration")
	}
	for range seq { //nolint:revive // drain
	}
	if hits.Load() != 1 {
		t.Fatalf("hits = %d, want 1", hits.Load())
	}
}
