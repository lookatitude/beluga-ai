// Package iface provides health check interfaces for the schema package.
// T024: Create health check interfaces
package iface

import (
	"context"
	"time"
)

// HealthChecker defines the interface for health checking schema components.
// It provides health status information and monitoring capabilities.
type HealthChecker interface {
	// CheckHealth returns the current health status of the component.
	CheckHealth(ctx context.Context) HealthStatus

	// IsHealthy returns a simple boolean health indicator.
	IsHealthy(ctx context.Context) bool

	// GetLastHealthCheck returns the timestamp of the last health check.
	GetLastHealthCheck() time.Time
}

// HealthStatus represents the health status of a schema component.
type HealthStatus struct {
	// Basic Status
	Status        string        `json:"status"` // "healthy", "degraded", "unhealthy"
	LastChecked   time.Time     `json:"last_checked"`
	CheckDuration time.Duration `json:"check_duration"`

	// Component Information
	Component string `json:"component"` // e.g., "message_factory", "validation_system"
	Version   string `json:"version,omitempty"`

	// Performance Metrics
	ResponseTime   time.Duration `json:"response_time"`
	MemoryUsage    int64         `json:"memory_usage,omitempty"`
	OperationCount int64         `json:"operation_count"`
	ErrorCount     int64         `json:"error_count"`
	SuccessRate    float64       `json:"success_rate"`

	// Detailed Information
	Details  map[string]interface{} `json:"details,omitempty"`
	Warnings []string               `json:"warnings,omitempty"`
	Errors   []string               `json:"errors,omitempty"`

	// Dependencies
	Dependencies []DependencyHealth `json:"dependencies,omitempty"`
}

// DependencyHealth represents the health status of a dependency.
type DependencyHealth struct {
	Name         string        `json:"name"`
	Status       string        `json:"status"`
	ResponseTime time.Duration `json:"response_time"`
	LastChecked  time.Time     `json:"last_checked"`
	Required     bool          `json:"required"`
}

// HealthMonitor defines the interface for monitoring health across multiple components.
type HealthMonitor interface {
	// RegisterComponent registers a component for health monitoring.
	RegisterComponent(name string, checker HealthChecker) error

	// UnregisterComponent removes a component from health monitoring.
	UnregisterComponent(name string) error

	// CheckAllComponents checks the health of all registered components.
	CheckAllComponents(ctx context.Context) map[string]HealthStatus

	// StartPeriodicChecks starts periodic health checking with the specified interval.
	StartPeriodicChecks(interval time.Duration) error

	// StopPeriodicChecks stops periodic health checking.
	StopPeriodicChecks() error

	// GetOverallHealth returns aggregated health status for all components.
	GetOverallHealth(ctx context.Context) OverallHealthStatus
}

// OverallHealthStatus represents the aggregated health status.
type OverallHealthStatus struct {
	Status          string                  `json:"status"`
	HealthyCount    int                     `json:"healthy_count"`
	UnhealthyCount  int                     `json:"unhealthy_count"`
	DegradedCount   int                     `json:"degraded_count"`
	TotalComponents int                     `json:"total_components"`
	LastChecked     time.Time               `json:"last_checked"`
	Components      map[string]HealthStatus `json:"components"`
	OverallScore    float64                 `json:"overall_score"` // 0.0 to 1.0
}

// ValidationHealth defines health checking specifically for validation systems.
type ValidationHealth interface {
	HealthChecker

	// CheckValidationRules verifies that validation rules are functioning correctly.
	CheckValidationRules(ctx context.Context) ValidationHealthStatus

	// ValidateHealthConfig checks if the health checking configuration is valid.
	ValidateHealthConfig(config HealthConfig) error

	// GetValidationMetrics returns metrics about validation operations.
	GetValidationMetrics() ValidationMetrics
}

// ValidationHealthStatus provides detailed health information for validation systems.
type ValidationHealthStatus struct {
	HealthStatus

	// Validation-specific metrics
	RulesChecked     int               `json:"rules_checked"`
	RulesPassed      int               `json:"rules_passed"`
	RulesFailed      int               `json:"rules_failed"`
	ValidationErrors []ValidationError `json:"validation_errors,omitempty"`

	// Performance metrics
	AverageValidationTime time.Duration `json:"average_validation_time"`
	ValidationThroughput  float64       `json:"validation_throughput"` // validations/second
}

// ValidationError represents a validation error with health context.
type ValidationError struct {
	Rule      string                 `json:"rule"`
	Message   string                 `json:"message"`
	Severity  string                 `json:"severity"` // "warning", "error", "critical"
	Timestamp time.Time              `json:"timestamp"`
	Context   map[string]interface{} `json:"context,omitempty"`
}

