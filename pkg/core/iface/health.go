// Package iface defines health checking interfaces for the core package.
// T003: Move HealthChecker interface to iface/health.go while preserving existing imports
package iface

import (
	"context"
)

// HealthChecker defines the interface for components that can report their health status.
// All core components should implement this interface for monitoring and observability.
type HealthChecker interface {
	// CheckHealth performs a health check and returns an error if the component is unhealthy.
	// Context should be used for timeout and cancellation support.
	CheckHealth(ctx context.Context) error
}

// HealthStatus represents detailed health information for a component.
type HealthStatus struct {
	Status      string                 `json:"status"`       // "healthy", "degraded", "unhealthy"
	Component   string                 `json:"component"`    // Component identifier
	LastChecked int64                  `json:"last_checked"` // Unix timestamp
	Details     map[string]interface{} `json:"details,omitempty"`
	Errors      []string               `json:"errors,omitempty"`
	Warnings    []string               `json:"warnings,omitempty"`
}

// AdvancedHealthChecker extends HealthChecker with detailed health reporting.
type AdvancedHealthChecker interface {
	HealthChecker

	// GetHealthStatus returns detailed health status information.
	GetHealthStatus(ctx context.Context) HealthStatus

	// IsHealthy returns a simple boolean health indicator.
	IsHealthy(ctx context.Context) bool
}
