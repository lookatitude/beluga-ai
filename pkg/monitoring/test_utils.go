// Package monitoring provides advanced test utilities and comprehensive mocks for testing monitoring implementations.
// This file contains utilities designed to support both unit tests and integration tests.
package monitoring

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/monitoring/iface"
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
	m.callCount++
	shouldError := m.shouldError
	errorToReturn := m.errorToReturn
	simulateDelay := m.simulateDelay
	m.mu.Unlock()

	if simulateDelay > 0 {
		time.Sleep(simulateDelay)
	}

	if shouldError {
		return errorToReturn
	}

	// Store metric
	m.mu.Lock()
	metricKey := fmt.Sprintf("%s_%d", name, len(m.metrics))
	m.metrics[metricKey] = map[string]any{
		"name":      name,
		"value":     value,
		"labels":    labels,
		"timestamp": time.Now(),
	}
	m.mu.Unlock()

	return nil
}

func (m *AdvancedMockMonitor) StartTrace(ctx context.Context, operation string) (context.Context, string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.callCount++

	if m.shouldError {
		return ctx, "", m.errorToReturn
	}

	traceID := fmt.Sprintf("trace_%s_%d", operation, time.Now().UnixNano())
	spanID := fmt.Sprintf("span_%d", len(m.traces)+1)

	// Record trace start
	m.traces = append(m.traces, TraceRecord{
		TraceID:   traceID,
		SpanID:    spanID,
		Operation: operation,
		Timestamp: time.Now(),
		Success:   true,
	})

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

// Monitor interface implementation methods
func (m *AdvancedMockMonitor) Logger() iface.Logger {
	return &MockLogger{}
}

func (m *AdvancedMockMonitor) Tracer() iface.Tracer {
	return &MockTracer{}
}

func (m *AdvancedMockMonitor) Metrics() iface.MetricsCollector {
	return &MockMetricsCollector{}
}

func (m *AdvancedMockMonitor) HealthChecker() iface.HealthChecker {
	return &MockHealthChecker{}
}

func (m *AdvancedMockMonitor) SafetyChecker() iface.SafetyChecker {
	return &MockSafetyChecker{}
}

func (m *AdvancedMockMonitor) EthicalChecker() iface.EthicalChecker {
	return &MockEthicalChecker{}
}

func (m *AdvancedMockMonitor) BestPracticesChecker() iface.BestPracticesChecker {
	return &MockBestPracticesChecker{}
}

func (m *AdvancedMockMonitor) Start(ctx context.Context) error {
	return nil
}

func (m *AdvancedMockMonitor) Stop(ctx context.Context) error {
	return nil
}

func (m *AdvancedMockMonitor) IsHealthy(ctx context.Context) bool {
	return m.healthState == "healthy"
}

// Mock implementations for interface methods
type MockLogger struct{}

func (m *MockLogger) Debug(ctx context.Context, message string, fields ...map[string]any)   {}
func (m *MockLogger) Info(ctx context.Context, message string, fields ...map[string]any)    {}
func (m *MockLogger) Warning(ctx context.Context, message string, fields ...map[string]any) {}
func (m *MockLogger) Error(ctx context.Context, message string, fields ...map[string]any)   {}
func (m *MockLogger) Fatal(ctx context.Context, message string, fields ...map[string]any)   {}
func (m *MockLogger) WithFields(fields map[string]any) iface.ContextLogger {
	return &MockContextLogger{}
}

type MockContextLogger struct{}

func (m *MockContextLogger) Debug(ctx context.Context, message string, fields ...map[string]any) {}
func (m *MockContextLogger) Info(ctx context.Context, message string, fields ...map[string]any)  {}
func (m *MockContextLogger) Error(ctx context.Context, message string, fields ...map[string]any) {}

type MockTracer struct{}

func (m *MockTracer) StartSpan(ctx context.Context, name string, opts ...iface.SpanOption) (context.Context, iface.Span) {
	return ctx, &MockSpan{}
}
func (m *MockTracer) FinishSpan(span iface.Span)                {}
func (m *MockTracer) GetSpan(spanID string) (iface.Span, bool)  { return nil, false }
func (m *MockTracer) GetTraceSpans(traceID string) []iface.Span { return nil }

type MockSpan struct{}

func (m *MockSpan) Log(message string, fields ...map[string]any) {}
func (m *MockSpan) SetError(err error)                           {}
func (m *MockSpan) SetStatus(status string)                      {}
func (m *MockSpan) GetDuration() time.Duration                   { return 0 }
func (m *MockSpan) IsFinished() bool                             { return false }
func (m *MockSpan) SetTag(key string, value any)                 {}

type MockMetricsCollector struct{}

func (m *MockMetricsCollector) Counter(ctx context.Context, name, description string, value float64, labels map[string]string) {
}
func (m *MockMetricsCollector) Gauge(ctx context.Context, name, description string, value float64, labels map[string]string) {
}
func (m *MockMetricsCollector) Histogram(ctx context.Context, name, description string, value float64, labels map[string]string) {
}
func (m *MockMetricsCollector) Timing(ctx context.Context, name, description string, duration time.Duration, labels map[string]string) {
}
func (m *MockMetricsCollector) Increment(ctx context.Context, name, description string, labels map[string]string) {
}
func (m *MockMetricsCollector) StartTimer(ctx context.Context, name string, labels map[string]string) iface.Timer {
	return &MockTimer{}
}

type MockTimer struct{}

func (m *MockTimer) Stop(ctx context.Context, description string) {}

type MockHealthChecker struct{}

func (m *MockHealthChecker) RegisterCheck(name string, check iface.HealthCheckFunc) error { return nil }
func (m *MockHealthChecker) RunChecks(ctx context.Context) map[string]iface.HealthCheckResult {
	return nil
}
func (m *MockHealthChecker) IsHealthy(ctx context.Context) bool { return true }

type MockSafetyChecker struct{}

func (m *MockSafetyChecker) CheckContent(ctx context.Context, content, contextInfo string) (iface.SafetyResult, error) {
	return iface.SafetyResult{Safe: true, Issues: []iface.SafetyIssue{}}, nil
}
func (m *MockSafetyChecker) RequestHumanReview(ctx context.Context, content, contextInfo string, riskScore float64) (iface.ReviewDecision, error) {
	return iface.ReviewDecision{Approved: true}, nil
}

type MockEthicalChecker struct{}

func (m *MockEthicalChecker) CheckContent(ctx context.Context, content string, ethicalCtx iface.EthicalContext) (iface.EthicalAnalysis, error) {
	return iface.EthicalAnalysis{}, nil
}

type MockBestPracticesChecker struct{}

func (m *MockBestPracticesChecker) Validate(ctx context.Context, data any, component string) []iface.ValidationIssue {
	return nil
}
func (m *MockBestPracticesChecker) AddValidator(validator iface.Validator) {}

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

	// Small delay to ensure all operations are fully recorded and locks are released
	time.Sleep(50 * time.Millisecond)

	// Verify expected call count
	// Each operation can be 1-2 calls:
	// - Metric recording: 1 call
	// - Trace operations: 2 calls (StartTrace + FinishTrace)
	// - Logging: 1 call
	// - Health check: 1 call
	// With 4 operation types, roughly 25% are trace operations (2 calls), 75% are single calls
	// Expected: ~(numOperations * 0.25 * 2) + (numOperations * 0.75 * 1) = numOperations * 1.25
	callCount := monitor.GetCallCount()
	// Allow some variance - at least numOperations, at most 2*numOperations
	assert.GreaterOrEqual(t, callCount, numOperations, "Expected at least %d calls, got %d", numOperations, callCount)
	assert.LessOrEqual(t, callCount, numOperations*2, "Expected at most %d calls, got %d", numOperations*2, callCount)
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
