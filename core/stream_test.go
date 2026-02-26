package core

import (
	"context"
	"fmt"
	"testing"
)

// makeStream creates a Stream from a slice of events.
func makeStream[T any](events []Event[T]) Stream[T] {
	return func(yield func(Event[T], error) bool) {
		for _, e := range events {
			if !yield(e, nil) {
				return
			}
		}
	}
}

// makeErrorStream creates a Stream that yields events then an error.
func makeErrorStream[T any](events []Event[T], err error) Stream[T] {
	return func(yield func(Event[T], error) bool) {
		for _, e := range events {
			if !yield(e, nil) {
				return
			}
		}
		yield(Event[T]{}, err)
	}
}

func TestCollectStream(t *testing.T) {
	t.Run("empty_stream", func(t *testing.T) {
		stream := makeStream[string](nil)
		events, err := CollectStream(stream)
		if err != nil {
			t.Fatalf("CollectStream() error = %v", err)
		}
		if len(events) != 0 {
			t.Errorf("len(events) = %d, want 0", len(events))
		}
	})

	t.Run("multiple_events", func(t *testing.T) {
		input := []Event[string]{
			{Type: EventData, Payload: "hello"},
			{Type: EventData, Payload: "world"},
			{Type: EventDone, Payload: ""},
		}
		stream := makeStream(input)
		events, err := CollectStream(stream)
		if err != nil {
			t.Fatalf("CollectStream() error = %v", err)
		}
		if len(events) != 3 {
			t.Fatalf("len(events) = %d, want 3", len(events))
		}
		if events[0].Payload != "hello" {
			t.Errorf("events[0].Payload = %q, want %q", events[0].Payload, "hello")
		}
		if events[1].Payload != "world" {
			t.Errorf("events[1].Payload = %q, want %q", events[1].Payload, "world")
		}
	})

	t.Run("stops_on_error", func(t *testing.T) {
		input := []Event[string]{
			{Type: EventData, Payload: "ok"},
		}
		stream := makeErrorStream(input, fmt.Errorf("stream failed"))
		events, err := CollectStream(stream)
		if err == nil {
			t.Fatal("CollectStream() expected error, got nil")
		}
		if err.Error() != "stream failed" {
			t.Errorf("error = %q, want %q", err.Error(), "stream failed")
		}
		if len(events) != 1 {
			t.Errorf("len(events) = %d, want 1 (events before error)", len(events))
		}
	})
}

func TestMapStream(t *testing.T) {
	t.Run("transform_payload", func(t *testing.T) {
		input := []Event[int]{
			{Type: EventData, Payload: 1},
			{Type: EventData, Payload: 2},
			{Type: EventData, Payload: 3},
		}
		stream := makeStream(input)

		mapped := MapStream(stream, func(e Event[int]) (Event[string], error) {
			return Event[string]{
				Type:    e.Type,
				Payload: fmt.Sprintf("num-%d", e.Payload),
			}, nil
		})

		events, err := CollectStream(mapped)
		if err != nil {
			t.Fatalf("CollectStream() error = %v", err)
		}
		if len(events) != 3 {
			t.Fatalf("len(events) = %d, want 3", len(events))
		}
		if events[0].Payload != "num-1" {
			t.Errorf("events[0].Payload = %q, want %q", events[0].Payload, "num-1")
		}
		if events[2].Payload != "num-3" {
			t.Errorf("events[2].Payload = %q, want %q", events[2].Payload, "num-3")
		}
	})

	t.Run("map_error_stops_stream", func(t *testing.T) {
		input := []Event[int]{
			{Type: EventData, Payload: 1},
			{Type: EventData, Payload: 2},
		}
		stream := makeStream(input)

		mapped := MapStream(stream, func(e Event[int]) (Event[string], error) {
			if e.Payload == 2 {
				return Event[string]{}, fmt.Errorf("bad value")
			}
			return Event[string]{Payload: "ok"}, nil
		})

		events, err := CollectStream(mapped)
		if err == nil {
			t.Fatal("expected error from map function")
		}
		if len(events) != 1 {
			t.Errorf("len(events) = %d, want 1", len(events))
		}
	})

	t.Run("source_error_propagates", func(t *testing.T) {
		input := []Event[int]{{Type: EventData, Payload: 1}}
		stream := makeErrorStream(input, fmt.Errorf("source error"))

		mapped := MapStream(stream, func(e Event[int]) (Event[string], error) {
			return Event[string]{Payload: "ok"}, nil
		})

		events, err := CollectStream(mapped)
		if err == nil {
			t.Fatal("expected source error to propagate")
		}
		if err.Error() != "source error" {
			t.Errorf("error = %q, want %q", err.Error(), "source error")
		}
		if len(events) != 1 {
			t.Errorf("len(events) = %d, want 1", len(events))
		}
	})
}

