// Package contracts defines the API contracts for health check operations
// These interfaces define the contract for health monitoring functionality
// implementing FR-002, FR-020

package contracts

import (
	"context"
	"time"
)

// HealthChecker defines the contract for health monitoring
type HealthChecker interface {
	// HealthCheck performs a health check and returns current status
	// Implements FR-002: System MUST implement comprehensive health checks
	HealthCheck(ctx context.Context) HealthStatus

	// StartHealthChecks begins periodic health checking
	// Enables continuous monitoring as per FR-002
	StartHealthChecks(ctx context.Context, interval time.Duration) error

	// StopHealthChecks stops periodic health checking
	// Provides lifecycle control for health monitoring
	StopHealthChecks() error

	// GetHealthHistory returns historical health data
	// Supports health trend analysis and debugging
	GetHealthHistory(limit int) []HealthStatus

	// RegisterHealthCallback registers a callback for health status changes
	// Enables reactive health management as per FR-020
	RegisterHealthCallback(callback HealthCallback) error
}

// HealthStatus represents the current health state
type HealthStatus struct {
	Status       HealthStatusType        `json:"status"`
	Timestamp    time.Time               `json:"timestamp"`
	Details      map[string]interface{}  `json:"details,omitempty"`
	LastError    string                  `json:"last_error,omitempty"`
	CheckCount   int64                   `json:"check_count"`
	Latency      time.Duration           `json:"latency"`
	Dependencies map[string]HealthStatus `json:"dependencies,omitempty"`
}

// HealthStatusType defines possible health states
type HealthStatusType string

const (
	HealthStatusHealthy   HealthStatusType = "healthy"   // All systems operational
	HealthStatusDegraded  HealthStatusType = "degraded"  // Some issues but functional
	HealthStatusUnhealthy HealthStatusType = "unhealthy" // Critical issues present
	HealthStatusUnknown   HealthStatusType = "unknown"   // Status cannot be determined
)

// SystemHealth aggregates health across all components
type SystemHealth struct {
	OverallStatus HealthStatusType        `json:"overall_status"`
	Timestamp     time.Time               `json:"timestamp"`
	Components    map[string]HealthStatus `json:"components"`

	// Component-specific health
	Registry  HealthStatus            `json:"registry"`
	Loader    HealthStatus            `json:"loader"`
	Validator HealthStatus            `json:"validator"`
	Providers map[string]HealthStatus `json:"providers"`

	// System metrics
	TotalChecks    int64         `json:"total_checks"`
	FailedChecks   int64         `json:"failed_checks"`
	AverageLatency time.Duration `json:"average_latency"`
}

// HealthCallback defines the signature for health change callbacks
type HealthCallback func(component string, oldStatus, newStatus HealthStatus)

// HealthCheckConfig defines configuration for health checking
type HealthCheckConfig struct {
	Enabled          bool          `mapstructure:"enabled" default:"true"`
	Interval         time.Duration `mapstructure:"interval" default:"30s"`
	Timeout          time.Duration `mapstructure:"timeout" default:"5s"`
	FailureThreshold int           `mapstructure:"failure_threshold" default:"3"`
	SuccessThreshold int           `mapstructure:"success_threshold" default:"1"`
	EnableRecovery   bool          `mapstructure:"enable_recovery" default:"true"`
	RecoveryInterval time.Duration `mapstructure:"recovery_interval" default:"60s"`
	HistorySize      int           `mapstructure:"history_size" default:"100"`
	EnableCallbacks  bool          `mapstructure:"enable_callbacks" default:"true"`
}

// HealthError defines structured errors for health check operations
type HealthError struct {
	Op        string // operation that failed
	Component string // component being checked
	Err       error  // underlying error
	Code      string // error code
}

const (
	ErrCodeHealthCheckTimeout = "HEALTH_CHECK_TIMEOUT"
	ErrCodeHealthCheckFailed  = "HEALTH_CHECK_FAILED"
	ErrCodeCallbackFailed     = "CALLBACK_FAILED"
	ErrCodeComponentUnknown   = "COMPONENT_UNKNOWN"
)
