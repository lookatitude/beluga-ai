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
				Validator:  "concurrency",
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
				Validator:  "error_handling",
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
				Validator:  "resource_management",
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
				Validator:  "security",
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
		"go func()",      // Uncontrolled goroutines
		"defer",          // Check for defer with mutex
		"mu.Unlock()",    // Mutex unlock in defer (can cause deadlocks)
		"sync.Mutex",     // Mutex usage
		"close.*nil",     // Closing nil channels (simple check)
	}

	// Check for "go func()" pattern
	if strings.Contains(code, "go func()") {
		return true
	}
	
	// Check for defer with mutex unlock (common deadlock pattern)
	if strings.Contains(code, "defer") && (strings.Contains(code, "Unlock") || strings.Contains(code, "mu.Unlock")) {
		return true
	}
	
	// Check for other patterns
	for _, pattern := range patterns {
		if strings.Contains(code, pattern) {
			return true
		}
	}
	return false
}

func containsErrorHandlingIssues(code string) bool {
	// Check for panic usage
	if strings.Contains(code, "panic(") {
		return true
	}

	// Check for log.Fatal
	if strings.Contains(code, "log.Fatal") {
		return true
	}

	// Check for error assignment without checking
	// Pattern: "err := " or ", err := " followed by code that doesn't check err
	if strings.Contains(code, "err :=") || strings.Contains(code, ", err :=") {
		// If err is assigned but not checked (no "if err != nil" or "if err == nil" nearby)
		// This is a simplified check - in practice, you'd do more sophisticated analysis
		if !strings.Contains(code, "err != nil") && !strings.Contains(code, "err == nil") {
			return true
		}
	}

	return false
}

func containsResourceIssues(code string) bool {
	// Check for file operations
	if strings.Contains(code, "os.Open") || strings.Contains(code, "Open(") {
		// Check if there's a defer close nearby (simplified check)
		if !strings.Contains(code, "defer") || !strings.Contains(code, "Close") {
			return true
		}
	}

	// Check for HTTP requests
	if strings.Contains(code, "http.Get") || strings.Contains(code, "http.Post") {
		// Check if there's proper cleanup (simplified check)
		if !strings.Contains(code, "defer") || !strings.Contains(code, "Close") {
			return true
		}
	}

	// Check for database operations
	if strings.Contains(code, "database") || strings.Contains(code, "sql.") || strings.Contains(code, "db.Query") || strings.Contains(code, "db.Exec") {
		// Simplified check - in practice, you'd do more sophisticated analysis
		return true
	}

	return false
}

func containsSecurityIssues(code string) bool {
	// Check for password in string (simplified - looks for password assignment)
	if strings.Contains(code, "password") && strings.Contains(code, ":=") {
		return true
	}

	// Check for SQL injection risks (string concatenation in queries)
	if strings.Contains(code, "SELECT") && strings.Contains(code, "+") {
		return true
	}

	// Check for eval() usage
	if strings.Contains(code, "eval(") {
		return true
	}

	// Check for exec.Command usage
	if strings.Contains(code, "exec.Command") {
		return true
	}

	return false
}

func containsPattern(text, pattern string) bool {
	// Simple substring check - in production, use regex
	// Handle empty pattern
	if pattern == "" {
		return false
	}
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

