package core

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"
)

// mockComponent is a test Lifecycle implementation that records calls.
type mockComponent struct {
	name       string
	startErr   error
	stopErr    error
	started    bool
	stopped    bool
	startOrder *[]string
	stopOrder  *[]string
	health     HealthStatus
}

func (m *mockComponent) Start(_ context.Context) error {
	if m.startOrder != nil {
		*m.startOrder = append(*m.startOrder, m.name)
	}
	if m.startErr != nil {
		return m.startErr
	}
	m.started = true
	return nil
}

func (m *mockComponent) Stop(_ context.Context) error {
	if m.stopOrder != nil {
		*m.stopOrder = append(*m.stopOrder, m.name)
	}
	if m.stopErr != nil {
		return m.stopErr
	}
	m.stopped = true
	return nil
}

func (m *mockComponent) Health() HealthStatus {
	return m.health
}

func TestNewApp(t *testing.T) {
	app := NewApp()
	if app == nil {
		t.Fatal("NewApp() returned nil")
	}
}

func TestApp_Register(t *testing.T) {
	app := NewApp()
	c1 := &mockComponent{name: "c1"}
	c2 := &mockComponent{name: "c2"}
	app.Register(c1, c2)

	// Verify registration by starting (starts both)
	if err := app.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if !c1.started || !c2.started {
		t.Error("expected both components to be started")
	}
}

func TestApp_Start_Order(t *testing.T) {
	var order []string
	app := NewApp()
	app.Register(
		&mockComponent{name: "a", startOrder: &order},
		&mockComponent{name: "b", startOrder: &order},
		&mockComponent{name: "c", startOrder: &order},
	)

	if err := app.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	want := []string{"a", "b", "c"}
	if len(order) != len(want) {
		t.Fatalf("start order length = %d, want %d", len(order), len(want))
	}
	for i, name := range want {
		if order[i] != name {
			t.Errorf("start order[%d] = %q, want %q", i, order[i], name)
		}
	}
}

func TestApp_Start_RollbackOnFailure(t *testing.T) {
	var stopOrder []string

	c1 := &mockComponent{name: "c1", stopOrder: &stopOrder}
	c2 := &mockComponent{name: "c2", stopOrder: &stopOrder}
	c3 := &mockComponent{name: "c3", startErr: fmt.Errorf("c3 failed")}

	app := NewApp()
	app.Register(c1, c2, c3)

	err := app.Start(context.Background())
	if err == nil {
		t.Fatal("Start() expected error, got nil")
	}
	if !strings.Contains(err.Error(), "c3 failed") {
		t.Errorf("error = %q, want to contain %q", err.Error(), "c3 failed")
	}

	// c2 stopped before c1 (reverse order of previously started).
	want := []string{"c2", "c1"}
	if len(stopOrder) != len(want) {
		t.Fatalf("rollback stop order length = %d, want %d", len(stopOrder), len(want))
	}
	for i, name := range want {
		if stopOrder[i] != name {
			t.Errorf("stop order[%d] = %q, want %q", i, stopOrder[i], name)
		}
	}
}

func TestApp_Shutdown_ReverseOrder(t *testing.T) {
	var stopOrder []string
	app := NewApp()
	app.Register(
		&mockComponent{name: "a", stopOrder: &stopOrder},
		&mockComponent{name: "b", stopOrder: &stopOrder},
		&mockComponent{name: "c", stopOrder: &stopOrder},
	)

	if err := app.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	if err := app.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}

	want := []string{"c", "b", "a"}
	if len(stopOrder) != len(want) {
		t.Fatalf("stop order length = %d, want %d", len(stopOrder), len(want))
	}
	for i, name := range want {
		if stopOrder[i] != name {
			t.Errorf("stop order[%d] = %q, want %q", i, stopOrder[i], name)
		}
	}
}

func TestApp_Shutdown_NotRunning(t *testing.T) {
	app := NewApp()
	app.Register(&mockComponent{name: "c1"})

	err := app.Shutdown(context.Background())
	if err != nil {
		t.Errorf("Shutdown() on non-running app error = %v, want nil", err)
	}
}

func TestApp_Shutdown_CollectsErrors(t *testing.T) {
	app := NewApp()
	app.Register(
		&mockComponent{name: "c1", stopErr: fmt.Errorf("c1 stop failed")},
		&mockComponent{name: "c2", stopErr: fmt.Errorf("c2 stop failed")},
	)

	if err := app.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	err := app.Shutdown(context.Background())
	if err == nil {
		t.Fatal("Shutdown() expected error, got nil")
	}
	if !strings.Contains(err.Error(), "c2 stop failed") {
		t.Errorf("error should contain c2 failure: %q", err.Error())
	}
	if !strings.Contains(err.Error(), "c1 stop failed") {
		t.Errorf("error should contain c1 failure: %q", err.Error())
	}
}

func TestApp_HealthCheck(t *testing.T) {
	now := time.Now()
	app := NewApp()
	app.Register(
		&mockComponent{
			name: "healthy",
			health: HealthStatus{
				Status:    HealthHealthy,
				Message:   "all good",
				Timestamp: now,
			},
		},
		&mockComponent{
			name: "degraded",
			health: HealthStatus{
				Status:    HealthDegraded,
				Message:   "high latency",
				Timestamp: now,
			},
		},
	)

	statuses := app.HealthCheck()
	if len(statuses) != 2 {
		t.Fatalf("HealthCheck() returned %d statuses, want 2", len(statuses))
	}
	if statuses[0].Status != HealthHealthy {
		t.Errorf("statuses[0].Status = %q, want %q", statuses[0].Status, HealthHealthy)
	}
	if statuses[1].Status != HealthDegraded {
		t.Errorf("statuses[1].Status = %q, want %q", statuses[1].Status, HealthDegraded)
	}
}

func TestApp_HealthCheck_Empty(t *testing.T) {
	app := NewApp()
	statuses := app.HealthCheck()
	if len(statuses) != 0 {
		t.Errorf("HealthCheck() on empty app returned %d statuses, want 0", len(statuses))
	}
}

func TestHealthState_Values(t *testing.T) {
	tests := []struct {
		state HealthState
		want  string
	}{
		{HealthHealthy, "healthy"},
		{HealthDegraded, "degraded"},
		{HealthUnhealthy, "unhealthy"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if string(tt.state) != tt.want {
				t.Errorf("HealthState = %q, want %q", string(tt.state), tt.want)
			}
		})
	}
}

func TestApp_Shutdown_TwiceIsNoop(t *testing.T) {
	app := NewApp()
	app.Register(&mockComponent{name: "c1"})

	if err := app.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	if err := app.Shutdown(context.Background()); err != nil {
		t.Fatalf("first Shutdown() error = %v", err)
	}

	// Second shutdown is a no-op because running is false.
	if err := app.Shutdown(context.Background()); err != nil {
		t.Errorf("second Shutdown() error = %v, want nil", err)
	}
}
