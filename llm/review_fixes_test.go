package llm

import (
	"context"
	"errors"
	"iter"
	"sync"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/v2/core"
	"github.com/lookatitude/beluga-ai/v2/schema"
)

// --- FWC-06: WithProviderLimits concurrency slot handling for Stream ---

func streamOf(chunks ...string) func(context.Context, []schema.Message, ...GenerateOption) iter.Seq2[schema.StreamChunk, error] {
	return func(context.Context, []schema.Message, ...GenerateOption) iter.Seq2[schema.StreamChunk, error] {
		return func(yield func(schema.StreamChunk, error) bool) {
			for _, c := range chunks {
				if !yield(schema.StreamChunk{Delta: c}, nil) {
					return
				}
			}
		}
	}
}

func generateWithin(t *testing.T, m ChatModel, d time.Duration) error {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), d)
	defer cancel()
	_, err := m.Generate(ctx, nil)
	return err
}

func TestWithProviderLimits_UnrangedStreamHoldsNoSlot(t *testing.T) {
	m := ApplyMiddleware(&stubModel{id: "b", streamFn: streamOf("a")}, WithProviderLimits(ProviderLimits{MaxConcurrent: 1}))
	for range 3 {
		_ = m.Stream(context.Background(), nil) // never ranged
	}
	if err := generateWithin(t, m, 200*time.Millisecond); err != nil {
		t.Fatalf("Generate blocked by un-iterated streams: %v", err)
	}
}

func TestWithProviderLimits_StreamRangedTwice(t *testing.T) {
	m := ApplyMiddleware(&stubModel{id: "b", streamFn: streamOf("a", "b")}, WithProviderLimits(ProviderLimits{MaxConcurrent: 1}))
	seq := m.Stream(context.Background(), nil)
	done := make(chan struct{})
	go func() {
		defer close(done)
		for range 2 {
			for _, err := range seq {
				if err != nil {
					t.Errorf("unexpected err: %v", err)
				}
			}
		}
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("second range of the same Seq deadlocked")
	}
	if err := generateWithin(t, m, 200*time.Millisecond); err != nil {
		t.Fatalf("slot leaked after double range: %v", err)
	}
}

func TestWithProviderLimits_StreamEarlyBreakReleases(t *testing.T) {
	m := ApplyMiddleware(&stubModel{id: "b", streamFn: streamOf("a", "b", "c")}, WithProviderLimits(ProviderLimits{MaxConcurrent: 1}))
	for range 3 {
		for range m.Stream(context.Background(), nil) {
			break
		}
	}
	if err := generateWithin(t, m, 200*time.Millisecond); err != nil {
		t.Fatalf("slot leaked after early break: %v", err)
	}
}

func TestWithProviderLimits_StreamHoldsSlotWhileIterating(t *testing.T) {
	m := ApplyMiddleware(&stubModel{id: "b", streamFn: streamOf("a", "b")}, WithProviderLimits(ProviderLimits{MaxConcurrent: 1}))
	for range m.Stream(context.Background(), nil) {
		// Mid-iteration the single slot is taken.
		if err := generateWithin(t, m, 50*time.Millisecond); !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("expected Generate to wait for the slot, got %v", err)
		}
		break
	}
}

func TestWithProviderLimits_StreamAcquireErrorIsYielded(t *testing.T) {
	m := ApplyMiddleware(&stubModel{id: "b", streamFn: streamOf("a")}, WithProviderLimits(ProviderLimits{RPM: 1}))
	for range m.Stream(context.Background(), nil) { //nolint:revive // drain
	}
	var got error
	for _, err := range m.Stream(context.Background(), nil) {
		got = err
	}
	if !core.IsRetryable(got) {
		t.Fatalf("expected rate-limit error on second stream, got %v", got)
	}
}

func TestWithProviderLimits_ConcurrentStreamsRace(t *testing.T) {
	m := ApplyMiddleware(&stubModel{id: "b", streamFn: streamOf("a", "b")}, WithProviderLimits(ProviderLimits{MaxConcurrent: 2}))
	var wg sync.WaitGroup
	for range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range m.Stream(context.Background(), nil) { //nolint:revive // drain
			}
		}()
	}
	wg.Wait()
	if err := generateWithin(t, m, 200*time.Millisecond); err != nil {
		t.Fatalf("slots leaked: %v", err)
	}
}

// --- FWC-07: TPM is enforced ---

func TestWithProviderLimits_TPM_Generate(t *testing.T) {
	base := &stubModel{id: "b", generateFn: func(context.Context, []schema.Message, ...GenerateOption) (*schema.AIMessage, error) {
		return &schema.AIMessage{Usage: schema.Usage{TotalTokens: 60}}, nil
	}}
	m := ApplyMiddleware(base, WithProviderLimits(ProviderLimits{TPM: 100}))
	for i := range 2 {
		if _, err := m.Generate(context.Background(), nil); err != nil {
			t.Fatalf("call %d: unexpected error %v", i, err)
		}
	}
	_, err := m.Generate(context.Background(), nil)
	var ce *core.Error
	if !errors.As(err, &ce) || ce.Code != core.ErrRateLimit {
		t.Fatalf("expected TPM rate limit, got %v", err)
	}
}