// ValidationMetrics contains metrics about validation operations.
type ValidationMetrics struct {
	TotalValidations      int64         `json:"total_validations"`
	SuccessfulValidations int64         `json:"successful_validations"`
	FailedValidations     int64         `json:"failed_validations"`
	AverageTime           time.Duration `json:"average_time"`
	ThroughputPerSecond   float64       `json:"throughput_per_second"`
	LastValidation        time.Time     `json:"last_validation"`
}

// HealthConfig defines configuration for health checking behavior.
type HealthConfig struct {
	// Check Configuration
	EnableHealthChecks bool          `json:"enable_health_checks"`
	CheckInterval      time.Duration `json:"check_interval"`
	CheckTimeout       time.Duration `json:"check_timeout"`

	// Performance Thresholds
	MaxResponseTime time.Duration `json:"max_response_time"`
	MaxMemoryUsage  int64         `json:"max_memory_usage"`
	MinSuccessRate  float64       `json:"min_success_rate"`

	// Alerting Configuration
	EnableAlerts    bool               `json:"enable_alerts"`
	AlertThresholds map[string]float64 `json:"alert_thresholds,omitempty"`
	AlertChannels   []string           `json:"alert_channels,omitempty"`

	// Component-specific Configuration
	ComponentConfigs map[string]ComponentHealthConfig `json:"component_configs,omitempty"`
}

// ComponentHealthConfig defines health check configuration for specific components.
type ComponentHealthConfig struct {
	Enabled            bool          `json:"enabled"`
	CheckInterval      time.Duration `json:"check_interval"`
	Timeout            time.Duration `json:"timeout"`
	MaxRetries         int           `json:"max_retries"`
	RequiredForOverall bool          `json:"required_for_overall"`
}

// HealthReporter defines the interface for reporting health check results.
type HealthReporter interface {
	// ReportHealth reports a health status result.
	ReportHealth(ctx context.Context, componentName string, status HealthStatus) error

	// ReportHealthEvent reports a health-related event.
	ReportHealthEvent(ctx context.Context, event HealthEvent) error

	// GetHealthHistory returns health check history for a component.
	GetHealthHistory(componentName string, since time.Time) ([]HealthStatus, error)

	// GetHealthSummary returns a summary of health check results.
	GetHealthSummary(since time.Time) HealthSummary
}

// HealthEvent represents a significant health-related event.
type HealthEvent struct {
	EventID        string                 `json:"event_id"`
	Type           string                 `json:"type"` // "status_change", "threshold_exceeded", "recovery"
	Component      string                 `json:"component"`
	Timestamp      time.Time              `json:"timestamp"`
	Message        string                 `json:"message"`
	Severity       string                 `json:"severity"`
	Context        map[string]interface{} `json:"context,omitempty"`
	PreviousStatus string                 `json:"previous_status,omitempty"`
	CurrentStatus  string                 `json:"current_status,omitempty"`
}

// HealthSummary provides a summary of health check results over a time period.
type HealthSummary struct {
	TimeWindow       time.Duration               `json:"time_window"`
	TotalChecks      int64                       `json:"total_checks"`
	HealthyChecks    int64                       `json:"healthy_checks"`
	UnhealthyChecks  int64                       `json:"unhealthy_checks"`
	AvgResponseTime  time.Duration               `json:"avg_response_time"`
	ComponentSummary map[string]ComponentSummary `json:"component_summary"`
	Events           []HealthEvent               `json:"events,omitempty"`
	Trends           HealthTrends                `json:"trends"`
}

// ComponentSummary provides health summary for a specific component.
type ComponentSummary struct {
	Component       string        `json:"component"`
	TotalChecks     int64         `json:"total_checks"`
	SuccessRate     float64       `json:"success_rate"`
	AvgResponseTime time.Duration `json:"avg_response_time"`
	LastStatus      string        `json:"last_status"`
	LastChecked     time.Time     `json:"last_checked"`
}

// HealthTrends contains trend information about health metrics.
type HealthTrends struct {
	SuccessRateTrend  string  `json:"success_rate_trend"`  // "improving", "degrading", "stable"
	ResponseTimeTrend string  `json:"response_time_trend"` // "improving", "degrading", "stable"
	ErrorRateTrend    string  `json:"error_rate_trend"`    // "improving", "degrading", "stable"
	OverallTrend      string  `json:"overall_trend"`       // "improving", "degrading", "stable"
	TrendConfidence   float64 `json:"trend_confidence"`    // 0.0 to 1.0
}

// Health status constants
const (
	HealthStatusHealthy   = "healthy"
	HealthStatusDegraded  = "degraded"
	HealthStatusUnhealthy = "unhealthy"
	HealthStatusUnknown   = "unknown"
)

// Health event types
const (
	HealthEventStatusChange      = "status_change"
	HealthEventThresholdExceeded = "threshold_exceeded"
	HealthEventRecovery          = "recovery"
	HealthEventPerformanceDrop   = "performance_drop"
	HealthEventErrorSpike        = "error_spike"
)
