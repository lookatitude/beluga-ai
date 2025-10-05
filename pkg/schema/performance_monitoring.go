// Package schema provides real-time performance monitoring and alerting.
// Enhancement: Real-time performance tracking with alerting capabilities
package schema

import (
	"context"
	"sync"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/schema/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema/internal"
)

// Global performance monitoring integration

// EnablePerformanceMonitoring initializes and starts global performance monitoring for schema operations
func EnablePerformanceMonitoring() error {
	thresholds := internal.DefaultPerformanceThresholds()

	// Initialize global health monitoring
	healthConfig := internal.CreateDefaultValidationConfig()
	internal.InitializeGlobalHealthMonitoring(healthConfig)

	// Initialize real-time performance monitoring
	internal.InitializeRealTimePerformanceMonitoring(thresholds, time.Minute)

	// Enable metrics health monitoring if global metrics is available
	if globalMetrics != nil {
		InitializeMetricsHealthMonitoring(globalMetrics, healthConfig)
	}

	return nil
}

// DisablePerformanceMonitoring stops global performance monitoring
func DisablePerformanceMonitoring() error {
	if monitor := internal.GetGlobalPerformanceMonitor(); monitor != nil {
		return monitor.StopMonitoring()
	}

	if healthManager := internal.GetGlobalHealthManager(); healthManager != nil {
		return healthManager.StopPeriodicChecks()
	}

	return nil
}

// GetPerformanceInsights returns current performance insights and recommendations
func GetPerformanceInsights() []internal.PerformanceInsight {
	return internal.GetGlobalPerformanceInsights()
}

// GetSchemaHealthStatus returns comprehensive health status for the schema package
func GetSchemaHealthStatus(ctx context.Context) map[string]interface{} {
	healthManager := internal.GetGlobalHealthManager()
	if healthManager == nil {
		return map[string]interface{}{
			"status":  "unknown",
			"message": "Health monitoring not initialized",
		}
	}

	overallHealth := healthManager.GetOverallHealth(ctx)

	result := map[string]interface{}{
		"status":            overallHealth.Status,
		"healthy_count":     overallHealth.HealthyCount,
		"unhealthy_count":   overallHealth.UnhealthyCount,
		"degraded_count":    overallHealth.DegradedCount,
		"total_components":  overallHealth.TotalComponents,
		"overall_score":     overallHealth.OverallScore,
		"last_checked":      overallHealth.LastChecked,
		"component_details": overallHealth.Components,
	}

	// Add performance insights
	insights := GetPerformanceInsights()
	if len(insights) > 0 {
		result["performance_insights"] = insights
	}

	// Add current metrics
	if monitor := internal.GetGlobalPerformanceMonitor(); monitor != nil {
		metrics := monitor.GetCurrentMetrics()
		result["operation_metrics"] = metrics
	}

	return result
}

// RegisterPerformanceAlertCallback allows external systems to receive performance alerts
func RegisterPerformanceAlertCallback(callback func(internal.PerformanceAlert)) error {
	if monitor := internal.GetGlobalPerformanceMonitor(); monitor != nil {
		monitor.RegisterAlertCallback(callback)
		return nil
	}

	return iface.NewSchemaError("MONITOR_NOT_INITIALIZED", "Performance monitor not initialized")
}

// Performance wrapper functions for automatic monitoring

// WithPerformanceMonitoring wraps an operation with automatic performance tracking
func WithPerformanceMonitoring(operationName string, operation func() error) error {
	start := time.Now()
	err := operation()
	duration := time.Since(start)

	// Record in global performance monitor
	internal.RecordGlobalOperation(operationName, duration, err == nil)

	// Record in health monitoring
	if validationHealth := internal.GetGlobalValidationHealth(); validationHealth != nil {
		validationHealth.RecordValidationOperation(duration, err == nil, err)
	}

	if factoryHealth := internal.GetGlobalFactoryHealth(); factoryHealth != nil {
		factoryHealth.RecordFactoryOperation(operationName, duration, err == nil, err)
	}

	return err
}

// MonitoredNewHumanMessage creates a human message with performance monitoring
func MonitoredNewHumanMessage(content string) Message {
	var result Message

	WithPerformanceMonitoring("NewHumanMessage", func() error {
		result = NewHumanMessage(content)
		if result == nil {
			return iface.NewSchemaError("CREATION_FAILED", "Failed to create human message")
		}
		return nil
	})

	return result
}

// MonitoredNewAIMessage creates an AI message with performance monitoring
func MonitoredNewAIMessage(content string) Message {
	var result Message

	WithPerformanceMonitoring("NewAIMessage", func() error {
		result = NewAIMessage(content)
		if result == nil {
			return iface.NewSchemaError("CREATION_FAILED", "Failed to create AI message")
		}
		return nil
	})

	return result
}

// MonitoredValidateMessage validates a message with performance monitoring
func MonitoredValidateMessage(message Message) error {
	var validationError error

	err := WithPerformanceMonitoring("ValidateMessage", func() error {
		validationError = ValidateMessage(message)
		return validationError
	})

	// If monitoring wrapper failed, return that error
	if err != nil {
		return err
	}

	// Return actual validation result
	return validationError
}

// Performance analysis helpers