func TestFilterStream(t *testing.T) {
	t.Run("filter_by_type", func(t *testing.T) {
		input := []Event[string]{
			{Type: EventData, Payload: "d1"},
			{Type: EventToolCall, Payload: "tc1"},
			{Type: EventData, Payload: "d2"},
			{Type: EventDone, Payload: ""},
		}
		stream := makeStream(input)

		filtered := FilterStream(stream, func(e Event[string]) bool {
			return e.Type == EventData
		})

		events, err := CollectStream(filtered)
		if err != nil {
			t.Fatalf("CollectStream() error = %v", err)
		}
		if len(events) != 2 {
			t.Fatalf("len(events) = %d, want 2", len(events))
		}
		if events[0].Payload != "d1" || events[1].Payload != "d2" {
			t.Errorf("filtered events = %v", events)
		}
	})

	t.Run("filter_all", func(t *testing.T) {
		input := []Event[string]{
			{Type: EventData, Payload: "a"},
			{Type: EventData, Payload: "b"},
		}
		stream := makeStream(input)

		filtered := FilterStream(stream, func(_ Event[string]) bool {
			return false
		})

		events, err := CollectStream(filtered)
		if err != nil {
			t.Fatalf("CollectStream() error = %v", err)
		}
		if len(events) != 0 {
			t.Errorf("len(events) = %d, want 0", len(events))
		}
	})

	t.Run("filter_none", func(t *testing.T) {
		input := []Event[string]{
			{Type: EventData, Payload: "a"},
			{Type: EventData, Payload: "b"},
		}
		stream := makeStream(input)

		filtered := FilterStream(stream, func(_ Event[string]) bool {
			return true
		})

		events, err := CollectStream(filtered)
		if err != nil {
			t.Fatalf("CollectStream() error = %v", err)
		}
		if len(events) != 2 {
			t.Errorf("len(events) = %d, want 2", len(events))
		}
	})

	t.Run("source_error_propagates", func(t *testing.T) {
		input := []Event[string]{{Type: EventData, Payload: "a"}}
		stream := makeErrorStream(input, fmt.Errorf("src err"))

		filtered := FilterStream(stream, func(_ Event[string]) bool {
			return true
		})

		_, err := CollectStream(filtered)
		if err == nil || err.Error() != "src err" {
			t.Errorf("error = %v, want %q", err, "src err")
		}
	})
}

