// Package iface provides best practices validation interfaces
package iface

import "context"

// BestPracticesChecker validates code and system best practices
type BestPracticesChecker interface {
	Validate(ctx context.Context, data interface{}, component string) []ValidationIssue
	AddValidator(validator Validator)
}

// Validator represents a best practices validator
type Validator interface {
	Name() string
	Validate(ctx context.Context, data interface{}) []ValidationIssue
}

// ValidationIssue represents a best practices violation
type ValidationIssue struct {
	Validator  string `json:"validator"`
	Issue      string `json:"issue"`
	Severity   string `json:"severity"`
	Suggestion string `json:"suggestion"`
	Location   string `json:"location,omitempty"`
}

// ConcurrencyValidator checks for concurrency best practices
type ConcurrencyValidator interface {
	Validator
}

// ErrorHandlingValidator checks for proper error handling
type ErrorHandlingValidator interface {
	Validator
}

// ResourceManagementValidator checks for proper resource management
type ResourceManagementValidator interface {
	Validator
}

// SecurityValidator checks for security best practices
type SecurityValidator interface {
	Validator
}

// PerformanceMonitor monitors performance metrics
type PerformanceMonitor interface {
	MonitorOperation(ctx context.Context, operationName string, fn func() error) error
	MonitorGoroutines(ctx context.Context)
}

// DeadlockDetector provides basic deadlock detection
type DeadlockDetector interface {
	RecordActivity(component string)
}
