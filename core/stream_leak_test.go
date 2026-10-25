package core

import (
	"context"
	"runtime"
	"testing"
	"time"
)

// infiniteStream yields events forever until the consumer stops.
func infiniteStream() Stream[int] {
	return func(yield func(Event[int], error) bool) {
		for i := 0; ; i++ {
			if !yield(Event[int]{Type: EventData, Payload: i}, nil) {
				return
			}
		}
	}
}

// waitGoroutines polls until the goroutine count drops to at most want, or
// fails the test after a timeout.
func waitGoroutines(t *testing.T, want int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for {
		got := runtime.NumGoroutine()
		if got <= want {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("goroutines leaked: have %d, want <= %d", got, want)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// TestMergeStreams_EarlyBreakNoLeak is a regression test for FWC-08.
func TestMergeStreams_EarlyBreakNoLeak(t *testing.T) {
	before := runtime.NumGoroutine()
	for range 10 {
		for range MergeStreams(context.Background(), infiniteStream(), infiniteStream()) {
			break
		}
	}
	waitGoroutines(t, before)
}

// TestFanOut_ConsumerBreakDoesNotBlockOthers is a regression test for FWC-08:
// one consumer breaking used to block every other branch.
func TestFanOut_ConsumerBreakDoesNotBlockOthers(t *testing.T) {
	before := runtime.NumGoroutine()
	for range 10 {
		src := func(yield func(Event[int], error) bool) {
			for i := range 100 {
				if !yield(Event[int]{Type: EventData, Payload: i}, nil) {
					return
				}
			}
		}
		streams := FanOut[int](context.Background(), src, 2)
		for range streams[0] {
			break
		}
		count := 0
		done := make(chan struct{})
		go func() {
			defer close(done)
			for range streams[1] {
				count++
			}
		}()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatal("second FanOut consumer blocked after the first one stopped")
		}
		if count != 100 {
			t.Fatalf("second consumer got %d events, want 100", count)
		}
	}
	waitGoroutines(t, before)
}

// TestFanOut_AllConsumersBreakStopsProducer checks the producer quits once
// every branch has stopped, even on an infinite source.
func TestFanOut_AllConsumersBreakStopsProducer(t *testing.T) {
	before := runtime.NumGoroutine()
	for range 10 {
		streams := FanOut[int](context.Background(), infiniteStream(), 2)
		for _, s := range streams {
			for range s {
				break
			}
		}
	}
	waitGoroutines(t, before)
}

// TestBufferedStream_EarlyBreakNoLeak is a regression test for FWC-08.
func TestBufferedStream_EarlyBreakNoLeak(t *testing.T) {
	before := runtime.NumGoroutine()
	for range 10 {
		bs := NewBufferedStream[int](context.Background(), infiniteStream(), 4)
		for range bs.Iter() {
			break
		}
	}
	waitGoroutines(t, before)
}
