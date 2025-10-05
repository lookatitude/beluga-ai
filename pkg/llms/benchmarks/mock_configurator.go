// Package benchmarks provides mock configuration utilities for testing
// LLM provider behaviors under various conditions.
package benchmarks

import (
	"fmt"
	"time"
)

// MockConfigurator defines the interface for configuring mock LLM behaviors
// for testing and benchmarking purposes.
type MockConfigurator interface {
	// ConfigureLatency sets the artificial latency for mock responses
	ConfigureLatency(latency time.Duration, variability float64) error

	// ConfigureErrorInjection sets error injection parameters
	ConfigureErrorInjection(errorRate float64, errorTypes []string) error

	// ConfigureTokenGeneration sets token generation speed
	ConfigureTokenGeneration(tokensPerSecond float64, variability float64) error

	// ConfigureMemoryUsage sets memory usage simulation parameters
	ConfigureMemoryUsage(baseUsage int64, streamingMultiplier float64) error

	// ResetToDefaults resets all configurations to default values
	ResetToDefaults() error
}

// DefaultMockConfigurator implements the MockConfigurator interface
type DefaultMockConfigurator struct {
	latency             time.Duration
	latencyVariability  float64
	errorRate           float64
	errorTypes          []string
	tokensPerSecond     float64
	tokenVariability    float64
	baseMemoryUsage     int64
	streamingMultiplier float64
}

// NewMockConfigurator creates a new mock configurator with the given options
func NewMockConfigurator(opts MockConfiguratorOptions) (*DefaultMockConfigurator, error) {
	configurator := &DefaultMockConfigurator{
		latency:             100 * time.Millisecond, // Default latency
		latencyVariability:  0.1,                    // Default variability
		errorRate:           0.0,                    // No errors by default
		errorTypes:          []string{},
		tokensPerSecond:     50.0,        // Default token generation speed
		tokenVariability:    0.1,         // Default variability
		baseMemoryUsage:     1024 * 1024, // 1MB default
		streamingMultiplier: 1.2,         // 20% increase for streaming
	}

	// Apply options if enabled
	if opts.EnableLatencySimulation {
		// Keep defaults
	}

	if opts.EnableErrorInjection {
		// Keep defaults
	}

	if opts.EnableMemorySimulation {
		// Keep defaults
	}

	return configurator, nil
}

// ConfigureLatency sets the artificial latency for mock responses
func (mc *DefaultMockConfigurator) ConfigureLatency(latency time.Duration, variability float64) error {
	if latency < 0 {
		return fmt.Errorf("latency cannot be negative: %v", latency)
	}
	if variability < 0 || variability > 1 {
		return fmt.Errorf("variability must be between 0 and 1: %f", variability)
	}

	mc.latency = latency
	mc.latencyVariability = variability
	return nil
}

// ConfigureErrorInjection sets error injection parameters
func (mc *DefaultMockConfigurator) ConfigureErrorInjection(errorRate float64, errorTypes []string) error {
	if errorRate < 0 || errorRate > 1 {
		return fmt.Errorf("error rate must be between 0 and 1: %f", errorRate)
	}
	if len(errorTypes) == 0 && errorRate > 0 {
		return fmt.Errorf("error types cannot be empty when error rate > 0")
	}

	mc.errorRate = errorRate
	mc.errorTypes = make([]string, len(errorTypes))
	copy(mc.errorTypes, errorTypes)
	return nil
}

// ConfigureTokenGeneration sets token generation speed
func (mc *DefaultMockConfigurator) ConfigureTokenGeneration(tokensPerSecond float64, variability float64) error {
	if tokensPerSecond < 0 {
		return fmt.Errorf("tokens per second cannot be negative: %f", tokensPerSecond)
	}
	if variability < 0 || variability > 1 {
		return fmt.Errorf("variability must be between 0 and 1: %f", variability)
	}

	mc.tokensPerSecond = tokensPerSecond
	mc.tokenVariability = variability
	return nil
}

// ConfigureMemoryUsage sets memory usage simulation parameters
func (mc *DefaultMockConfigurator) ConfigureMemoryUsage(baseUsage int64, streamingMultiplier float64) error {
	if baseUsage < 0 {
		return fmt.Errorf("base memory usage cannot be negative: %d", baseUsage)
	}
	if streamingMultiplier < 0 {
		return fmt.Errorf("streaming multiplier cannot be negative: %f", streamingMultiplier)
	}

	mc.baseMemoryUsage = baseUsage
	mc.streamingMultiplier = streamingMultiplier
	return nil
}

// ResetToDefaults resets all configurations to default values
func (mc *DefaultMockConfigurator) ResetToDefaults() error {
	mc.latency = 100 * time.Millisecond
	mc.latencyVariability = 0.1
	mc.errorRate = 0.0
	mc.errorTypes = []string{}
	mc.tokensPerSecond = 50.0
	mc.tokenVariability = 0.1
	mc.baseMemoryUsage = 1024 * 1024
	mc.streamingMultiplier = 1.2
	return nil
}

// GetLatency returns the current latency configuration
func (mc *DefaultMockConfigurator) GetLatency() (time.Duration, float64) {
	return mc.latency, mc.latencyVariability
}

// GetErrorInjection returns the current error injection configuration
func (mc *DefaultMockConfigurator) GetErrorInjection() (float64, []string) {
	return mc.errorRate, mc.errorTypes
}

// GetTokenGeneration returns the current token generation configuration
func (mc *DefaultMockConfigurator) GetTokenGeneration() (float64, float64) {
	return mc.tokensPerSecond, mc.tokenVariability
}

// GetMemoryUsage returns the current memory usage configuration
func (mc *DefaultMockConfigurator) GetMemoryUsage() (int64, float64) {
	return mc.baseMemoryUsage, mc.streamingMultiplier
}
