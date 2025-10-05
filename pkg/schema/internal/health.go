// Package internal provides health monitoring implementations for validation systems.
// T025: Implement health monitoring for validation systems
package internal

import (
	"context"
	"sync"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/schema/iface"
)

// ValidationHealthMonitor implements health monitoring for validation systems
type ValidationHealthMonitor struct {
	mu                  sync.RWMutex
	lastCheck           time.Time
	validationCount     int64
	successCount        int64
	errorCount          int64
	totalValidationTime time.Duration
	errors              []iface.ValidationError
	isEnabled           bool
	config              iface.HealthConfig
}

// NewValidationHealthMonitor creates a new validation health monitor
func NewValidationHealthMonitor(config iface.HealthConfig) *ValidationHealthMonitor {
	return &ValidationHealthMonitor{
		lastCheck: time.Now(),
		errors:    make([]iface.ValidationError, 0),
		isEnabled: config.EnableHealthChecks,
		config:    config,
	}
}

// CheckHealth implements the HealthChecker interface
func (vhm *ValidationHealthMonitor) CheckHealth(ctx context.Context) iface.HealthStatus {
	vhm.mu.RLock()
	defer vhm.mu.RUnlock()

	start := time.Now()
	defer func() {
		vhm.mu.Lock()
		vhm.lastCheck = time.Now()
		vhm.mu.Unlock()
	}()

	status := iface.HealthStatus{
		Component:      "validation_system",
		LastChecked:    start,
		CheckDuration:  time.Since(start),
		OperationCount: vhm.validationCount,
		ErrorCount:     vhm.errorCount,
		Details:        make(map[string]interface{}),
	}

	// Calculate success rate
	if vhm.validationCount > 0 {
		status.SuccessRate = float64(vhm.successCount) / float64(vhm.validationCount)
	} else {
		status.SuccessRate = 1.0 // No operations yet
	}

	// Calculate average response time
	if vhm.validationCount > 0 {
		status.ResponseTime = time.Duration(int64(vhm.totalValidationTime) / vhm.validationCount)
	}

	// Determine overall status
	switch {
	case status.SuccessRate >= vhm.config.MinSuccessRate && status.ResponseTime <= vhm.config.MaxResponseTime:
		status.Status = iface.HealthStatusHealthy
	case status.SuccessRate >= vhm.config.MinSuccessRate*0.8: // 80% of minimum
		status.Status = iface.HealthStatusDegraded
		status.Warnings = append(status.Warnings, "Success rate below optimal threshold")
	default:
		status.Status = iface.HealthStatusUnhealthy
		status.Errors = append(status.Errors, "Success rate below minimum threshold")
	}

	// Check response time
	if status.ResponseTime > vhm.config.MaxResponseTime {
		if status.Status == iface.HealthStatusHealthy {
			status.Status = iface.HealthStatusDegraded
		}
		status.Warnings = append(status.Warnings, "Response time exceeds threshold")
	}

	// Add detailed metrics
	status.Details["validation_count"] = vhm.validationCount
	status.Details["success_count"] = vhm.successCount
	status.Details["error_count"] = vhm.errorCount
	status.Details["avg_validation_time"] = status.ResponseTime
	status.Details["recent_errors"] = len(vhm.errors)

	return status
}

// IsHealthy implements the HealthChecker interface
func (vhm *ValidationHealthMonitor) IsHealthy(ctx context.Context) bool {
	status := vhm.CheckHealth(ctx)
	return status.Status == iface.HealthStatusHealthy
}

// GetLastHealthCheck implements the HealthChecker interface
func (vhm *ValidationHealthMonitor) GetLastHealthCheck() time.Time {
	vhm.mu.RLock()
	defer vhm.mu.RUnlock()
	return vhm.lastCheck
}

