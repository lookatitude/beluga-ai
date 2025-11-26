// Package monitoring provides advanced test utilities and comprehensive mocks for testing monitoring implementations.
// This file contains utilities designed to support both unit tests and integration tests.
package monitoring

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// AdvancedMockMonitor provides a comprehensive mock implementation for testing.
type AdvancedMockMonitor struct {
	lastHealthCheck time.Time
	errorToReturn   error
	metrics         map[string]any
	mock.Mock
	name          string
	monitorType   string
	healthState   string
	traces        []TraceRecord
	logs          []LogRecord
	healthChecks  []HealthRecord
	simulateDelay time.Duration
	callCount     int
	mu            sync.RWMutex
	shouldError   bool
}

// TraceRecord represents a trace record for testing.
type TraceRecord struct {
	Timestamp time.Time
	TraceID   string
	SpanID    string
	Operation string
	Duration  time.Duration
	Success   bool
}

// LogRecord represents a log record for testing.
type LogRecord struct {
	Timestamp time.Time
	Fields    map[string]any
	Level     string
	Message   string
}

// HealthRecord represents a health check record for testing.
type HealthRecord struct {
	Timestamp time.Time
	Details   map[string]any
	Component string
	Status    string
}

// NewAdvancedMockMonitor creates a new advanced mock monitor.
func NewAdvancedMockMonitor(name, monitorType string, options ...MockMonitorOption) *AdvancedMockMonitor {
	mock := &AdvancedMockMonitor{
		name:         name,
		monitorType:  monitorType,
		metrics:      make(map[string]any),
		traces:       make([]TraceRecord, 0),
		logs:         make([]LogRecord, 0),
		healthChecks: make([]HealthRecord, 0),
		healthState:  "healthy",
	}

	// Apply options
	for _, opt := range options {
		opt(mock)
	}

	return mock
}

// MockMonitorOption defines functional options for mock configuration.
type MockMonitorOption func(*AdvancedMockMonitor)

// WithMockError configures the mock to return errors.
func WithMockError(shouldError bool, err error) MockMonitorOption {
	return func(m *AdvancedMockMonitor) {
		m.shouldError = shouldError
		m.errorToReturn = err
	}
}

// WithMockDelay adds artificial delay to mock operations.
func WithMockDelay(delay time.Duration) MockMonitorOption {
	return func(m *AdvancedMockMonitor) {
		m.simulateDelay = delay
	}
}

// Mock implementation methods.
func (m *AdvancedMockMonitor) RecordMetric(ctx context.Context, name string, value any, labels map[string]string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.callCount++

	if m.simulateDelay > 0 {
		time.Sleep(m.simulateDelay)
	}

	if m.shouldError {
		return m.errorToReturn
	}

	// Store metric
	metricKey := fmt.Sprintf("%s_%d", name, len(m.metrics))
	m.metrics[metricKey] = map[string]any{
		"name":      name,
		"value":     value,
		"labels":    labels,
		"timestamp": time.Now(),
	}

	return nil
}

func (m *AdvancedMockMonitor) StartTrace(ctx context.Context, operation string) (context.Context, string, error) {
	m.mu.Lock()
	m.callCount++
	m.mu.Unlock()

	if m.shouldError {
		return ctx, "", m.errorToReturn
	}

	traceID := fmt.Sprintf("trace_%s_%d", operation, time.Now().UnixNano())
	spanID := fmt.Sprintf("span_%d", len(m.traces)+1)

	// Record trace start
	m.mu.Lock()
	m.traces = append(m.traces, TraceRecord{
		TraceID:   traceID,
		SpanID:    spanID,
		Operation: operation,
		Timestamp: time.Now(),
		Success:   true,
	})
	m.mu.Unlock()

	return ctx, traceID, nil
}

func (m *AdvancedMockMonitor) FinishTrace(ctx context.Context, traceID string, success bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.callCount++

	if m.shouldError {
		return m.errorToReturn
	}

	// Find and update trace
	for i, trace := range m.traces {
		if trace.TraceID == traceID {
			m.traces[i].Duration = time.Since(trace.Timestamp)
			m.traces[i].Success = success
			break
		}
	}

	return nil
}