func TestBufferedStream(t *testing.T) {
	t.Run("basic_buffering", func(t *testing.T) {
		input := []Event[string]{
			{Type: EventData, Payload: "a"},
			{Type: EventData, Payload: "b"},
			{Type: EventData, Payload: "c"},
		}
		stream := makeStream(input)

		ctx := context.Background()
		bs := NewBufferedStream(ctx, stream, 10)

		events, err := CollectStream(bs.Iter())
		if err != nil {
			t.Fatalf("CollectStream() error = %v", err)
		}
		if len(events) != 3 {
			t.Fatalf("len(events) = %d, want 3", len(events))
		}
		if events[0].Payload != "a" || events[2].Payload != "c" {
			t.Errorf("unexpected events: %v", events)
		}
	})

	t.Run("cap_and_len", func(t *testing.T) {
		stream := makeStream[string](nil)
		ctx := context.Background()
		bs := NewBufferedStream(ctx, stream, 5)

		if bs.Cap() != 5 {
			t.Errorf("Cap() = %d, want 5", bs.Cap())
		}
	})

	t.Run("bufsize_clamped_to_1", func(t *testing.T) {
		stream := makeStream[string](nil)
		ctx := context.Background()
		bs := NewBufferedStream(ctx, stream, 0)

		if bs.Cap() != 1 {
			t.Errorf("Cap() = %d, want 1 (clamped from 0)", bs.Cap())
		}
	})

	t.Run("context_cancellation_stops_buffering", func(t *testing.T) {
		// A stream that produces events slowly. Cancelling the context
		// should cause the buffered stream to stop producing.
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		slowStream := func(yield func(Event[string], error) bool) {
			for i := 0; i < 100; i++ {
				if !yield(Event[string]{Type: EventData, Payload: fmt.Sprintf("e%d", i)}, nil) {
					return
				}
				// Give the test goroutine a chance to cancel after first few events.
				select {
				case <-ctx.Done():
					return
				default:
				}
			}
		}

		bs := NewBufferedStream(ctx, Stream[string](slowStream), 2)

		// Read one event, then cancel.
		count := 0
		for _, err := range bs.Iter() {
			if err != nil {
				break
			}
			count++
			if count == 1 {
				cancel()
			}
		}
		// We should get at least 1 event but not all 100.
		if count < 1 {
			t.Errorf("expected at least 1 event, got %d", count)
		}
		if count >= 100 {
			t.Errorf("expected fewer than 100 events (context was cancelled), got %d", count)
		}
	})
}

func TestEventType_Values(t *testing.T) {
	types := map[EventType]string{
		EventData:       "data",
		EventToolCall:   "tool_call",
		EventToolResult: "tool_result",
		EventHandoff:    "handoff",
		EventDone:       "done",
		EventError:      "error",
	}
	for typ, want := range types {
		if string(typ) != want {
			t.Errorf("EventType %v = %q, want %q", typ, string(typ), want)
		}
	}
}

func TestEvent_Meta(t *testing.T) {
	e := Event[string]{
		Type:    EventData,
		Payload: "test",
		Meta:    map[string]any{"trace_id": "abc", "latency_ms": 42},
	}

	if e.Meta["trace_id"] != "abc" {
		t.Errorf("Meta[trace_id] = %v, want %q", e.Meta["trace_id"], "abc")
	}
	if e.Meta["latency_ms"] != 42 {
		t.Errorf("Meta[latency_ms] = %v, want 42", e.Meta["latency_ms"])
	}
}

func TestMergeStreams(t *testing.T) {
	s1 := makeStream([]Event[string]{
		{Type: EventData, Payload: "a"},
		{Type: EventData, Payload: "b"},
	})
	s2 := makeStream([]Event[string]{
		{Type: EventData, Payload: "c"},
	})

	ctx := context.Background()
	merged := MergeStreams(ctx, s1, s2)

	events, err := CollectStream(merged)
	if err != nil {
		t.Fatalf("CollectStream() error = %v", err)
	}
	if len(events) != 3 {
		t.Fatalf("len(events) = %d, want 3", len(events))
	}

	// Verify all payloads present (order not guaranteed).
	seen := map[string]bool{}
	for _, e := range events {
		seen[e.Payload] = true
	}
	for _, want := range []string{"a", "b", "c"} {
		if !seen[want] {
			t.Errorf("missing payload %q in merged stream", want)
		}
	}
}

func TestFanOut(t *testing.T) {
	input := []Event[string]{
		{Type: EventData, Payload: "x"},
		{Type: EventData, Payload: "y"},
	}
	src := makeStream(input)

	ctx := context.Background()
	streams := FanOut(ctx, src, 3)

	if len(streams) != 3 {
		t.Fatalf("FanOut returned %d streams, want 3", len(streams))
	}

	for i, s := range streams {
		events, err := CollectStream(s)
		if err != nil {
			t.Fatalf("stream[%d] error = %v", i, err)
		}
		if len(events) != 2 {
			t.Errorf("stream[%d] len = %d, want 2", i, len(events))
		}
	}
}