// RecordValidationOperation records a validation operation for health monitoring
func (vhm *ValidationHealthMonitor) RecordValidationOperation(duration time.Duration, success bool, err error) {
	if !vhm.isEnabled {
		return
	}

	vhm.mu.Lock()
	defer vhm.mu.Unlock()

	vhm.validationCount++
	vhm.totalValidationTime += duration

	if success {
		vhm.successCount++
	} else {
		vhm.errorCount++

		if err != nil {
			// Record error details
			validationErr := iface.ValidationError{
				Rule:      "validation_operation",
				Message:   err.Error(),
				Severity:  "error",
				Timestamp: time.Now(),
				Context:   map[string]interface{}{"duration": duration.String()},
			}

			// Keep only recent errors (last 100)
			vhm.errors = append(vhm.errors, validationErr)
			if len(vhm.errors) > 100 {
				vhm.errors = vhm.errors[1:]
			}
		}
	}
}

// GetValidationMetrics returns current validation metrics
func (vhm *ValidationHealthMonitor) GetValidationMetrics() iface.ValidationMetrics {
	vhm.mu.RLock()
	defer vhm.mu.RUnlock()

	avgTime := time.Duration(0)
	if vhm.validationCount > 0 {
		avgTime = time.Duration(int64(vhm.totalValidationTime) / vhm.validationCount)
	}

	throughput := float64(0)
	if vhm.totalValidationTime > 0 {
		throughput = float64(vhm.validationCount) / vhm.totalValidationTime.Seconds()
	}

	return iface.ValidationMetrics{
		TotalValidations:      vhm.validationCount,
		SuccessfulValidations: vhm.successCount,
		FailedValidations:     vhm.errorCount,
		AverageTime:           avgTime,
		ThroughputPerSecond:   throughput,
		LastValidation:        vhm.lastCheck,
	}
}

// FactoryHealthMonitor implements health monitoring for factory functions
type FactoryHealthMonitor struct {
	mu                sync.RWMutex
	lastCheck         time.Time
	factoryOperations map[string]*FactoryMetrics
	isEnabled         bool
	config            iface.HealthConfig
}

// FactoryMetrics contains metrics for a specific factory function
type FactoryMetrics struct {
	Name         string
	CallCount    int64
	SuccessCount int64
	ErrorCount   int64
	TotalTime    time.Duration
	LastCall     time.Time
	RecentErrors []error
}

// NewFactoryHealthMonitor creates a new factory health monitor
func NewFactoryHealthMonitor(config iface.HealthConfig) *FactoryHealthMonitor {
	return &FactoryHealthMonitor{
		lastCheck:         time.Now(),
		factoryOperations: make(map[string]*FactoryMetrics),
		isEnabled:         config.EnableHealthChecks,
		config:            config,
	}
}

// CheckHealth implements the HealthChecker interface for factory functions
func (fhm *FactoryHealthMonitor) CheckHealth(ctx context.Context) iface.HealthStatus {
	fhm.mu.RLock()
	defer fhm.mu.RUnlock()

	start := time.Now()
	status := iface.HealthStatus{
		Component:     "factory_system",
		LastChecked:   start,
		CheckDuration: time.Since(start),
		Details:       make(map[string]interface{}),
	}

	var totalOps, totalSuccess, totalErrors int64
	var totalTime time.Duration
	var maxTime time.Duration

	// Aggregate metrics from all factory functions
	for name, metrics := range fhm.factoryOperations {
		totalOps += metrics.CallCount
		totalSuccess += metrics.SuccessCount
		totalErrors += metrics.ErrorCount
		totalTime += metrics.TotalTime

		avgTime := time.Duration(0)
		if metrics.CallCount > 0 {
			avgTime = time.Duration(int64(metrics.TotalTime) / metrics.CallCount)
			if avgTime > maxTime {
				maxTime = avgTime
			}
		}

		status.Details[name+"_calls"] = metrics.CallCount
		status.Details[name+"_avg_time"] = avgTime
		status.Details[name+"_errors"] = len(metrics.RecentErrors)
	}

	status.OperationCount = totalOps
	status.ErrorCount = totalErrors

	// Calculate success rate
	if totalOps > 0 {
		status.SuccessRate = float64(totalSuccess) / float64(totalOps)
	} else {
		status.SuccessRate = 1.0
	}

	// Calculate average response time
	if totalOps > 0 {
		status.ResponseTime = time.Duration(int64(totalTime) / totalOps)
	}

	// Determine health status
	switch {
	case status.SuccessRate >= fhm.config.MinSuccessRate && maxTime <= fhm.config.MaxResponseTime:
		status.Status = iface.HealthStatusHealthy
	case status.SuccessRate >= fhm.config.MinSuccessRate*0.8:
		status.Status = iface.HealthStatusDegraded
		status.Warnings = append(status.Warnings, "Some factory functions performing below optimal threshold")
	default:
		status.Status = iface.HealthStatusUnhealthy
		status.Errors = append(status.Errors, "Factory functions failing at unacceptable rate")
	}

	return status
}

