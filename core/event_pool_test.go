package core

import (
	"sync"
	"testing"
)

// TestEventReset verifies that Reset zeroes every field of an Event.
func TestEventReset(t *testing.T) {
	tests := []struct {
		name string
		seed func() *Event[string]
	}{
		{
			name: "type and payload cleared",
			seed: func() *Event[string] {
				return &Event[string]{Type: EventData, Payload: "hello"}
			},
		},
		{
			name: "error field cleared",
			seed: func() *Event[string] {
				return &Event[string]{
					Type: EventError,
					Err:  NewError("op", ErrInvalidInput, "bad", nil),
				}
			},
		},
		{
			name: "meta map cleared",
			seed: func() *Event[string] {
				return &Event[string]{
					Type: EventData,
					Meta: map[string]any{"trace": "abc"},
				}
			},
		},
		{
			name: "all fields set",
			seed: func() *Event[string] {
				return &Event[string]{
					Type:    EventToolCall,
					Payload: "call",
					Err:     NewError("op", ErrTimeout, "timeout", nil),
					Meta:    map[string]any{"k": "v"},
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := tt.seed()
			e.Reset()

			if e.Type != "" {
				t.Errorf("Type = %q, want empty string", e.Type)
			}
			if e.Payload != "" {
				t.Errorf("Payload = %q, want zero value", e.Payload)
			}
			if e.Err != nil {
				t.Errorf("Err = %v, want nil", e.Err)
			}
			if e.Meta != nil {
				t.Errorf("Meta = %v, want nil", e.Meta)
			}
		})
	}
}

// TestAcquireReleaseEvent verifies the acquire/release lifecycle.
func TestAcquireReleaseEvent(t *testing.T) {
	tests := []struct {
		name    string
		setup   func() *Event[string]
		verify  func(t *testing.T, e *Event[string])
		release bool
	}{
		{
			name: "acquire returns non-nil pointer",
			setup: func() *Event[string] {
				return AcquireEvent[string]()
			},
			verify: func(t *testing.T, e *Event[string]) {
				if e == nil {
					t.Error("AcquireEvent() returned nil")
				}
			},
			release: true,
		},
		{
			name: "acquired event has zero fields",
			setup: func() *Event[string] {
				return AcquireEvent[string]()
			},
			verify: func(t *testing.T, e *Event[string]) {
				if e.Type != "" {
					t.Errorf("Type = %q, want empty", e.Type)
				}
				if e.Payload != "" {
					t.Errorf("Payload = %q, want zero", e.Payload)
				}
				if e.Err != nil {
					t.Errorf("Err = %v, want nil", e.Err)
				}
				if e.Meta != nil {
					t.Errorf("Meta = %v, want nil", e.Meta)
				}
			},
			release: true,
		},
		{
			name: "released event can be re-acquired",
			setup: func() *Event[string] {
				e := AcquireEvent[string]()
				e.Type = EventData
				e.Payload = "temp"
				ReleaseEvent(e)
				return AcquireEvent[string]()
			},
			verify: func(t *testing.T, e *Event[string]) {
				// After release and re-acquire the event must be zeroed.
				if e.Type != "" {
					t.Errorf("re-acquired Type = %q, want empty (not reset)", e.Type)
				}
				if e.Payload != "" {
					t.Errorf("re-acquired Payload = %q, want zero", e.Payload)
				}
			},
			release: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := tt.setup()
			tt.verify(t, e)
			if tt.release && e != nil {
				ReleaseEvent(e)
			}
		})
	}
}

// TestReleaseEventNilSafe verifies that releasing a nil pointer does not panic.
func TestReleaseEventNilSafe(t *testing.T) {
	// Must not panic.
	ReleaseEvent[string](nil)
}

// TestAcquireReleaseEventInt verifies the pool works with a non-string type.
func TestAcquireReleaseEventInt(t *testing.T) {
	e := AcquireEvent[int]()
	if e == nil {
		t.Fatal("AcquireEvent[int]() returned nil")
	}
	e.Type = EventData
	e.Payload = 42
	ReleaseEvent(e)

	e2 := AcquireEvent[int]()
	if e2 == nil {
		t.Fatal("second AcquireEvent[int]() returned nil")
	}
	defer ReleaseEvent(e2)
	// Payload must be zero after reset.
	if e2.Payload != 0 {
		t.Errorf("Payload = %d, want 0 after recycle", e2.Payload)
	}
}

// TestEventPoolConcurrentAccess exercises the pool under heavy concurrent load
// to surface any data races (run with -race flag).
func TestEventPoolConcurrentAccess(t *testing.T) {
	const goroutines = 64
	const iters = 100

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := range goroutines {
		go func(id int) {
			defer wg.Done()
			for range iters {
				e := AcquireEvent[string]()
				e.Type = EventData
				e.Payload = "concurrent"
				e.Meta = map[string]any{"id": id}
				ReleaseEvent(e)
			}
		}(i)
	}

	wg.Wait()
}

// BenchmarkEventPool measures allocations per acquire/release cycle.
// In steady state the pool should return 0 allocs/op.
func BenchmarkEventPool(b *testing.B) {
	b.ReportAllocs()
	// Warm up the pool so b.N iterations operate against recycled objects.
	warmup := AcquireEvent[string]()
	ReleaseEvent(warmup)

	b.ResetTimer()
	for range b.N {
		e := AcquireEvent[string]()
		e.Type = EventData
		e.Payload = "bench"
		ReleaseEvent(e)
	}
}

// BenchmarkEventPoolParallel measures throughput across multiple goroutines.
func BenchmarkEventPoolParallel(b *testing.B) {
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			e := AcquireEvent[string]()
			e.Type = EventData
			e.Payload = "parallel"
			ReleaseEvent(e)
		}
	})
}