func TestWithProviderLimits_TPM_StreamAndCooldown(t *testing.T) {
	base := &stubModel{id: "b", streamFn: func(context.Context, []schema.Message, ...GenerateOption) iter.Seq2[schema.StreamChunk, error] {
		return func(yield func(schema.StreamChunk, error) bool) {
			if !yield(schema.StreamChunk{Delta: "x"}, nil) {
				return
			}
			yield(schema.StreamChunk{Usage: &schema.Usage{TotalTokens: 500}}, nil)
		}
	}}
	m := ApplyMiddleware(base, WithProviderLimits(ProviderLimits{TPM: 100, CooldownOnRetry: time.Millisecond}))
	for range m.Stream(context.Background(), nil) { //nolint:revive // drain
	}
	var got error
	for _, err := range m.Stream(context.Background(), nil) {
		got = err
	}
	if got == nil || !core.IsRetryable(got) {
		t.Fatalf("expected TPM error after cooldown, got %v", got)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	slow := ApplyMiddleware(base, WithProviderLimits(ProviderLimits{TPM: 1, CooldownOnRetry: time.Hour}))
	for range slow.Stream(context.Background(), nil) { //nolint:revive // drain
	}
	if _, err := slow.Generate(ctx, nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled during cooldown, got %v", err)
	}
}

func TestTokenWindow_Expires(t *testing.T) {
	w := &tokenWindow{maxTokens: 10, window: 20 * time.Millisecond}
	w.add(10)
	if w.allow() {
		t.Fatal("budget should be spent")
	}
	time.Sleep(30 * time.Millisecond)
	if !w.allow() {
		t.Fatal("budget should recover after the window")
	}
}

// --- FWC-14: FailoverRouter.Stream must not fail over after partial output ---

func TestFailoverRouter_Stream_NoFailoverAfterPartialOutput(t *testing.T) {
	midErr := core.NewError("test", core.ErrProviderDown, "dropped", nil)
	a := &stubModel{id: "a", streamFn: func(context.Context, []schema.Message, ...GenerateOption) iter.Seq2[schema.StreamChunk, error] {
		return func(yield func(schema.StreamChunk, error) bool) {
			if !yield(schema.StreamChunk{Delta: "partial-A"}, nil) {
				return
			}
			yield(schema.StreamChunk{}, midErr)
		}
	}}
	bCalled := false
	b := &stubModel{id: "b", streamFn: func(context.Context, []schema.Message, ...GenerateOption) iter.Seq2[schema.StreamChunk, error] {
		bCalled = true
		return streamOf("full-B")(context.Background(), nil)
	}}

	var deltas []string
	var gotErr error
	for chunk, err := range NewFailoverRouter(a, b).Stream(context.Background(), nil) {
		if err != nil {
			gotErr = err
			break
		}
		deltas = append(deltas, chunk.Delta)
	}
	if bCalled {
		t.Fatal("failed over to model B after model A had already produced output")
	}
	if len(deltas) != 1 || deltas[0] != "partial-A" {
		t.Fatalf("deltas = %v", deltas)
	}
	if !errors.Is(gotErr, midErr) {
		t.Fatalf("expected the mid-stream error to be surfaced, got %v", gotErr)
	}
}

func TestFailoverRouter_Stream_NoModels(t *testing.T) {
	var got error
	for _, err := range NewFailoverRouter().Stream(context.Background(), nil) {
		got = err
	}
	var ce *core.Error
	if !errors.As(got, &ce) || ce.Code != core.ErrProviderDown {
		t.Fatalf("expected provider_unavailable, got %v", got)
	}
}

// --- FWC-36: suppressing an error must not yield (nil, nil) ---

func TestWithHooks_OnErrorSuppressionReturnsNonNilResponse(t *testing.T) {
	base := &stubModel{id: "b", generateFn: func(context.Context, []schema.Message, ...GenerateOption) (*schema.AIMessage, error) {
		return nil, errors.New("boom")
	}}
	m := ApplyMiddleware(base, WithHooks(Hooks{OnError: func(context.Context, error) error { return nil }}))
	resp, err := m.Generate(context.Background(), nil)
	if err != nil {
		t.Fatalf("expected suppressed error, got %v", err)
	}
	if resp == nil {
		t.Fatal("Generate returned (nil, nil) after OnError suppression")
	}
	if len(resp.ToolCalls) != 0 {
		t.Fatal("expected an empty message")
	}
}

// --- FWC-35: SimpleTokenizer.Decode matches its documentation ---

func TestSimpleTokenizer_DecodePlaceholder(t *testing.T) {
	tok := &SimpleTokenizer{}
	if got := tok.Decode([]int{7, 8, 9}); got != "x x x" {
		t.Fatalf("Decode = %q, want %q", got, "x x x")
	}
	if got := tok.Decode(nil); got != "" {
		t.Fatalf("Decode(nil) = %q", got)
	}
}