// IsHealthy implements the HealthChecker interface
func (fhm *FactoryHealthMonitor) IsHealthy(ctx context.Context) bool {
	status := fhm.CheckHealth(ctx)
	return status.Status == iface.HealthStatusHealthy
}

// GetLastHealthCheck implements the HealthChecker interface
func (fhm *FactoryHealthMonitor) GetLastHealthCheck() time.Time {
	fhm.mu.RLock()
	defer fhm.mu.RUnlock()
	return fhm.lastCheck
}

// RecordFactoryOperation records a factory operation for health monitoring
func (fhm *FactoryHealthMonitor) RecordFactoryOperation(factoryName string, duration time.Duration, success bool, err error) {
	if !fhm.isEnabled {
		return
	}

	fhm.mu.Lock()
	defer fhm.mu.Unlock()

	metrics, exists := fhm.factoryOperations[factoryName]
	if !exists {
		metrics = &FactoryMetrics{
			Name:         factoryName,
			RecentErrors: make([]error, 0),
		}
		fhm.factoryOperations[factoryName] = metrics
	}

	metrics.CallCount++
	metrics.TotalTime += duration
	metrics.LastCall = time.Now()

	if success {
		metrics.SuccessCount++
	} else {
		metrics.ErrorCount++

		if err != nil {
			// Keep only recent errors (last 10 per factory)
			metrics.RecentErrors = append(metrics.RecentErrors, err)
			if len(metrics.RecentErrors) > 10 {
				metrics.RecentErrors = metrics.RecentErrors[1:]
			}
		}
	}
}

// GetFactoryMetrics returns metrics for all factory functions
func (fhm *FactoryHealthMonitor) GetFactoryMetrics() map[string]FactoryMetrics {
	fhm.mu.RLock()
	defer fhm.mu.RUnlock()

	result := make(map[string]FactoryMetrics)
	for name, metrics := range fhm.factoryOperations {
		// Create a copy to avoid race conditions
		result[name] = FactoryMetrics{
			Name:         metrics.Name,
			CallCount:    metrics.CallCount,
			SuccessCount: metrics.SuccessCount,
			ErrorCount:   metrics.ErrorCount,
			TotalTime:    metrics.TotalTime,
			LastCall:     metrics.LastCall,
			RecentErrors: append([]error(nil), metrics.RecentErrors...),
		}
	}

	return result
}

// ComponentHealthManager manages health monitoring for multiple schema components
type ComponentHealthManager struct {
	mu                  sync.RWMutex
	components          map[string]iface.HealthChecker
	periodicCheckTicker *time.Ticker
	isRunning           bool
	config              iface.HealthConfig
}

// NewComponentHealthManager creates a new component health manager
func NewComponentHealthManager(config iface.HealthConfig) *ComponentHealthManager {
	return &ComponentHealthManager{
		components: make(map[string]iface.HealthChecker),
		isRunning:  false,
		config:     config,
	}
}

// RegisterComponent implements the HealthMonitor interface
func (chm *ComponentHealthManager) RegisterComponent(name string, checker iface.HealthChecker) error {
	chm.mu.Lock()
	defer chm.mu.Unlock()

	chm.components[name] = checker
	return nil
}