// AnalyzeMessageCreationPerformance provides detailed performance analysis for message creation
func AnalyzeMessageCreationPerformance(iterations int) map[string]interface{} {
	if iterations <= 0 {
		iterations = 1000
	}

	results := make(map[string]interface{})

	// Test different message types
	messageTypes := map[string]func() Message{
		"HumanMessage":    func() Message { return NewHumanMessage("performance test") },
		"AIMessage":       func() Message { return NewAIMessage("performance test") },
		"SystemMessage":   func() Message { return NewSystemMessage("performance test") },
		"ToolMessage":     func() Message { return NewToolMessage("result", "call_id") },
		"FunctionMessage": func() Message { return NewFunctionMessage("func", "result") },
	}

	for typeName, creator := range messageTypes {
		start := time.Now()

		for i := 0; i < iterations; i++ {
			msg := creator()
			if msg.GetContent() == "" {
				results[typeName+"_errors"] = "Empty content detected"
				continue
			}
		}

		elapsed := time.Since(start)
		avgTime := elapsed / time.Duration(iterations)

		results[typeName] = map[string]interface{}{
			"total_time":   elapsed.String(),
			"avg_time":     avgTime.String(),
			"avg_time_ns":  avgTime.Nanoseconds(),
			"iterations":   iterations,
			"ops_per_sec":  float64(iterations) / elapsed.Seconds(),
			"meets_target": avgTime < time.Millisecond, // 1ms target
		}
	}

	return results
}

// AnalyzeFactoryPerformance provides detailed performance analysis for factory functions
func AnalyzeFactoryPerformance(iterations int) map[string]interface{} {
	if iterations <= 0 {
		iterations = 10000
	}

	results := make(map[string]interface{})

	// Test factory functions
	factories := map[string]func(){
		"NewHumanMessage":    func() { NewHumanMessage("test") },
		"NewAIMessage":       func() { NewAIMessage("test") },
		"NewSystemMessage":   func() { NewSystemMessage("test") },
		"NewBaseChatHistory": func() { NewBaseChatHistory() },
		"NewDocument":        func() { NewDocument("test", map[string]string{"type": "test"}) },
	}

	for factoryName, factory := range factories {
		start := time.Now()

		for i := 0; i < iterations; i++ {
			factory()
		}

		elapsed := time.Since(start)
		avgTime := elapsed / time.Duration(iterations)

		results[factoryName] = map[string]interface{}{
			"total_time":   elapsed.String(),
			"avg_time":     avgTime.String(),
			"avg_time_ns":  avgTime.Nanoseconds(),
			"iterations":   iterations,
			"ops_per_sec":  float64(iterations) / elapsed.Seconds(),
			"meets_target": avgTime < 100*time.Microsecond, // 100μs target
		}
	}

	return results
}

// ValidatePerformanceTargets checks if all operations meet constitutional performance targets
func ValidatePerformanceTargets() map[string]bool {
	// Test with reasonable iteration counts
	messageResults := AnalyzeMessageCreationPerformance(1000)
	factoryResults := AnalyzeFactoryPerformance(10000)

	validation := make(map[string]bool)

	// Check message creation targets (<1ms)
	for messageType := range messageResults {
		if typeResult, ok := messageResults[messageType].(map[string]interface{}); ok {
			if meetsTarget, exists := typeResult["meets_target"]; exists {
				validation["message_"+messageType] = meetsTarget.(bool)
			}
		}
	}

	// Check factory targets (<100μs)
	for factoryName := range factoryResults {
		if factoryResult, ok := factoryResults[factoryName].(map[string]interface{}); ok {
			if meetsTarget, exists := factoryResult["meets_target"]; exists {
				validation["factory_"+factoryName] = meetsTarget.(bool)
			}
		}
	}

	return validation
}

// Global performance status
var (
	performanceStatusMu   sync.RWMutex
	lastPerformanceCheck  time.Time
	performanceTargetsMet bool
)

// GetPerformanceStatus returns the current performance status of the schema package
func GetPerformanceStatus() map[string]interface{} {
	performanceStatusMu.RLock()
	defer performanceStatusMu.RUnlock()

	status := map[string]interface{}{
		"last_check":        lastPerformanceCheck,
		"targets_met":       performanceTargetsMet,
		"monitoring_active": internal.GetGlobalPerformanceMonitor() != nil,
		"health_active":     internal.GetGlobalHealthManager() != nil,
	}

	// Add current performance metrics if available
	if monitor := internal.GetGlobalPerformanceMonitor(); monitor != nil {
		metrics := monitor.GetCurrentMetrics()
		status["current_metrics"] = metrics
	}

	return status
}

// UpdatePerformanceStatus updates the global performance status
func UpdatePerformanceStatus() {
	performanceStatusMu.Lock()
	defer performanceStatusMu.Unlock()

	targets := ValidatePerformanceTargets()
	allTargetsMet := true

	for _, targetMet := range targets {
		if !targetMet {
			allTargetsMet = false
			break
		}
	}

	lastPerformanceCheck = time.Now()
	performanceTargetsMet = allTargetsMet
}

// InitializeSchemaMonitoring provides a single function to initialize all monitoring capabilities
func InitializeSchemaMonitoring(ctx context.Context) error {
	// Enable all performance monitoring
	if err := EnablePerformanceMonitoring(); err != nil {
		return err
	}

	// Update performance status
	UpdatePerformanceStatus()

	// Register default alert callback for logging
	if err := RegisterPerformanceAlertCallback(func(alert internal.PerformanceAlert) {
		// In production, this would integrate with logging systems
		// For now, just update the performance status
		UpdatePerformanceStatus()
	}); err != nil {
		return err
	}

	return nil
}
