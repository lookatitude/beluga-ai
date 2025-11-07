package health

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
func TestNewHealthCheck(t *testing.T) {
	checkFunc := func() *HealthCheckResult {
		return &HealthCheckResult{
			Status:    StatusHealthy,
			Message:   "Test check",
			CheckName: "test",
		}
	}

	hc := NewHealthCheck("test_check", "test_component", 30*time.Second, checkFunc)

	assert.NotNil(t, hc)
	assert.Equal(t, "test_check", hc.Name)
	assert.Equal(t, "test_component", hc.ComponentID)
	assert.Equal(t, 30*time.Second, hc.Interval)
	assert.Equal(t, 10*time.Second, hc.Timeout)
	assert.NotNil(t, hc.Check)
	assert.NotNil(t, hc.LastResult)
	assert.Equal(t, StatusUnknown, hc.LastResult.Status)
	assert.Equal(t, 3, hc.MaxRetries)
	assert.Equal(t, 2*time.Second, hc.RetryDelay)
	assert.NotNil(t, hc.Alerts)
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
}

func TestHealthCheckStartStop(t *testing.T) {
	called := false
	checkFunc := func() *HealthCheckResult {
		called = true
		return &HealthCheckResult{
			Status:    StatusHealthy,
			Message:   "Test check",
			CheckName: "test",
			Timestamp: time.Now(),
		}
	}

	hc := NewHealthCheck("test_check", "test_component", 100*time.Millisecond, checkFunc)

	hc.Start()

	// Wait for at least one check to run
	time.Sleep(150 * time.Millisecond)

	hc.Stop()
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()

	assert.True(t, called, "Health check function should have been called")
}

func TestHealthCheckRunCheck(t *testing.T) {
	t.Run("successful check", func(t *testing.T) {
		callCount := 0
		checkFunc := func() *HealthCheckResult {
			callCount++
			return &HealthCheckResult{
				Status:      StatusHealthy,
				Message:     "All good",
				CheckName:   "test_check",
				ComponentID: "test_component",
				Timestamp:   time.Now(),
			}
		}

		hc := NewHealthCheck("test_check", "test_component", time.Minute, checkFunc)

		hc.RunCheck()

		assert.Equal(t, 1, callCount)
		assert.Equal(t, StatusHealthy, hc.LastResult.Status)
		assert.Equal(t, "All good", hc.LastResult.Message)
	})

	t.Run("failing check with retry", func(t *testing.T) {
		callCount := 0
		checkFunc := func() *HealthCheckResult {
			callCount++
			if callCount < 3 {
				return &HealthCheckResult{
					Status:      StatusUnhealthy,
					Message:     "Still failing",
					CheckName:   "test_check",
					ComponentID: "test_component",
					Timestamp:   time.Now(),
				}
			}
			return &HealthCheckResult{
				Status:      StatusHealthy,
				Message:     "Recovered",
				CheckName:   "test_check",
				ComponentID: "test_component",
				Timestamp:   time.Now(),
			}
		}

		hc := NewHealthCheck("test_check", "test_component", time.Minute, checkFunc)
		hc.MaxRetries = 3
		hc.RetryDelay = 10 * time.Millisecond

		hc.RunCheck()

		assert.Equal(t, 3, callCount) // Initial + 2 retries
		assert.Equal(t, StatusHealthy, hc.LastResult.Status)
		assert.Contains(t, hc.LastResult.Message, "Recovered on retry")
	})

	t.Run("timeout check", func(t *testing.T) {
		checkFunc := func() *HealthCheckResult {
			time.Sleep(10 * time.Millisecond)
			return &HealthCheckResult{
				Status:    StatusHealthy,
				Message:   "Should timeout",
				CheckName: "test_check",
			}
		}

		hc := NewHealthCheck("test_check", "test_component", time.Minute, checkFunc)
		hc.Timeout = 50 * time.Millisecond
		hc.MaxRetries = 0 // Disable retries for timeout test

		hc.RunCheck()

		assert.Equal(t, StatusUnhealthy, hc.LastResult.Status)
		assert.Contains(t, hc.LastResult.Message, "timed out")
	})

	t.Run("nil result handling", func(t *testing.T) {
		checkFunc := func() *HealthCheckResult {
			return nil
		}

		hc := NewHealthCheck("test_check", "test_component", time.Minute, checkFunc)

		hc.RunCheck()
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()

		assert.Equal(t, StatusUnhealthy, hc.LastResult.Status)
		assert.Contains(t, hc.LastResult.Message, "returned nil result")
	})
}

