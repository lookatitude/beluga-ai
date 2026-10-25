package inmemory

import (
	"context"
	"errors"
	"runtime"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/v2/state"
)

// TestStore_ClosedErrorsUseSentinel is a regression test for FWC-21: closed
// store errors were plain strings, so errors.Is(err, state.ErrStoreClosed)
// was false.
func TestStore_ClosedErrorsUseSentinel(t *testing.T) {
	ctx := context.Background()
	s := New()
	_ = s.Close()
	_, errGet := s.Get(ctx, "k")
	_, _, errGV := s.GetVersioned(ctx, "k")
	errSet := s.Set(ctx, "k", 1)
	_, errCAS := s.CompareAndSwap(ctx, "k", 0, 1)
	errDel := s.Delete(ctx, "k")
	var errWatch error
	for _, err := range s.Watch(ctx, "k") {
		errWatch = err
	}
	for name, err := range map[string]error{"get": errGet, "get_versioned": errGV, "set": errSet, "cas": errCAS, "delete": errDel, "watch": errWatch} {
		if !errors.Is(err, state.ErrStoreClosed) {
			t.Errorf("%s: err = %v, want ErrStoreClosed", name, err)
		}
	}
}

func waitForGoroutines(t *testing.T, want int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for runtime.NumGoroutine() > want {
		if time.Now().After(deadline) {
			t.Fatalf("goroutines leaked: have %d, want <= %d", runtime.NumGoroutine(), want)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// TestStore_WatchBreakReleasesGoroutine is a regression test for FWC-22: the
// per-watch helper goroutine waited only for ctx or Close, so with a
// long-lived ctx every Watch leaked one goroutine.
func TestStore_WatchBreakReleasesGoroutine(t *testing.T) {
	s := New()
	defer s.Close()
	ctx := context.Background()
	before := runtime.NumGoroutine()
	for i := range 10 {
		seq := s.Watch(ctx, "k")
		go func() { _ = s.Set(ctx, "k", i) }()
		for range seq {
			break
		}
	}
	waitForGoroutines(t, before)
}

// TestWatchSeq_BufferedBreakReleasesRelay is a regression test for FWC-22:
// the buffered relay ranged over a watch bound to the parent ctx and stayed
// blocked after the consumer returned.
func TestWatchSeq_BufferedBreakReleasesRelay(t *testing.T) {
	s := New()
	defer s.Close()
	ctx := context.Background()
	before := runtime.NumGoroutine()
	for i := range 10 {
		seq := state.WatchSeq(ctx, s, "k", state.WithBufferSize(4))
		go func() { _ = s.Set(ctx, "k", i) }()
		for range seq {
			break
		}
	}
	waitForGoroutines(t, before)
}
