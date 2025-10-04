// Package best_practices provides best practices validation implementations
package best_practices

import (
	"context"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/monitoring/iface"
	"github.com/lookatitude/beluga-ai/pkg/monitoring/internal/logger"
	"github.com/lookatitude/beluga-ai/pkg/monitoring/internal/metrics"
)

// BestPracticesChecker provides comprehensive best practices validation
type BestPracticesChecker struct {
	logger     *logger.StructuredLogger
	metrics    *metrics.MetricsCollector
	validators []iface.Validator
	mutex      sync.RWMutex
}

// NewBestPracticesChecker creates a new best practices checker
func NewBestPracticesChecker(logger *logger.StructuredLogger, metrics *metrics.MetricsCollector) *BestPracticesChecker {
	checker := &BestPracticesChecker{
		logger:  logger,
		metrics: metrics,
		validators: []iface.Validator{
			&ConcurrencyValidator{},
			&ErrorHandlingValidator{},
			&ResourceManagementValidator{},
			&SecurityValidator{},
		},
	}

	return checker
}

// Validate checks data against all registered validators
func (bpc *BestPracticesChecker) Validate(ctx context.Context, data interface{}, component string) []iface.ValidationIssue {
	allIssues := make([]iface.ValidationIssue, 0)

	for _, validator := range bpc.validators {
		issues := validator.Validate(ctx, data)
		for _, issue := range issues {
			issue.Validator = validator.Name()
			allIssues = append(allIssues, issue)
		}
	}

	// Record metrics
	bpc.metrics.Counter(ctx, "best_practices_checks_total", "Total best practices checks", 1, map[string]string{
		"component": component,
	})

	if len(allIssues) > 0 {
		bpc.metrics.Counter(ctx, "best_practices_issues_total", "Total best practices issues", float64(len(allIssues)), map[string]string{
			"component": component,
		})

		// Log issues
		bpc.logger.Warning(ctx, "Best practices violations detected",
			map[string]interface{}{
				"component": component,
				"issues":    len(allIssues),
			})
	}

	return allIssues
}

// AddValidator adds a custom validator
func (bpc *BestPracticesChecker) AddValidator(validator iface.Validator) {
	bpc.mutex.Lock()
	defer bpc.mutex.Unlock()
	bpc.validators = append(bpc.validators, validator)
}

// ConcurrencyValidator checks for concurrency best practices
type ConcurrencyValidator struct{}

func (cv *ConcurrencyValidator) Name() string { return "concurrency" }

func (cv *ConcurrencyValidator) Validate(ctx context.Context, data interface{}) []iface.ValidationIssue {
	issues := make([]iface.ValidationIssue, 0)

	// This is a placeholder - in a real implementation, you would analyze
	// goroutine usage, mutex patterns, channel operations, etc.
	// For now, we'll check basic patterns in the data

	if str, ok := data.(string); ok {
		// Check for common concurrency anti-patterns
		if containsUnsafePatterns(str) {
			issues = append(issues, iface.ValidationIssue{
				Issue:      "Potential unsafe concurrency pattern detected",
				Severity:   "medium",
				Suggestion: "Review goroutine usage, mutex patterns, and channel operations",
			})
		}
	}

	return issues
}

// ErrorHandlingValidator checks for proper error handling
type ErrorHandlingValidator struct{}

func (ehv *ErrorHandlingValidator) Name() string { return "error_handling" }

func (ehv *ErrorHandlingValidator) Validate(ctx context.Context, data interface{}) []iface.ValidationIssue {
	issues := make([]iface.ValidationIssue, 0)

	if str, ok := data.(string); ok {
		// Check for common error handling anti-patterns
		if containsErrorHandlingIssues(str) {
			issues = append(issues, iface.ValidationIssue{
				Issue:      "Potential error handling issue detected",
				Severity:   "high",
				Suggestion: "Ensure proper error handling with context, wrapping, and logging",
			})
		}
	}

	return issues
}

// ResourceManagementValidator checks for proper resource management
type ResourceManagementValidator struct{}

func (rmv *ResourceManagementValidator) Name() string { return "resource_management" }

func (rmv *ResourceManagementValidator) Validate(ctx context.Context, data interface{}) []iface.ValidationIssue {
	issues := make([]iface.ValidationIssue, 0)

	// Check for resource leaks, proper cleanup, etc.
	// This is a simplified example

	if str, ok := data.(string); ok {
		if containsResourceIssues(str) {
			issues = append(issues, iface.ValidationIssue{
				Issue:      "Potential resource management issue",
				Severity:   "medium",
				Suggestion: "Ensure proper resource cleanup and defer statements",
			})
		}
	}

	return issues
}

// SecurityValidator checks for security best practices
type SecurityValidator struct{}

func (sv *SecurityValidator) Name() string { return "security" }

func (sv *SecurityValidator) Validate(ctx context.Context, data interface{}) []iface.ValidationIssue {
	issues := make([]iface.ValidationIssue, 0)

	if str, ok := data.(string); ok {
		if containsSecurityIssues(str) {
			issues = append(issues, iface.ValidationIssue{
				Issue:      "Potential security issue detected",
				Severity:   "high",
				Suggestion: "Review input validation, authentication, and authorization",
			})
		}
	}

	return issues
}