func TestHealthCheckRegisterAlert(t *testing.T) {
	hc := NewHealthCheck("test_check", "test_component", time.Minute, func() *HealthCheckResult {
		return &HealthCheckResult{
			Status:    StatusHealthy,
			Message:   "Test",
			CheckName: "test_check",
		}
	})

	alertCalled := false
	alertFunc := func(result *HealthCheckResult) {
		alertCalled = true
		assert.Equal(t, "test_check", result.CheckName)
	}

	hc.RegisterAlert(alertFunc)
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	assert.Len(t, hc.Alerts, 1)

	hc.RunCheck()
	// Give time for goroutine to execute
	time.Sleep(10 * time.Millisecond)
	assert.True(t, alertCalled)
}

func TestHealthCheckGetLastResult(t *testing.T) {
	hc := NewHealthCheck("test_check", "test_component", time.Minute, func() *HealthCheckResult {
		return &HealthCheckResult{
			Status:    StatusHealthy,
			Message:   "Test result",
			CheckName: "test_check",
		}
	})

	// Initially unknown
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	result := hc.GetLastResult()
	assert.Equal(t, StatusUnknown, result.Status)

	// After running check
	hc.RunCheck()
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	result = hc.GetLastResult()
	assert.Equal(t, StatusHealthy, result.Status)
	assert.Equal(t, "Test result", result.Message)
}

func TestHealthCheckManager(t *testing.T) {
	manager := NewHealthCheckManager()
	assert.NotNil(t, manager)
	assert.NotNil(t, manager.checks)
	assert.NotNil(t, manager.Logger)
}

func TestHealthCheckManagerAddCheck(t *testing.T) {
	manager := NewHealthCheckManager()

	checkFunc := func() *HealthCheckResult {
		return &HealthCheckResult{
			Status:    StatusHealthy,
			Message:   "Test",
			CheckName: "test_check",
		}
	}

	hc := NewHealthCheck("test_check", "test_component", time.Minute, checkFunc)

	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	t.Run("add new check", func(t *testing.T) {
		err := manager.AddCheck(hc)
		assert.NoError(t, err)
		assert.Len(t, manager.checks, 1)
	})

	t.Run("add duplicate check", func(t *testing.T) {
		hc2 := NewHealthCheck("test_check", "test_component", time.Minute, checkFunc)
		err := manager.AddCheck(hc2)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already exists")
	})
}

