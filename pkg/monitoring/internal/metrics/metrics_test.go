package metrics

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/monitoring/iface"
	"github.com/stretchr/testify/assert"
)

func TestNewMetricsCollector(t *testing.T) {
	collector := NewMetricsCollector()
	assert.NotNil(t, collector)
	assert.NotNil(t, collector.metrics)
	assert.Len(t, collector.metrics, 0)
}

func TestMetricsCollectorCounter(t *testing.T) {
	collector := NewMetricsCollector()
	ctx := context.Background()

	t.Run("increment counter", func(t *testing.T) {
		collector.Counter(ctx, "test_counter", "Test counter", 5, map[string]string{
			"service": "test",
		})

		metric, exists := collector.GetMetric("test_counter", map[string]string{
			"service": "test",
		})
		assert.True(t, exists)
		assert.NotNil(t, metric)
		assert.Equal(t, "test_counter", metric.Name)
		assert.Equal(t, "Test counter", metric.Description)
		assert.Equal(t, float64(5), metric.Value)
		assert.Equal(t, Counter, metric.Type)
		assert.Equal(t, "test", metric.Labels["service"])
	})

	t.Run("increment existing counter", func(t *testing.T) {
		collector.Counter(ctx, "test_counter", "Test counter", 3, map[string]string{
			"service": "test",
		})

		metric, exists := collector.GetMetric("test_counter", map[string]string{
			"service": "test",
		})
		assert.True(t, exists)
		assert.Equal(t, float64(8), metric.Value) // 5 + 3
	})
}

func TestMetricsCollectorGauge(t *testing.T) {
	collector := NewMetricsCollector()
	ctx := context.Background()

	collector.Gauge(ctx, "test_gauge", "Test gauge", 42.5, map[string]string{
		"component": "test",
	})

	metric, exists := collector.GetMetric("test_gauge", map[string]string{
		"component": "test",
	})
	assert.True(t, exists)
	assert.Equal(t, "test_gauge", metric.Name)
	assert.Equal(t, Gauge, metric.Type)
	assert.Equal(t, 42.5, metric.Value)
}

func TestMetricsCollectorHistogram(t *testing.T) {
	collector := NewMetricsCollector()
	ctx := context.Background()

	collector.Histogram(ctx, "test_histogram", "Test histogram", 1.5, map[string]string{
		"endpoint": "/api/test",
	})

	metric, exists := collector.GetMetric("test_histogram", map[string]string{
		"endpoint": "/api/test",
	})
	assert.True(t, exists)
	assert.Equal(t, "test_histogram", metric.Name)
	assert.Equal(t, Histogram, metric.Type)
	assert.Equal(t, 1.5, metric.Value)
}

func TestMetricsCollectorTiming(t *testing.T) {
	collector := NewMetricsCollector()
	ctx := context.Background()

	duration := 150 * time.Millisecond
	collector.Timing(ctx, "test_timing", "Test timing", duration, map[string]string{
		"operation": "test_op",
	})

	metric, exists := collector.GetMetric("test_timing", map[string]string{
		"operation": "test_op",
	})
	assert.True(t, exists)
	assert.Equal(t, "test_timing", metric.Name)
	assert.Equal(t, Histogram, metric.Type)
	assert.Equal(t, duration.Seconds(), metric.Value)
}

func TestMetricsCollectorIncrement(t *testing.T) {
	collector := NewMetricsCollector()
	ctx := context.Background()

	collector.Increment(ctx, "test_increment", "Test increment", map[string]string{
		"type": "test",
	})

	metric, exists := collector.GetMetric("test_increment", map[string]string{
		"type": "test",
	})
	assert.True(t, exists)
	assert.Equal(t, float64(1), metric.Value)
}

func TestMetricsCollectorStartTimer(t *testing.T) {
	collector := NewMetricsCollector()
	ctx := context.Background()

	timer := collector.StartTimer(ctx, "test_timer", map[string]string{
		"operation": "test",
	})
	assert.NotNil(t, timer)

	time.Sleep(10 * time.Millisecond)
	timer.Stop(ctx, "Test timer completed")

	// Timing uses the same name as passed to StartTimer, not with "_duration" suffix
	metric, exists := collector.GetMetric("test_timer", map[string]string{
		"operation": "test",
	})
	assert.True(t, exists)
	assert.NotNil(t, metric)
	assert.True(t, metric.Value >= 0.01) // At least 10ms
}

func TestMetricsCollectorGetMetric(t *testing.T) {
	collector := NewMetricsCollector()

	// Test non-existent metric
	metric, exists := collector.GetMetric("non_existent", nil)
	assert.False(t, exists)
	assert.Nil(t, metric)

	// Add a metric and test retrieval
	ctx := context.Background()
	collector.Counter(ctx, "test_metric", "Test metric", 1, map[string]string{
		"key": "value",
	})

	// Test with correct labels
	metric, exists = collector.GetMetric("test_metric", map[string]string{
		"key": "value",
	})
	assert.True(t, exists)
	assert.NotNil(t, metric)

	// Test with different labels
	metric, exists = collector.GetMetric("test_metric", map[string]string{
		"key": "different_value",
	})
	assert.False(t, exists)
	assert.Nil(t, metric)
}

