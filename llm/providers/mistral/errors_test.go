package mistral

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"testing"
	"time"

	"github.com/gage-technologies/mistral-go"
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
		msg  string
		want core.ErrorCode
	}{
		{"(HTTP Error 429) too many", core.ErrRateLimit},
		{"(HTTP Error 401) unauthorized", core.ErrAuth},
		{"(HTTP Error 400) bad", core.ErrInvalidInput},
		{"(HTTP Error 503) down", core.ErrProviderDown},
		{"(HTTP Error 500) oops", core.ErrProviderDown},
	}
	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			if got := codeOf(classifyError("mistral: generate failed", fmt.Errorf("%s", tt.msg))); got != tt.want {
				t.Fatalf("code = %q, want %q", got, tt.want)
			}
		})
	}
	if core.IsRetryable(classifyError("mistral", errors.New("invalid response type"))) {
		t.Fatal("unknown error must not be retryable")
	}
	if classifyError("mistral", nil) != nil {
		t.Fatal("nil must stay nil")
	}
}

// TestConsumeStreamChannel_EarlyBreakDrains checks that stopping early does not
// leave mistral-go's sender goroutine blocked on the unbuffered channel.
func TestConsumeStreamChannel_EarlyBreakDrains(t *testing.T) {
	before := runtime.NumGoroutine()
	for range 5 {
		ch := make(chan mistral.ChatCompletionStreamResponse)
		senderDone := make(chan struct{})
		go func() {
			defer close(senderDone)
			defer close(ch)
			for range 3 {
				ch <- mistral.ChatCompletionStreamResponse{}
			}
		}()
		consumeStreamChannel(context.Background(), ch, "m", func(schema.StreamChunk, error) bool { return false })
		select {
		case <-senderDone:
		case <-time.After(2 * time.Second):
			t.Fatal("sender goroutine still blocked after consumer stopped")
		}
	}
	time.Sleep(50 * time.Millisecond)
	if after := runtime.NumGoroutine(); after > before+2 {
		t.Fatalf("goroutines grew: before=%d after=%d", before, after)
	}
}

func TestConsumeStreamChannel_ErrorClassified(t *testing.T) {
	ch := make(chan mistral.ChatCompletionStreamResponse, 1)
	ch <- mistral.ChatCompletionStreamResponse{Error: errors.New("(HTTP Error 401) nope")}
	close(ch)
	var got error
	consumeStreamChannel(context.Background(), ch, "m", func(_ schema.StreamChunk, err error) bool {
		got = err
		return true
	})
	if codeOf(got) != core.ErrAuth {
		t.Fatalf("want auth error, got %v", got)
	}
}
