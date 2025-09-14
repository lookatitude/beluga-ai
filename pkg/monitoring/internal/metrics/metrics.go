// Package metrics provides metrics collection and reporting implementations
package metrics

import (
	"context"
	"sync"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/monitoring/iface"
)

// MetricsCollector provides metrics collection and reporting
type MetricsCollector struct {
	mutex   sync.RWMutex
	metrics map[string]*Metric
}

// Metric represents a single metric
type Metric struct {
	Name        string
	Description string
	Value       float64
	Labels      map[string]string
	Type        MetricType
	Timestamp   time.Time
}

// MetricType represents the type of metric
type MetricType int

const (
	Counter MetricType = iota
	Gauge
	Histogram
	Summary
)

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		metrics: make(map[string]*Metric),
	}
}

// Counter creates or increments a counter metric
func (mc *MetricsCollector) Counter(ctx context.Context, name, description string, value float64, labels map[string]string) {
	mc.updateMetric(name, description, value, labels, Counter, func(m *Metric) {
		m.Value += value
	})
}

// Gauge sets a gauge metric value
func (mc *MetricsCollector) Gauge(ctx context.Context, name, description string, value float64, labels map[string]string) {
	mc.updateMetric(name, description, value, labels, Gauge, func(m *Metric) {
		m.Value = value
	})
}

// Histogram records a histogram observation
func (mc *MetricsCollector) Histogram(ctx context.Context, name, description string, value float64, labels map[string]string) {
	mc.updateMetric(name, description, value, labels, Histogram, func(m *Metric) {
		// For histogram, we could implement buckets here
		// For now, just store the latest value
		m.Value = value
	})
}

// Timing records the duration of an operation
func (mc *MetricsCollector) Timing(ctx context.Context, name, description string, duration time.Duration, labels map[string]string) {
	mc.Histogram(ctx, name, description, duration.Seconds(), labels)
}

// Increment increments a counter by 1
func (mc *MetricsCollector) Increment(ctx context.Context, name, description string, labels map[string]string) {
	mc.Counter(ctx, name, description, 1, labels)
}

// updateMetric updates or creates a metric
func (mc *MetricsCollector) updateMetric(name, description string, value float64, labels map[string]string, typ MetricType, updater func(*Metric)) {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	key := mc.metricKey(name, labels)
	metric, exists := mc.metrics[key]

	if !exists {
		metric = &Metric{
			Name:        name,
			Description: description,
			Value:       0,
			Labels:      labels,
			Type:        typ,
		}
		mc.metrics[key] = metric
	}

	updater(metric)
	metric.Timestamp = time.Now()
}

// metricKey generates a unique key for a metric
func (mc *MetricsCollector) metricKey(name string, labels map[string]string) string {
	key := name
	if len(labels) > 0 {
		key += "{"
		for k, v := range labels {
			key += k + "=" + v + ","
		}
		key = key[:len(key)-1] + "}"
	}
	return key
}

// GetMetric retrieves a metric by name and labels
func (mc *MetricsCollector) GetMetric(name string, labels map[string]string) (*Metric, bool) {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()

	key := mc.metricKey(name, labels)
	metric, exists := mc.metrics[key]
	return metric, exists
}

// GetAllMetrics returns all collected metrics
func (mc *MetricsCollector) GetAllMetrics() []*Metric {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()

	metrics := make([]*Metric, 0, len(mc.metrics))
	for _, metric := range mc.metrics {
		metrics = append(metrics, metric)
	}
	return metrics
}

// Reset clears all metrics
func (mc *MetricsCollector) Reset() {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()
	mc.metrics = make(map[string]*Metric)
}

// Timer provides a convenient way to time operations
type Timer struct {
	collector *MetricsCollector
	name      string
	start     time.Time
	labels    map[string]string
}

// StartTimer starts a timer for measuring operation duration
func (mc *MetricsCollector) StartTimer(ctx context.Context, name string, labels map[string]string) iface.Timer {
	return &Timer{
		collector: mc,
		name:      name,
		start:     time.Now(),
		labels:    labels,
	}
}