func TestMetricsCollectorGetAllMetrics(t *testing.T) {
	collector := NewMetricsCollector()
	ctx := context.Background()

	// Initially empty
	metrics := collector.GetAllMetrics()
	assert.Len(t, metrics, 0)

	// Add some metrics
	collector.Counter(ctx, "counter1", "Counter 1", 1, map[string]string{"type": "a"})
	collector.Gauge(ctx, "gauge1", "Gauge 1", 2.0, map[string]string{"type": "b"})
	collector.Histogram(ctx, "hist1", "Histogram 1", 3.0, map[string]string{"type": "c"})

	metrics = collector.GetAllMetrics()
	assert.Len(t, metrics, 3)

	// Verify metric names
	names := make(map[string]bool)
	for _, metric := range metrics {
		names[metric.Name] = true
	}
	assert.Contains(t, names, "counter1")
	assert.Contains(t, names, "gauge1")
	assert.Contains(t, names, "hist1")
}

func TestMetricsCollectorReset(t *testing.T) {
	collector := NewMetricsCollector()
	ctx := context.Background()

	// Add some metrics
	collector.Counter(ctx, "test_counter", "Test counter", 5, nil)
	collector.Gauge(ctx, "test_gauge", "Test gauge", 10.0, nil)

	metrics := collector.GetAllMetrics()
	assert.Len(t, metrics, 2)

	// Reset
	collector.Reset()

	metrics = collector.GetAllMetrics()
	assert.Len(t, metrics, 0)
}

func TestMetricsCollectorConcurrency(t *testing.T) {
	collector := NewMetricsCollector()
	ctx := context.Background()

	var wg sync.WaitGroup
	numGoroutines := 10
	numOperations := 100

	// Start multiple goroutines updating the same metric
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				collector.Counter(ctx, "concurrent_counter", "Concurrent counter", 1, map[string]string{
					"goroutine": string(rune(id)),
				})
			}
		}(i)
	}

	wg.Wait()

	// Verify total count
	totalCount := 0.0
	for i := 0; i < numGoroutines; i++ {
		metric, exists := collector.GetMetric("concurrent_counter", map[string]string{
			"goroutine": string(rune(i)),
		})
		if exists {
			totalCount += metric.Value
		}
	}

	assert.Equal(t, float64(numGoroutines*numOperations), totalCount)
}

func TestMetricsCollectorMetricKey(t *testing.T) {
	collector := NewMetricsCollector()

	t.Run("metric key without labels", func(t *testing.T) {
		key := collector.metricKey("test_metric", nil)
		assert.Equal(t, "test_metric", key)
	})

	t.Run("metric key with labels", func(t *testing.T) {
		labels := map[string]string{
			"service": "test",
			"version": "1.0",
		}
		key := collector.metricKey("test_metric", labels)
		assert.Contains(t, key, "test_metric")
		assert.Contains(t, key, "service=test")
		assert.Contains(t, key, "version=1.0")
		assert.Contains(t, key, "{")
		assert.Contains(t, key, "}")
	})
}

func TestTimerStop(t *testing.T) {
	collector := NewMetricsCollector()
	ctx := context.Background()

	timer := &Timer{
		collector: collector,
		name:      "test_timer",
		start:     time.Now().Add(-50 * time.Millisecond), // Simulate 50ms elapsed
		labels:    map[string]string{"test": "value"},
	}

	timer.Stop(ctx, "Test timer stopped")

	metric, exists := collector.GetMetric("test_timer", map[string]string{"test": "value"})
	assert.True(t, exists)
	assert.True(t, metric.Value >= 0.05) // At least 50ms
	assert.True(t, metric.Value < 1.0)   // Less than 1 second
}

func TestStatisticalMetrics(t *testing.T) {
	statMetrics := NewStatisticalMetrics("test_stats", "Test statistical metrics")

	t.Run("initial state", func(t *testing.T) {
		assert.Equal(t, "test_stats", statMetrics.Name)
		assert.Equal(t, "Test statistical metrics", statMetrics.Description)
		assert.Len(t, statMetrics.Observations, 0)
		assert.Equal(t, 0, statMetrics.Count())
	})

	t.Run("observe values", func(t *testing.T) {
		statMetrics.Observe(1.0)
		statMetrics.Observe(2.0)
		statMetrics.Observe(3.0)

		assert.Equal(t, 3, statMetrics.Count())
		assert.Equal(t, 2.0, statMetrics.Mean())
		assert.Equal(t, 1.0, statMetrics.Min())
		assert.Equal(t, 3.0, statMetrics.Max())
	})

	t.Run("clear", func(t *testing.T) {
		statMetrics.Clear()
		assert.Equal(t, 0, statMetrics.Count())
		assert.Equal(t, 0.0, statMetrics.Mean())
		assert.Equal(t, 0.0, statMetrics.Min())
		assert.Equal(t, 0.0, statMetrics.Max())
	})
}