func (m *AdvancedMockMonitor) Log(ctx context.Context, level, message string, fields map[string]any) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.callCount++

	if m.simulateDelay > 0 {
		time.Sleep(m.simulateDelay)
	}

	if m.shouldError {
		return m.errorToReturn
	}

	// Store log
	logRecord := LogRecord{
		Level:     level,
		Message:   message,
		Fields:    make(map[string]any),
		Timestamp: time.Now(),
	}

	// Copy fields
	for k, v := range fields {
		logRecord.Fields[k] = v
	}

	m.logs = append(m.logs, logRecord)
	return nil
}

func (m *AdvancedMockMonitor) CheckComponentHealth(ctx context.Context, component string) (map[string]any, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.callCount++

	if m.shouldError {
		return nil, m.errorToReturn
	}

	healthStatus := map[string]any{
		"component": component,
		"status":    "healthy",
		"timestamp": time.Now(),
		"details": map[string]any{
			"uptime":  "10m",
			"version": "1.0.0",
		},
	}

	// Record health check
	m.healthChecks = append(m.healthChecks, HealthRecord{
		Component: component,
		Status:    "healthy",
		Details:   healthStatus["details"].(map[string]any),
		Timestamp: time.Now(),
	})

	return healthStatus, nil
}

// Helper methods for testing.
func (m *AdvancedMockMonitor) GetName() string {
	return m.name
}

func (m *AdvancedMockMonitor) GetMonitorType() string {
	return m.monitorType
}

func (m *AdvancedMockMonitor) GetCallCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.callCount
}

func (m *AdvancedMockMonitor) GetMetrics() map[string]any {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make(map[string]any)
	for k, v := range m.metrics {
		result[k] = v
	}
	return result
}

func (m *AdvancedMockMonitor) GetTraces() []TraceRecord {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]TraceRecord, len(m.traces))
	copy(result, m.traces)
	return result
}

func (m *AdvancedMockMonitor) GetLogs() []LogRecord {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]LogRecord, len(m.logs))
	copy(result, m.logs)
	return result
}

func (m *AdvancedMockMonitor) GetHealthChecks() []HealthRecord {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]HealthRecord, len(m.healthChecks))
	copy(result, m.healthChecks)
	return result
}

func (m *AdvancedMockMonitor) CheckHealth() map[string]any {
	m.lastHealthCheck = time.Now()
	return map[string]any{
		"status":              m.healthState,
		"name":                m.name,
		"type":                m.monitorType,
		"call_count":          m.callCount,
		"metrics_count":       len(m.metrics),
		"traces_count":        len(m.traces),
		"logs_count":          len(m.logs),
		"health_checks_count": len(m.healthChecks),
		"last_checked":        m.lastHealthCheck,
	}
}

// Test data creation helpers

// CreateTestMetrics creates test metrics for monitoring.
func CreateTestMetrics(count int) map[string]any {
	metrics := make(map[string]any)

	for i := 0; i < count; i++ {
		key := fmt.Sprintf("test_metric_%d", i+1)
		metrics[key] = map[string]any{
			"value":     float64(i * 100),
			"timestamp": time.Now(),
			"labels": map[string]string{
				"component": "test",
				"operation": fmt.Sprintf("op_%d", i+1),
			},
		}
	}

	return metrics
}

// CreateTestTraces creates test trace records.
func CreateTestTraces(count int) []TraceRecord {
	traces := make([]TraceRecord, count)

	for i := 0; i < count; i++ {
		traces[i] = TraceRecord{
			TraceID:   fmt.Sprintf("trace_%d", i+1),
			SpanID:    fmt.Sprintf("span_%d", i+1),
			Operation: fmt.Sprintf("test_operation_%d", i+1),
			Duration:  time.Duration(i+1) * 100 * time.Millisecond,
			Timestamp: time.Now(),
			Success:   i%4 != 0, // 75% success rate
		}
	}

	return traces
}

// CreateTestLogs creates test log records.
func CreateTestLogs(count int) []LogRecord {
	logs := make([]LogRecord, count)
	levels := []string{"DEBUG", "INFO", "WARN", "ERROR"}

	for i := 0; i < count; i++ {
		level := levels[i%len(levels)]
		logs[i] = LogRecord{
			Level:   level,
			Message: fmt.Sprintf("Test log message %d at %s level", i+1, level),
			Fields: map[string]any{
				"component":  "test",
				"request_id": fmt.Sprintf("req_%d", i+1),
				"user_id":    fmt.Sprintf("user_%d", (i%10)+1),
			},
			Timestamp: time.Now(),
		}
	}

	return logs
}