// UnregisterComponent implements the HealthMonitor interface
func (chm *ComponentHealthManager) UnregisterComponent(name string) error {
	chm.mu.Lock()
	defer chm.mu.Unlock()

	delete(chm.components, name)
	return nil
}

// CheckAllComponents implements the HealthMonitor interface
func (chm *ComponentHealthManager) CheckAllComponents(ctx context.Context) map[string]iface.HealthStatus {
	chm.mu.RLock()
	components := make(map[string]iface.HealthChecker, len(chm.components))
	for name, checker := range chm.components {
		components[name] = checker
	}
	chm.mu.RUnlock()

	results := make(map[string]iface.HealthStatus)

	// Check all components (can be done in parallel)
	resultsChan := make(chan struct {
		name   string
		status iface.HealthStatus
	}, len(components))

	for name, checker := range components {
		go func(n string, c iface.HealthChecker) {
			status := c.CheckHealth(ctx)
			resultsChan <- struct {
				name   string
				status iface.HealthStatus
			}{name: n, status: status}
		}(name, checker)
	}

	// Collect results
	for i := 0; i < len(components); i++ {
		result := <-resultsChan
		results[result.name] = result.status
	}

	return results
}

// StartPeriodicChecks implements the HealthMonitor interface
func (chm *ComponentHealthManager) StartPeriodicChecks(interval time.Duration) error {
	chm.mu.Lock()
	defer chm.mu.Unlock()

	if chm.isRunning {
		return nil // Already running
	}

	chm.periodicCheckTicker = time.NewTicker(interval)
	chm.isRunning = true

	go func() {
		for {
			select {
			case <-chm.periodicCheckTicker.C:
				ctx, cancel := context.WithTimeout(context.Background(), chm.config.CheckTimeout)
				statuses := chm.CheckAllComponents(ctx)
				cancel()

				// Log health check results (in production, this would integrate with logging)
				for name, status := range statuses {
					if status.Status != iface.HealthStatusHealthy {
						// Would log warning/error here
						_ = name // Placeholder to avoid unused variable
					}
				}
			}
		}
	}()

	return nil
}

// StopPeriodicChecks implements the HealthMonitor interface
func (chm *ComponentHealthManager) StopPeriodicChecks() error {
	chm.mu.Lock()
	defer chm.mu.Unlock()

	if !chm.isRunning {
		return nil // Not running
	}

	if chm.periodicCheckTicker != nil {
		chm.periodicCheckTicker.Stop()
		chm.periodicCheckTicker = nil
	}

	chm.isRunning = false
	return nil
}

// GetOverallHealth implements the HealthMonitor interface
func (chm *ComponentHealthManager) GetOverallHealth(ctx context.Context) iface.OverallHealthStatus {
	statuses := chm.CheckAllComponents(ctx)

	overall := iface.OverallHealthStatus{
		LastChecked:     time.Now(),
		Components:      statuses,
		TotalComponents: len(statuses),
	}

	// Count components by status
	for _, status := range statuses {
		switch status.Status {
		case iface.HealthStatusHealthy:
			overall.HealthyCount++
		case iface.HealthStatusDegraded:
			overall.DegradedCount++
		case iface.HealthStatusUnhealthy:
			overall.UnhealthyCount++
		}
	}

	// Determine overall status
	if overall.UnhealthyCount > 0 {
		overall.Status = iface.HealthStatusUnhealthy
	} else if overall.DegradedCount > 0 {
		overall.Status = iface.HealthStatusDegraded
	} else {
		overall.Status = iface.HealthStatusHealthy
	}

	// Calculate overall score
	if overall.TotalComponents > 0 {
		overall.OverallScore = float64(overall.HealthyCount) / float64(overall.TotalComponents)
	} else {
		overall.OverallScore = 1.0
	}

	return overall
}