func TestHealthCheckManagerRemoveCheck(t *testing.T) {
	manager := NewHealthCheckManager()

	checkFunc := func() *HealthCheckResult {
		return &HealthCheckResult{
			Status:    StatusHealthy,
			Message:   "Test",
			CheckName: "test_check",
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	hc := NewHealthCheck("test_check", "test_component", time.Minute, checkFunc)
	manager.AddCheck(hc)

	t.Run("remove existing check", func(t *testing.T) {
		err := manager.RemoveCheck("test_component", "test_check")
		assert.NoError(t, err)
		assert.Len(t, manager.checks, 0)
	})

	t.Run("remove non-existent check", func(t *testing.T) {
		err := manager.RemoveCheck("non_existent", "test_check")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestHealthCheckManagerStartStopAllChecks(t *testing.T) {
	manager := NewHealthCheckManager()

	checkFunc := func() *HealthCheckResult {
		return &HealthCheckResult{
			Status:    StatusHealthy,
			Message:   "Test",
			CheckName: "test_check",
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	hc1 := NewHealthCheck("check1", "component1", 100*time.Millisecond, checkFunc)
	hc2 := NewHealthCheck("check2", "component2", 100*time.Millisecond, checkFunc)

	manager.AddCheck(hc1)
	manager.AddCheck(hc2)

	manager.StartAllChecks()

	// Wait for checks to run
	time.Sleep(150 * time.Millisecond)

	manager.StopAllChecks()

	// Verify checks are stopped
	results := manager.GetCheckResults()
	assert.Len(t, results, 2)
}

func TestHealthCheckManagerGetCheckResults(t *testing.T) {
	manager := NewHealthCheckManager()

	checkFunc := func() *HealthCheckResult {
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
		return &HealthCheckResult{
			Status:      StatusHealthy,
			Message:     "Test",
			CheckName:   "test_check",
			ComponentID: "test_component",
			Timestamp:   time.Now(),
		}
	}

	hc := NewHealthCheck("test_check", "test_component", time.Minute, checkFunc)
	manager.AddCheck(hc)

	hc.RunCheck()

	results := manager.GetCheckResults()
	assert.Len(t, results, 1)
	assert.Contains(t, results, "test_component:test_check")
	assert.Equal(t, StatusHealthy, results["test_component:test_check"].Status)
}

func TestHealthCheckManagerCheckSystemHealth(t *testing.T) {
	t.Run("all healthy", func(t *testing.T) {
		manager := NewHealthCheckManager()
		healthyCheck := NewHealthCheck("healthy", "component", time.Minute, func() *HealthCheckResult {
			return &HealthCheckResult{
				Status:    StatusHealthy,
				Message:   "All good",
				CheckName: "healthy",
			}
		})
		manager.AddCheck(healthyCheck)

		overallStatus, results := manager.CheckSystemHealth()
		assert.Equal(t, StatusHealthy, overallStatus)
		assert.Len(t, results, 1)
	})

	t.Run("mixed status", func(t *testing.T) {
		manager := NewHealthCheckManager()
		healthyCheck := NewHealthCheck("healthy", "component", time.Minute, func() *HealthCheckResult {
			return &HealthCheckResult{
				Status:    StatusHealthy,
				Message:   "All good",
				CheckName: "healthy",
			}
		})
		manager.AddCheck(healthyCheck)
		healthyCheck.RunCheck()

		degradedCheck := NewHealthCheck("degraded", "component2", time.Minute, func() *HealthCheckResult {
			return &HealthCheckResult{
				Status:    StatusDegraded,
				Message:   "Some issues",
				CheckName: "degraded",
			}
		})
		manager.AddCheck(degradedCheck)
		// Run the check so it has a result
		degradedCheck.RunCheck()

		overallStatus, results := manager.CheckSystemHealth()
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
		assert.Equal(t, StatusDegraded, overallStatus)
		assert.Len(t, results, 2)
	})

	t.Run("unhealthy status", func(t *testing.T) {
		manager := NewHealthCheckManager()
		unhealthyCheck := NewHealthCheck("unhealthy", "component3", time.Minute, func() *HealthCheckResult {
			return &HealthCheckResult{
				Status:    StatusUnhealthy,
				Message:   "Major issues",
				CheckName: "unhealthy",
			}
		})
		manager.AddCheck(unhealthyCheck)
		unhealthyCheck.RunCheck()

		overallStatus, results := manager.CheckSystemHealth()
		assert.Equal(t, StatusUnhealthy, overallStatus)
		assert.Len(t, results, 1)
	})
}

func TestCreateAgentHealthCheckFunc(t *testing.T) {
	t.Run("healthy agent", func(t *testing.T) {
		getHealthFunc := func() map[string]interface{} {
			return map[string]interface{}{
				"state":         "running",
				"error_count":   0,
				"name":          "test_agent",
				"response_time": 150,
			}
		}
		checkFunc := CreateAgentHealthCheckFunc(getHealthFunc)
		result := checkFunc()
		assert.NotNil(t, result)
		assert.Equal(t, StatusHealthy, result.Status)
		assert.Contains(t, result.Message, "healthy")
		assert.Equal(t, "test_agent", result.ComponentID)
	})

	t.Run("error state agent", func(t *testing.T) {
		getHealthFunc := func() map[string]interface{} {
			return map[string]interface{}{
				"state": "error",
				"name":  "test_agent",
			}
		}
		checkFunc := CreateAgentHealthCheckFunc(getHealthFunc)
		result := checkFunc()
		assert.Equal(t, StatusUnhealthy, result.Status)
		assert.Contains(t, result.Message, "error state")
	})

	t.Run("paused agent", func(t *testing.T) {
		getHealthFunc := func() map[string]interface{} {
			return map[string]interface{}{
				"state": "paused",
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
				"name":  "test_agent",
			}
		}
		checkFunc := CreateAgentHealthCheckFunc(getHealthFunc)
		result := checkFunc()
		assert.Equal(t, StatusDegraded, result.Status)
		assert.Contains(t, result.Message, "paused")
	})

	t.Run("high error count", func(t *testing.T) {
		getHealthFunc := func() map[string]interface{} {
			return map[string]interface{}{
				"state":       "running",
				"error_count": 10,
				"name":        "test_agent",
			}
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
		}
		checkFunc := CreateAgentHealthCheckFunc(getHealthFunc)
		result := checkFunc()
		assert.Equal(t, StatusUnhealthy, result.Status)
		assert.Contains(t, result.Message, "high error count")
	})
}

func TestHealthStatus(t *testing.T) {
	tests := []struct {
		status   HealthStatus
		expected string
	}{
		{StatusHealthy, "healthy"},
		{StatusDegraded, "degraded"},
		{StatusUnhealthy, "unhealthy"},
		{StatusUnknown, "unknown"},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.status))
		})
	}
}

func TestHealthCheckConcurrency(t *testing.T) {
	manager := NewHealthCheckManager()

	var counter int
	var mu sync.Mutex

	checkFunc := func() *HealthCheckResult {
		mu.Lock()
		counter++
		mu.Unlock()

		return &HealthCheckResult{
			Status:      StatusHealthy,
			Message:     "Concurrent check",
			CheckName:   "concurrent_check",
			ComponentID: "test_component",
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
			Timestamp:   time.Now(),
		}
	}

	hc := NewHealthCheck("concurrent_check", "test_component", time.Minute, checkFunc)
	manager.AddCheck(hc)

	var wg sync.WaitGroup
	numGoroutines := 10

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			hc.RunCheck()
		}()
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	}

	wg.Wait()

	mu.Lock()
	finalCount := counter
	mu.Unlock()

	assert.Equal(t, numGoroutines, finalCount)
}

// Benchmark tests
func BenchmarkHealthCheck_RunCheck(b *testing.B) {
	checkFunc := func() *HealthCheckResult {
		return &HealthCheckResult{
			Status:    StatusHealthy,
			Message:   "Benchmark check",
			CheckName: "bench_check",
			Timestamp: time.Now(),
		}
	}

	hc := NewHealthCheck("bench_check", "bench_component", time.Minute, checkFunc)
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hc.RunCheck()
	}
}

func BenchmarkHealthCheckManager_GetCheckResults(b *testing.B) {
	manager := NewHealthCheckManager()

	// Add some checks
	for i := 0; i < 10; i++ {
		checkFunc := func() *HealthCheckResult {
			return &HealthCheckResult{
				Status:      StatusHealthy,
				Message:     "Bench check",
				CheckName:   "bench_check",
				ComponentID: "bench_component",
				Timestamp:   time.Now(),
			}
		}
		hc := NewHealthCheck("bench_check", "bench_component", time.Minute, checkFunc)
		manager.AddCheck(hc)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		results := manager.GetCheckResults()
		_ = results
	}
}

func BenchmarkCreateAgentHealthCheckFunc(b *testing.B) {
	getHealthFunc := func() map[string]interface{} {
		return map[string]interface{}{
			"state":       "running",
			"error_count": 0,
			"name":        "bench_agent",
		}
	}

	checkFunc := CreateAgentHealthCheckFunc(getHealthFunc)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := checkFunc()
		_ = result
	}
}
