package o11y

import (
	"context"
	"testing"
	"time"
)

type alwaysHealthy struct{}

func (alwaysHealthy) HealthCheck(ctx context.Context) HealthResult {
	return HealthResult{
		Status:  Healthy,
		Message: "OK",
	}
}

type alwaysUnhealthy struct{}

func (alwaysUnhealthy) HealthCheck(ctx context.Context) HealthResult {
	return HealthResult{
		Status:  Unhealthy,
		Message: "connection refused",
	}
}

func TestHealthRegistry(t *testing.T) {
	t.Run("empty registry returns nil", func(t *testing.T) {
		reg := NewHealthRegistry()
		results := reg.CheckAll(context.Background())
		if results != nil {
			t.Errorf("expected nil results, got %v", results)
		}
	})

	t.Run("single healthy checker", func(t *testing.T) {
		reg := NewHealthRegistry()
		reg.Register("db", alwaysHealthy{})

		results := reg.CheckAll(context.Background())
		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if results[0].Status != Healthy {
			t.Errorf("expected Healthy, got %s", results[0].Status)
		}
		if results[0].Component != "db" {
			t.Errorf("expected component 'db', got %q", results[0].Component)
		}
		if results[0].Timestamp.IsZero() {
			t.Error("expected non-zero timestamp")
		}
	})

	t.Run("multiple checkers run concurrently", func(t *testing.T) {
		reg := NewHealthRegistry()
		reg.Register("db", alwaysHealthy{})
		reg.Register("cache", alwaysHealthy{})
		reg.Register("llm", alwaysUnhealthy{})

		results := reg.CheckAll(context.Background())
		if len(results) != 3 {
			t.Fatalf("expected 3 results, got %d", len(results))
		}

		statuses := make(map[HealthStatus]int)
		for _, r := range results {
			statuses[r.Status]++
		}
		if statuses[Healthy] != 2 {
			t.Errorf("expected 2 healthy, got %d", statuses[Healthy])
		}
		if statuses[Unhealthy] != 1 {
			t.Errorf("expected 1 unhealthy, got %d", statuses[Unhealthy])
		}
	})

	t.Run("register replaces existing checker", func(t *testing.T) {
		reg := NewHealthRegistry()
		reg.Register("db", alwaysUnhealthy{})
		reg.Register("db", alwaysHealthy{})

		results := reg.CheckAll(context.Background())
		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if results[0].Status != Healthy {
			t.Errorf("expected Healthy after replacement, got %s", results[0].Status)
		}
	})
}

func TestHealthCheckerFunc(t *testing.T) {
	checker := HealthCheckerFunc(func(ctx context.Context) HealthResult {
		return HealthResult{
			Status:    Degraded,
			Message:   "high latency",
			Component: "api",
			Timestamp: time.Now(),
		}
	})

	result := checker.HealthCheck(context.Background())
	if result.Status != Degraded {
		t.Errorf("expected Degraded, got %s", result.Status)
	}
}

func TestHealthStatusConstants(t *testing.T) {
	tests := []struct {
		status HealthStatus
		want   string
	}{
		{Healthy, "healthy"},
		{Degraded, "degraded"},
		{Unhealthy, "unhealthy"},
	}
	for _, tt := range tests {
		if string(tt.status) != tt.want {
			t.Errorf("expected %q, got %q", tt.want, string(tt.status))
		}
	}
}