func TestStatisticalMetricsEdgeCases(t *testing.T) {
	statMetrics := NewStatisticalMetrics("edge_test", "Edge case test")

	t.Run("empty statistics", func(t *testing.T) {
		assert.Equal(t, 0, statMetrics.Count())
		assert.Equal(t, 0.0, statMetrics.Mean())
		assert.Equal(t, 0.0, statMetrics.Min())
		assert.Equal(t, 0.0, statMetrics.Max())
	})

	t.Run("single observation", func(t *testing.T) {
		statMetrics.Observe(42.0)
		assert.Equal(t, 1, statMetrics.Count())
		assert.Equal(t, 42.0, statMetrics.Mean())
		assert.Equal(t, 42.0, statMetrics.Min())
		assert.Equal(t, 42.0, statMetrics.Max())
	})
}

func TestStatisticalMetricsConcurrency(t *testing.T) {
	statMetrics := NewStatisticalMetrics("concurrent_test", "Concurrent test")

	var wg sync.WaitGroup
	numGoroutines := 5
	numObservations := 20

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(base int) {
			defer wg.Done()
			for j := 0; j < numObservations; j++ {
				statMetrics.Observe(float64(base*numObservations + j))
			}
		}(i)
	}

	wg.Wait()

	expectedCount := numGoroutines * numObservations
	assert.Equal(t, expectedCount, statMetrics.Count())

	// Verify all observations were recorded
	sum := 0.0
	for i := 0; i < expectedCount; i++ {
		sum += float64(i)
	}
	expectedMean := sum / float64(expectedCount)
	assert.Equal(t, expectedMean, statMetrics.Mean())
}

func TestSimpleHealthChecker(t *testing.T) {
	checker := NewSimpleHealthChecker()

	t.Run("initial state", func(t *testing.T) {
		results := checker.RunChecks(context.Background())
		assert.Len(t, results, 0)

		healthy := checker.IsHealthy(context.Background())
		assert.True(t, healthy)
	})

	t.Run("register and run checks", func(t *testing.T) {
		err := checker.RegisterCheck("test_check", func(ctx context.Context) iface.HealthCheckResult {
			return iface.HealthCheckResult{
				Status:    iface.StatusHealthy,
				Message:   "Test check passed",
				CheckName: "test_check",
				Timestamp: time.Now(),
			}
		})
		assert.NoError(t, err)

		results := checker.RunChecks(context.Background())
		assert.Len(t, results, 1)
		assert.Contains(t, results, "test_check")
		assert.Equal(t, iface.StatusHealthy, results["test_check"].Status)

		healthy := checker.IsHealthy(context.Background())
		assert.True(t, healthy)
	})

	t.Run("unhealthy check", func(t *testing.T) {
		err := checker.RegisterCheck("unhealthy_check", func(ctx context.Context) iface.HealthCheckResult {
			return iface.HealthCheckResult{
				Status:    iface.StatusUnhealthy,
				Message:   "Test check failed",
				CheckName: "unhealthy_check",
				Timestamp: time.Now(),
			}
		})
		assert.NoError(t, err)

		results := checker.RunChecks(context.Background())
		assert.Len(t, results, 2)

		healthy := checker.IsHealthy(context.Background())
		assert.False(t, healthy)
	})
}

// Benchmark tests
func BenchmarkMetricsCollector_Counter(b *testing.B) {
	collector := NewMetricsCollector()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		collector.Counter(ctx, "bench_counter", "Benchmark counter", 1, map[string]string{
			"iteration": string(rune(i % 10)), // Limited labels for realistic scenario
		})
	}
}

func BenchmarkMetricsCollector_Gauge(b *testing.B) {
	collector := NewMetricsCollector()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		collector.Gauge(ctx, "bench_gauge", "Benchmark gauge", float64(i), nil)
	}
}

func BenchmarkMetricsCollector_Histogram(b *testing.B) {
	collector := NewMetricsCollector()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		collector.Histogram(ctx, "bench_histogram", "Benchmark histogram", float64(i)*0.001, nil)
	}
}

func BenchmarkMetricsCollector_StartTimer(b *testing.B) {
	collector := NewMetricsCollector()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		timer := collector.StartTimer(ctx, "bench_timer", nil)
		timer.Stop(ctx, "Benchmark timer")
	}
}

func BenchmarkStatisticalMetrics_Observe(b *testing.B) {
	statMetrics := NewStatisticalMetrics("bench_stats", "Benchmark stats")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		statMetrics.Observe(float64(i))
	}
}

func BenchmarkStatisticalMetrics_Mean(b *testing.B) {
	statMetrics := NewStatisticalMetrics("bench_stats", "Benchmark stats")

	// Pre-populate with data
	for i := 0; i < 1000; i++ {
		statMetrics.Observe(float64(i))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = statMetrics.Mean()
	}
}
