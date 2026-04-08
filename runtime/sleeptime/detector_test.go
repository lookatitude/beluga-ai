package sleeptime

import (
	"testing"
	"time"
)

func TestInactivityDetector_IsIdle(t *testing.T) {
	tests := []struct {
		name     string
		timeout  time.Duration
		elapsed  time.Duration
		wantIdle bool
	}{
		{
			name:     "not idle when recently active",
			timeout:  10 * time.Second,
			elapsed:  1 * time.Second,
			wantIdle: false,
		},
		{
			name:     "idle after timeout",
			timeout:  5 * time.Second,
			elapsed:  6 * time.Second,
			wantIdle: true,
		},
		{
			name:     "idle exactly at timeout",
			timeout:  5 * time.Second,
			elapsed:  5 * time.Second,
			wantIdle: true,
		},
		{
			name:     "not idle just before timeout",
			timeout:  5 * time.Second,
			elapsed:  4 * time.Second,
			wantIdle: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			now := time.Now()
			d := NewInactivityDetector(tt.timeout)
			d.lastSeen = now
			d.nowFn = func() time.Time { return now.Add(tt.elapsed) }

			got := d.IsIdle()
			if got != tt.wantIdle {
				t.Errorf("IsIdle() = %v, want %v", got, tt.wantIdle)
			}
		})
	}
}

func TestInactivityDetector_OnActivity(t *testing.T) {
	now := time.Now()
	d := NewInactivityDetector(5 * time.Second)
	d.lastSeen = now.Add(-10 * time.Second) // simulate old activity
	d.nowFn = func() time.Time { return now }

	// Should be idle before OnActivity.
	if !d.IsIdle() {
		t.Fatal("expected idle before OnActivity")
	}

	d.OnActivity()

	// Should not be idle after OnActivity.
	if d.IsIdle() {
		t.Fatal("expected not idle after OnActivity")
	}
}

func TestInactivityDetector_MinTimeout(t *testing.T) {
	d := NewInactivityDetector(100 * time.Millisecond)
	if d.timeout < time.Second {
		t.Errorf("timeout = %v, want >= 1s (clamped)", d.timeout)
	}
}

func TestInactivityDetector_ConcurrentAccess(t *testing.T) {
	d := NewInactivityDetector(2 * time.Second)
	done := make(chan struct{})

	go func() {
		defer close(done)
		for i := 0; i < 100; i++ {
			d.OnActivity()
		}
	}()

	for i := 0; i < 100; i++ {
		_ = d.IsIdle()
	}

	<-done
}