// Stop stops the timer and records the duration
func (t *Timer) Stop(ctx context.Context, description string) {
	duration := time.Since(t.start)
	t.collector.Timing(ctx, t.name, description, duration, t.labels)
}

// Observation represents a single observation for statistical metrics
type Observation struct {
	Value     float64
	Timestamp time.Time
}

// StatisticalMetrics provides statistical calculations
type StatisticalMetrics struct {
	Name         string
	Description  string
	Observations []Observation
	mutex        sync.RWMutex
}

// NewStatisticalMetrics creates a new statistical metrics collector
func NewStatisticalMetrics(name, description string) *StatisticalMetrics {
	return &StatisticalMetrics{
		Name:         name,
		Description:  description,
		Observations: make([]Observation, 0),
	}
}

// Observe adds a new observation
func (sm *StatisticalMetrics) Observe(value float64) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	sm.Observations = append(sm.Observations, Observation{
		Value:     value,
		Timestamp: time.Now(),
	})
}

// Count returns the number of observations
func (sm *StatisticalMetrics) Count() int {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()
	return len(sm.Observations)
}

// Mean calculates the arithmetic mean
func (sm *StatisticalMetrics) Mean() float64 {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	if len(sm.Observations) == 0 {
		return 0
	}

	sum := 0.0
	for _, obs := range sm.Observations {
		sum += obs.Value
	}
	return sum / float64(len(sm.Observations))
}

// Min returns the minimum value
func (sm *StatisticalMetrics) Min() float64 {
	sm.mutex.RLock()
	defer sm.mutex.RLock()

	if len(sm.Observations) == 0 {
		return 0
	}

	min := sm.Observations[0].Value
	for _, obs := range sm.Observations[1:] {
		if obs.Value < min {
			min = obs.Value
		}
	}
	return min
}

// Max returns the maximum value
func (sm *StatisticalMetrics) Max() float64 {
	sm.mutex.RLock()
	defer sm.mutex.RLock()

	if len(sm.Observations) == 0 {
		return 0
	}

	max := sm.Observations[0].Value
	for _, obs := range sm.Observations[1:] {
		if obs.Value > max {
			max = obs.Value
		}
	}
	return max
}

// Clear removes all observations
func (sm *StatisticalMetrics) Clear() {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	sm.Observations = sm.Observations[:0]
}

// SimpleHealthChecker provides a simple health check wrapper
type SimpleHealthChecker struct {
	checks map[string]iface.HealthCheckFunc
	mutex  sync.RWMutex
}

// NewSimpleHealthChecker creates a new simple health checker
func NewSimpleHealthChecker() *SimpleHealthChecker {
	return &SimpleHealthChecker{
		checks: make(map[string]iface.HealthCheckFunc),
	}
}

// RegisterCheck registers a health check function
func (shc *SimpleHealthChecker) RegisterCheck(name string, check iface.HealthCheckFunc) error {
	shc.mutex.Lock()
	defer shc.mutex.Unlock()
	shc.checks[name] = check
	return nil
}

// RunChecks runs all registered health checks
func (shc *SimpleHealthChecker) RunChecks(ctx context.Context) map[string]iface.HealthCheckResult {
	shc.mutex.RLock()
	defer shc.mutex.RUnlock()

	results := make(map[string]iface.HealthCheckResult)

	for name, check := range shc.checks {
		select {
		case <-ctx.Done():
			results[name] = iface.HealthCheckResult{
				Status:    iface.StatusUnhealthy,
				Message:   "Health check timed out",
				Timestamp: time.Now(),
				CheckName: name,
			}
		default:
			results[name] = check(ctx)
		}
	}

	return results
}

// IsHealthy returns true if all health checks pass
func (shc *SimpleHealthChecker) IsHealthy(ctx context.Context) bool {
	results := shc.RunChecks(ctx)
	for _, result := range results {
		if result.Status != iface.StatusHealthy {
			return false
		}
	}
	return true
}