func TestFlowController(t *testing.T) {
	t.Run("basic_acquire_release", func(t *testing.T) {
		fc := NewFlowController(2)
		ctx := context.Background()

		// Acquire twice should succeed.
		if err := fc.Acquire(ctx); err != nil {
			t.Fatalf("first Acquire() error = %v", err)
		}
		if err := fc.Acquire(ctx); err != nil {
			t.Fatalf("second Acquire() error = %v", err)
		}

		// Release twice to free capacity.
		fc.Release()
		fc.Release()

		// Acquire again should work.
		if err := fc.Acquire(ctx); err != nil {
			t.Fatalf("third Acquire() error = %v", err)
		}
		fc.Release()
	})

	t.Run("try_acquire_when_full", func(t *testing.T) {
		fc := NewFlowController(1)
		ctx := context.Background()

		// First acquire should succeed.
		if err := fc.Acquire(ctx); err != nil {
			t.Fatalf("Acquire() error = %v", err)
		}

		// TryAcquire should fail when full.
		if ok := fc.TryAcquire(); ok {
			t.Error("TryAcquire() = true, want false (flow controller is full)")
		}

		// After release, TryAcquire should succeed.
		fc.Release()
		if ok := fc.TryAcquire(); !ok {
			t.Error("TryAcquire() = false, want true (capacity available)")
		}

		fc.Release()
	})

	t.Run("try_acquire_when_available", func(t *testing.T) {
		fc := NewFlowController(3)

		// TryAcquire should succeed when capacity is available.
		if ok := fc.TryAcquire(); !ok {
			t.Error("TryAcquire() = false, want true")
		}
		if ok := fc.TryAcquire(); !ok {
			t.Error("TryAcquire() = false, want true")
		}

		fc.Release()
		fc.Release()
	})

	t.Run("context_cancellation_during_acquire", func(t *testing.T) {
		fc := NewFlowController(1)
		ctx, cancel := context.WithCancel(context.Background())

		// Fill the controller.
		if err := fc.Acquire(ctx); err != nil {
			t.Fatalf("Acquire() error = %v", err)
		}

		// Start a goroutine that tries to acquire (will block).
		errCh := make(chan error, 1)
		go func() {
			errCh <- fc.Acquire(ctx)
		}()

		// Cancel the context.
		cancel()

		// The blocked acquire should return context.Canceled.
		err := <-errCh
		if err != context.Canceled {
			t.Errorf("Acquire() error = %v, want context.Canceled", err)
		}

		fc.Release()
	})

	t.Run("concurrency_safety", func(t *testing.T) {
		fc := NewFlowController(10)
		ctx := context.Background()

		const goroutines = 20
		const opsPerGoroutine = 50

		// Launch multiple goroutines that acquire and release concurrently.
		errCh := make(chan error, goroutines)
		for i := 0; i < goroutines; i++ {
			go func() {
				for j := 0; j < opsPerGoroutine; j++ {
					if err := fc.Acquire(ctx); err != nil {
						errCh <- err
						return
					}
					// Simulate some work.
					fc.Release()
				}
				errCh <- nil
			}()
		}

		// Wait for all goroutines to complete.
		for i := 0; i < goroutines; i++ {
			if err := <-errCh; err != nil {
				t.Errorf("goroutine error: %v", err)
			}
		}
	})

	t.Run("max_concurrency_clamped_to_1", func(t *testing.T) {
		fc := NewFlowController(0)
		ctx := context.Background()

		// Should allow at least 1 acquisition.
		if err := fc.Acquire(ctx); err != nil {
			t.Fatalf("Acquire() error = %v (maxConcurrency should be clamped to 1)", err)
		}

		// Second acquire should block (but we use TryAcquire to test).
		if fc.TryAcquire() {
			t.Error("TryAcquire() = true, want false (maxConcurrency=1)")
			fc.Release()
		}

		fc.Release()
	})

	t.Run("multiple_release_doesnt_panic", func(t *testing.T) {
		fc := NewFlowController(1)

		// Release without acquire should not panic (graceful handling).
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Release() panicked: %v", r)
			}
		}()

		fc.Release()
		fc.Release()
	})
}