// Helper functions for pattern detection
func containsUnsafePatterns(code string) bool {
	patterns := []string{
		"go func()",    // Uncontrolled goroutines
		"defer.*mutex", // Mutex in defer (can cause deadlocks)
		"close.*nil",   // Closing nil channels
	}

	for _, pattern := range patterns {
		if containsPattern(code, pattern) {
			return true
		}
	}
	return false
}

func containsErrorHandlingIssues(code string) bool {
	patterns := []string{
		"err != nil", // Check if error handling is missing
		"panic(",     // Avoid panics in production code
		"log.Fatal",  // Avoid Fatal calls
	}

	// Look for error checks without proper handling
	if containsPattern(code, "err != nil") && !containsPattern(code, "return") {
		return true
	}

	for _, pattern := range patterns {
		if containsPattern(code, pattern) {
			return true // Simplified - in practice, you'd do more sophisticated analysis
		}
	}
	return false
}

func containsResourceIssues(code string) bool {
	patterns := []string{
		"open.*file", // File operations without defer close
		"http.Get",   // HTTP requests without proper cleanup
		"database",   // Database operations
	}

	for _, pattern := range patterns {
		if containsPattern(code, pattern) {
			return true // Simplified check
		}
	}
	return false
}

func containsSecurityIssues(code string) bool {
	patterns := []string{
		"password.*string", // Storing passwords as strings
		"sql.*inject",      // SQL injection risks
		"eval(",            // Code injection
		"exec.Command",     // Command injection
	}

	for _, pattern := range patterns {
		if containsPattern(code, pattern) {
			return true
		}
	}
	return false
}

func containsPattern(text, pattern string) bool {
	// Simple substring check - in production, use regex
	return strings.Contains(text, pattern)
}

// PerformanceMonitor monitors performance metrics
type PerformanceMonitor struct {
	logger  *logger.StructuredLogger
	metrics *metrics.MetricsCollector
}

// NewPerformanceMonitor creates a new performance monitor
func NewPerformanceMonitor(logger *logger.StructuredLogger, metrics *metrics.MetricsCollector) *PerformanceMonitor {
	return &PerformanceMonitor{
		logger:  logger,
		metrics: metrics,
	}
}

// MonitorOperation monitors the performance of an operation
func (pm *PerformanceMonitor) MonitorOperation(ctx context.Context, operationName string, fn func() error) error {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		pm.metrics.Timing(ctx, "operation_duration", "Operation duration", duration, map[string]string{
			"operation": operationName,
		})

		// Log slow operations
		if duration > 100*time.Millisecond {
			pm.logger.Warning(ctx, "Slow operation detected",
				map[string]interface{}{
					"operation": operationName,
					"duration":  duration.String(),
				})
		}
	}()

	return fn()
}

// MonitorGoroutines monitors goroutine count
func (pm *PerformanceMonitor) MonitorGoroutines(ctx context.Context) {
	// Note: In Go 1.14+, you can use runtime.NumGoroutine()
	goroutines := runtime.NumGoroutine()

	pm.metrics.Gauge(ctx, "goroutines_total", "Total number of goroutines", float64(goroutines), nil)

	if goroutines > 1000 {
		pm.logger.Warning(ctx, "High goroutine count detected",
			map[string]interface{}{
				"goroutines": goroutines,
			})
	}
}

// DeadlockDetector provides basic deadlock detection
type DeadlockDetector struct {
	logger        *logger.StructuredLogger
	lastActivity  map[string]time.Time
	checkInterval time.Duration
	mutex         sync.RWMutex
}

// NewDeadlockDetector creates a new deadlock detector
func NewDeadlockDetector(logger *logger.StructuredLogger, checkInterval time.Duration) *DeadlockDetector {
	dd := &DeadlockDetector{
		logger:        logger,
		lastActivity:  make(map[string]time.Time),
		checkInterval: checkInterval,
	}

	// Start monitoring
	go dd.monitor()

	return dd
}

// RecordActivity records activity for a component
func (dd *DeadlockDetector) RecordActivity(component string) {
	dd.mutex.Lock()
	defer dd.mutex.Unlock()
	dd.lastActivity[component] = time.Now()
}

// monitor checks for inactive components
func (dd *DeadlockDetector) monitor() {
	ticker := time.NewTicker(dd.checkInterval)
	defer ticker.Stop()

	for range ticker.C {
		dd.checkForDeadlocks()
	}
}

func (dd *DeadlockDetector) checkForDeadlocks() {
	dd.mutex.RLock()
	defer dd.mutex.RUnlock()

	now := time.Now()
	threshold := 5 * time.Minute // Consider deadlocked if no activity for 5 minutes

	for component, lastActivity := range dd.lastActivity {
		if now.Sub(lastActivity) > threshold {
			dd.logger.Warning(context.Background(), "Potential deadlock or inactive component",
				map[string]interface{}{
					"component":     component,
					"last_activity": lastActivity,
					"time_since":    now.Sub(lastActivity).String(),
				})
		}
	}
}
