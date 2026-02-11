package llm

import (
	"context"
	"errors"
	"iter"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/core"
	"github.com/lookatitude/beluga-ai/schema"
)

func TestWithProviderLimits_Generate_PassesThrough(t *testing.T) {
	base := &stubModel{id: "base"}
	mw := WithProviderLimits(ProviderLimits{})
	wrapped := mw(base)

	resp, err := wrapped.Generate(context.Background(), []schema.Message{schema.NewHumanMessage("hi")})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
}

func TestWithProviderLimits_Generate_ConcurrencyLimit(t *testing.T) {
	base := &stubModel{id: "base"}
	mw := WithProviderLimits(ProviderLimits{MaxConcurrent: 1})
	wrapped := mw(base)

	resp, err := wrapped.Generate(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
}

func TestWithProviderLimits_Generate_RPMExceeded(t *testing.T) {
	base := &stubModel{id: "base"}
	mw := WithProviderLimits(ProviderLimits{RPM: 1})
	wrapped := mw(base)

	// First call should succeed.
	_, err := wrapped.Generate(context.Background(), nil)
	if err != nil {
		t.Fatalf("first call error: %v", err)
	}

	// Second call should exceed the RPM limit.
	_, err = wrapped.Generate(context.Background(), nil)
	if err == nil {
		t.Fatal("expected rate limit error on second call")
	}

	var coreErr *core.Error
	if !errors.As(err, &coreErr) {
		t.Fatalf("expected core.Error, got %T", err)
	}
	if coreErr.Code != core.ErrRateLimit {
		t.Errorf("expected error code %q, got %q", core.ErrRateLimit, coreErr.Code)
	}
}

func TestWithProviderLimits_Generate_RPMWithCooldown(t *testing.T) {
	base := &stubModel{id: "base"}
	mw := WithProviderLimits(ProviderLimits{
		RPM:             1,
		CooldownOnRetry: 50 * time.Millisecond,
	})
	wrapped := mw(base)

	// First call uses the one RPM slot.
	_, err := wrapped.Generate(context.Background(), nil)
	if err != nil {
		t.Fatalf("first call error: %v", err)
	}

	// Second call should hit cooldown; with a 50ms cooldown the sliding window
	// won't have cleared (1-minute window), so it should still fail after cooldown.
	_, err = wrapped.Generate(context.Background(), nil)
	if err == nil {
		t.Fatal("expected rate limit error after cooldown")
	}
}

func TestWithProviderLimits_Generate_RPMCooldownCancelled(t *testing.T) {
	base := &stubModel{id: "base"}
	mw := WithProviderLimits(ProviderLimits{
		RPM:             1,
		CooldownOnRetry: 5 * time.Second,
	})
	wrapped := mw(base)

	// Use the RPM slot.
	_, _ = wrapped.Generate(context.Background(), nil)

	// Cancel context during cooldown wait.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := wrapped.Generate(ctx, nil)
	if err == nil {
		t.Fatal("expected context cancelled error")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestWithProviderLimits_Generate_ConcurrencyContextCancelled(t *testing.T) {
	// Create a model with 1 concurrency slot.
	base := &stubModel{
		id: "slow",
		generateFn: func(ctx context.Context, msgs []schema.Message, opts ...GenerateOption) (*schema.AIMessage, error) {
			// Block until context cancelled.
			<-ctx.Done()
			return nil, ctx.Err()
		},
	}
	mw := WithProviderLimits(ProviderLimits{MaxConcurrent: 1})
	wrapped := mw(base)

	// Start first call to occupy the slot.
	ctx1, cancel1 := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		_, _ = wrapped.Generate(ctx1, nil)
		close(done)
	}()

	// Give goroutine time to acquire the semaphore.
	time.Sleep(20 * time.Millisecond)

	// Second call with already-cancelled context should fail.
	ctx2, cancel2 := context.WithCancel(context.Background())
	cancel2()
	_, err := wrapped.Generate(ctx2, nil)
	if err == nil {
		t.Fatal("expected error when context cancelled while waiting for concurrency slot")
	}

	cancel1()
	<-done
}

func TestWithProviderLimits_Stream_PassesThrough(t *testing.T) {
	base := &stubModel{
		id: "base",
		streamFn: func(ctx context.Context, msgs []schema.Message, opts ...GenerateOption) iter.Seq2[schema.StreamChunk, error] {
			return func(yield func(schema.StreamChunk, error) bool) {
				yield(schema.StreamChunk{Delta: "chunk1"}, nil)
				yield(schema.StreamChunk{Delta: "chunk2"}, nil)
			}
		},
	}
	mw := WithProviderLimits(ProviderLimits{})
	wrapped := mw(base)

	var deltas []string
	for chunk, err := range wrapped.Stream(context.Background(), nil) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		deltas = append(deltas, chunk.Delta)
	}
	if len(deltas) != 2 || deltas[0] != "chunk1" || deltas[1] != "chunk2" {
		t.Errorf("unexpected deltas: %v", deltas)
	}
}

func TestWithProviderLimits_Stream_RPMExceeded(t *testing.T) {
	base := &stubModel{id: "base"}
	mw := WithProviderLimits(ProviderLimits{RPM: 1})
	wrapped := mw(base)

	// Use up the RPM slot.
	_, _ = wrapped.Generate(context.Background(), nil)

	// Stream should fail with rate limit.
	var gotErr error
	for _, err := range wrapped.Stream(context.Background(), nil) {
		if err != nil {
			gotErr = err
			break
		}
	}
	if gotErr == nil {
		t.Fatal("expected rate limit error on stream")
	}

	var coreErr *core.Error
	if !errors.As(gotErr, &coreErr) {
		t.Fatalf("expected core.Error, got %T: %v", gotErr, gotErr)
	}
	if coreErr.Code != core.ErrRateLimit {
		t.Errorf("expected error code %q, got %q", core.ErrRateLimit, coreErr.Code)
	}
}

func TestWithProviderLimits_Stream_ReleasesOnCompletion(t *testing.T) {
	base := &stubModel{id: "base"}
	mw := WithProviderLimits(ProviderLimits{MaxConcurrent: 1})
	wrapped := mw(base)

	// First stream — consume all chunks.
	for _, err := range wrapped.Stream(context.Background(), nil) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	// Second stream should succeed (slot released).
	for _, err := range wrapped.Stream(context.Background(), nil) {
		if err != nil {
			t.Fatalf("unexpected error on second stream: %v", err)
		}
	}
}

func TestWithProviderLimits_BindTools(t *testing.T) {
	base := &stubModel{id: "base"}
	mw := WithProviderLimits(ProviderLimits{MaxConcurrent: 2})
	wrapped := mw(base)

	bound := wrapped.BindTools([]schema.ToolDefinition{{Name: "search"}})
	if bound == nil {
		t.Fatal("BindTools returned nil")
	}
	if bound.ModelID() != "base" {
		t.Errorf("ModelID = %q, want %q", bound.ModelID(), "base")
	}
}

func TestWithProviderLimits_ModelID(t *testing.T) {
	base := &stubModel{id: "test-model"}
	mw := WithProviderLimits(ProviderLimits{})
	wrapped := mw(base)

	if wrapped.ModelID() != "test-model" {
		t.Errorf("ModelID = %q, want %q", wrapped.ModelID(), "test-model")
	}
}

func TestSlidingWindow_AllowsUpToMax(t *testing.T) {
	w := &slidingWindow{
		maxCount: 3,
		window:   time.Minute,
	}

	for i := 0; i < 3; i++ {
		if !w.allow() {
			t.Fatalf("allow() returned false on call %d, expected true", i+1)
		}
	}

	if w.allow() {
		t.Fatal("allow() returned true when max count reached")
	}
}

func TestSlidingWindow_ExpiredEntriesCleared(t *testing.T) {
	w := &slidingWindow{
		maxCount: 1,
		window:   10 * time.Millisecond,
	}

	if !w.allow() {
		t.Fatal("first call should be allowed")
	}

	if w.allow() {
		t.Fatal("second call should be denied")
	}

	// Wait for entries to expire.
	time.Sleep(15 * time.Millisecond)

	if !w.allow() {
		t.Fatal("call after window expiry should be allowed")
	}
}

func TestProviderLimits_ZeroValues(t *testing.T) {
	limits := ProviderLimits{}
	if limits.RPM != 0 {
		t.Error("expected zero RPM")
	}
	if limits.TPM != 0 {
		t.Error("expected zero TPM")
	}
	if limits.MaxConcurrent != 0 {
		t.Error("expected zero MaxConcurrent")
	}
	if limits.CooldownOnRetry != 0 {
		t.Error("expected zero CooldownOnRetry")
	}
}

// TestWithProviderLimits_Stream_Error tests rateLimitedModel.Stream error path.
func TestWithProviderLimits_Stream_Error(t *testing.T) {
	base := &stubModel{
		id: "base",
		streamFn: func(ctx context.Context, msgs []schema.Message, opts ...GenerateOption) iter.Seq2[schema.StreamChunk, error] {
			return func(yield func(schema.StreamChunk, error) bool) {
				yield(schema.StreamChunk{}, errors.New("stream error"))
			}
		},
	}
	mw := WithProviderLimits(ProviderLimits{MaxConcurrent: 2})
	wrapped := mw(base)

	var gotErr error
	for _, err := range wrapped.Stream(context.Background(), nil) {
		if err != nil {
			gotErr = err
			break
		}
	}
	if gotErr == nil {
		t.Fatal("expected error from stream")
	}
	if gotErr.Error() != "stream error" {
		t.Errorf("unexpected error: %v", gotErr)
	}
}