// Assertion helpers

// AssertMonitoringData validates monitoring data collection.
func AssertMonitoringData(t *testing.T, monitor *AdvancedMockMonitor, expectedMinMetrics, expectedMinTraces, expectedMinLogs int) {
	metrics := monitor.GetMetrics()
	traces := monitor.GetTraces()
	logs := monitor.GetLogs()

	assert.GreaterOrEqual(t, len(metrics), expectedMinMetrics, "Should have minimum metrics")
	assert.GreaterOrEqual(t, len(traces), expectedMinTraces, "Should have minimum traces")
	assert.GreaterOrEqual(t, len(logs), expectedMinLogs, "Should have minimum logs")
}

// AssertTraceRecord validates trace record properties.
func AssertTraceRecord(t *testing.T, trace TraceRecord, expectedOperation string) {
	assert.NotEmpty(t, trace.TraceID, "Trace should have ID")
	assert.NotEmpty(t, trace.SpanID, "Trace should have span ID")
	assert.Equal(t, expectedOperation, trace.Operation, "Trace operation should match")
	assert.Greater(t, trace.Duration, time.Duration(0), "Trace should have positive duration")
	assert.False(t, trace.Timestamp.IsZero(), "Trace should have timestamp")
}

// AssertLogRecord validates log record properties.
func AssertLogRecord(t *testing.T, log LogRecord, expectedLevel string) {
	assert.Equal(t, expectedLevel, log.Level, "Log level should match")
	assert.NotEmpty(t, log.Message, "Log should have message")
	assert.NotNil(t, log.Fields, "Log should have fields (can be empty)")
	assert.False(t, log.Timestamp.IsZero(), "Log should have timestamp")
}

// AssertMonitorHealth validates monitor health check results.
func AssertMonitorHealth(t *testing.T, health map[string]any, expectedStatus string) {
	assert.Contains(t, health, "status")
	assert.Equal(t, expectedStatus, health["status"])
	assert.Contains(t, health, "name")
	assert.Contains(t, health, "type")
	assert.Contains(t, health, "call_count")
}

// Performance testing helpers

// RunLoadTest executes a load test scenario on monitor.
func RunLoadTest(t *testing.T, monitor *AdvancedMockMonitor, numOperations, concurrency int) {
	t.Helper()
	var wg sync.WaitGroup
	errChan := make(chan error, numOperations)

	semaphore := make(chan struct{}, concurrency)
	ctx := context.Background()

	for i := 0; i < numOperations; i++ {
		wg.Add(1)
		go func(opID int) {
			defer wg.Done()

			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			switch opID % 4 {
			case 0:
				// Test metric recording
				err := monitor.RecordMetric(ctx, fmt.Sprintf("test_metric_%d", opID),
					float64(opID), map[string]string{"operation": "load_test"})
				if err != nil {
					errChan <- err
				}
			case 1:
				// Test trace recording
				_, traceID, err := monitor.StartTrace(ctx, fmt.Sprintf("load_operation_%d", opID))
				if err != nil {
					errChan <- err
					return
				}
				err = monitor.FinishTrace(ctx, traceID, true)
				if err != nil {
					errChan <- err
				}
			case 2:
				// Test logging
				err := monitor.Log(ctx, "INFO", fmt.Sprintf("Load test log %d", opID),
					map[string]any{"operation": "load_test"})
				if err != nil {
					errChan <- err
				}
			default:
				// Test health check
				_, err := monitor.CheckComponentHealth(ctx, fmt.Sprintf("component_%d", opID))
				if err != nil {
					errChan <- err
				}
			}
		}(i)
	}

	wg.Wait()
	close(errChan)

	// Verify no errors occurred
	for err := range errChan {
		require.NoError(t, err)
	}

	// Verify expected call count
	assert.Equal(t, numOperations, monitor.GetCallCount())
}

// Integration test helpers

// IntegrationTestHelper provides utilities for integration testing.
type IntegrationTestHelper struct {
	monitors map[string]*AdvancedMockMonitor
}

func NewIntegrationTestHelper() *IntegrationTestHelper {
	return &IntegrationTestHelper{
		monitors: make(map[string]*AdvancedMockMonitor),
	}
}