// DefaultHealthConfig provides a default health configuration for schema components
func DefaultHealthConfig() iface.HealthConfig {
	return iface.HealthConfig{
		EnableHealthChecks: true,
		CheckInterval:      time.Minute,
		CheckTimeout:       time.Second * 10,
		MaxResponseTime:    time.Millisecond * 100, // 100μs for factory functions
		MaxMemoryUsage:     1024 * 1024,            // 1MB
		MinSuccessRate:     0.99,                   // 99% success rate
		EnableAlerts:       false,
		ComponentConfigs:   make(map[string]iface.ComponentHealthConfig),
	}
}

// CreateDefaultValidationConfig creates default configuration for validation health monitoring
func CreateDefaultValidationConfig() iface.HealthConfig {
	config := DefaultHealthConfig()
	config.MaxResponseTime = time.Millisecond * 5 // 5ms for validation operations
	config.ComponentConfigs["validation"] = iface.ComponentHealthConfig{
		Enabled:            true,
		CheckInterval:      time.Minute * 5,
		Timeout:            time.Second * 2,
		MaxRetries:         3,
		RequiredForOverall: true,
	}
	return config
}

// CreateDefaultFactoryConfig creates default configuration for factory health monitoring
func CreateDefaultFactoryConfig() iface.HealthConfig {
	config := DefaultHealthConfig()
	config.MaxResponseTime = time.Microsecond * 100 // 100μs for factory functions
	config.ComponentConfigs["factory"] = iface.ComponentHealthConfig{
		Enabled:            true,
		CheckInterval:      time.Minute * 2,
		Timeout:            time.Millisecond * 500,
		MaxRetries:         1,
		RequiredForOverall: true,
	}
	return config
}

// Global health monitors for schema package components
var (
	globalValidationHealth *ValidationHealthMonitor
	globalFactoryHealth    *FactoryHealthMonitor
	globalHealthManager    *ComponentHealthManager
	healthOnce             sync.Once
)

// InitializeGlobalHealthMonitoring initializes global health monitoring for the schema package
func InitializeGlobalHealthMonitoring(config iface.HealthConfig) {
	healthOnce.Do(func() {
		globalValidationHealth = NewValidationHealthMonitor(config)
		globalFactoryHealth = NewFactoryHealthMonitor(config)
		globalHealthManager = NewComponentHealthManager(config)

		// Register components with the manager
		globalHealthManager.RegisterComponent("validation", globalValidationHealth)
		globalHealthManager.RegisterComponent("factory", globalFactoryHealth)

		// Start periodic health checks if enabled
		if config.EnableHealthChecks && config.CheckInterval > 0 {
			globalHealthManager.StartPeriodicChecks(config.CheckInterval)
		}
	})
}

// GetGlobalValidationHealth returns the global validation health monitor
func GetGlobalValidationHealth() *ValidationHealthMonitor {
	return globalValidationHealth
}

// GetGlobalFactoryHealth returns the global factory health monitor
func GetGlobalFactoryHealth() *FactoryHealthMonitor {
	return globalFactoryHealth
}

// GetGlobalHealthManager returns the global health manager
func GetGlobalHealthManager() *ComponentHealthManager {
	return globalHealthManager
}

// HealthCheckWrapper provides a wrapper for adding health monitoring to existing functions
type HealthCheckWrapper struct {
	monitor   iface.HealthChecker
	component string
}

// NewHealthCheckWrapper creates a new health check wrapper
func NewHealthCheckWrapper(monitor iface.HealthChecker, component string) *HealthCheckWrapper {
	return &HealthCheckWrapper{
		monitor:   monitor,
		component: component,
	}
}

// WrapOperation wraps an operation with health monitoring
func (hcw *HealthCheckWrapper) WrapOperation(operation func() error) error {
	start := time.Now()
	err := operation()
	duration := time.Since(start)

	// Record operation metrics
	if vhm, ok := hcw.monitor.(*ValidationHealthMonitor); ok {
		vhm.RecordValidationOperation(duration, err == nil, err)
	} else if fhm, ok := hcw.monitor.(*FactoryHealthMonitor); ok {
		fhm.RecordFactoryOperation(hcw.component, duration, err == nil, err)
	}

	return err
}