func (h *IntegrationTestHelper) AddMonitor(name string, monitor *AdvancedMockMonitor) {
	h.monitors[name] = monitor
}

func (h *IntegrationTestHelper) GetMonitor(name string) *AdvancedMockMonitor {
	return h.monitors[name]
}

func (h *IntegrationTestHelper) Reset() {
	for _, monitor := range h.monitors {
		monitor.callCount = 0
		monitor.metrics = make(map[string]any)
		monitor.traces = make([]TraceRecord, 0)
		monitor.logs = make([]LogRecord, 0)
		monitor.healthChecks = make([]HealthRecord, 0)
	}
}

// MonitoringScenarioRunner runs common monitoring scenarios.
type MonitoringScenarioRunner struct {
	monitor *AdvancedMockMonitor
}

func NewMonitoringScenarioRunner(monitor *AdvancedMockMonitor) *MonitoringScenarioRunner {
	return &MonitoringScenarioRunner{
		monitor: monitor,
	}
}

func (r *MonitoringScenarioRunner) RunFullObservabilityScenario(ctx context.Context, operations []string) error {
	for i, operation := range operations {
		// Start trace
		traceCtx, traceID, err := r.monitor.StartTrace(ctx, operation)
		if err != nil {
			return fmt.Errorf("failed to start trace for operation %d: %w", i+1, err)
		}

		// Log operation start
		err = r.monitor.Log(traceCtx, "INFO", "Starting "+operation,
			map[string]any{"trace_id": traceID, "operation": operation})
		if err != nil {
			return fmt.Errorf("failed to log operation start %d: %w", i+1, err)
		}

		// Record metrics
		err = r.monitor.RecordMetric(traceCtx, "operation_count", 1,
			map[string]string{"operation": operation})
		if err != nil {
			return fmt.Errorf("failed to record metrics for operation %d: %w", i+1, err)
		}

		// Simulate work with health check
		healthResult, err := r.monitor.CheckComponentHealth(traceCtx, operation)
		if err != nil {
			return fmt.Errorf("failed health check for operation %d: %w", i+1, err)
		}

		// Finish trace
		success := healthResult != nil
		err = r.monitor.FinishTrace(traceCtx, traceID, success)
		if err != nil {
			return fmt.Errorf("failed to finish trace for operation %d: %w", i+1, err)
		}

		// Log operation completion
		err = r.monitor.Log(traceCtx, "INFO", "Completed "+operation,
			map[string]any{"trace_id": traceID, "success": success})
		if err != nil {
			return fmt.Errorf("failed to log operation completion %d: %w", i+1, err)
		}
	}

	return nil
}

// BenchmarkHelper provides benchmarking utilities for monitoring.
type BenchmarkHelper struct {
	monitor *AdvancedMockMonitor
}

func NewBenchmarkHelper(monitor *AdvancedMockMonitor) *BenchmarkHelper {
	return &BenchmarkHelper{
		monitor: monitor,
	}
}

func (b *BenchmarkHelper) BenchmarkMetricRecording(iterations int) (time.Duration, error) {
	ctx := context.Background()

	start := time.Now()
	for i := 0; i < iterations; i++ {
		err := b.monitor.RecordMetric(ctx, "benchmark_metric", float64(i),
			map[string]string{"benchmark": "true"})
		if err != nil {
			return 0, err
		}
	}

	return time.Since(start), nil
}

func (b *BenchmarkHelper) BenchmarkTracing(iterations int) (time.Duration, error) {
	ctx := context.Background()

	start := time.Now()
	for i := 0; i < iterations; i++ {
		_, traceID, err := b.monitor.StartTrace(ctx, "benchmark_operation")
		if err != nil {
			return 0, err
		}

		err = b.monitor.FinishTrace(ctx, traceID, true)
		if err != nil {
			return 0, err
		}
	}

	return time.Since(start), nil
}

func (b *BenchmarkHelper) BenchmarkLogging(iterations int) (time.Duration, error) {
	ctx := context.Background()

	start := time.Now()
	for i := 0; i < iterations; i++ {
		err := b.monitor.Log(ctx, "INFO", fmt.Sprintf("Benchmark log %d", i),
			map[string]any{"iteration": i})
		if err != nil {
			return 0, err
		}
	}

	return time.Since(start), nil
}
