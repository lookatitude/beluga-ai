// Package internal provides advanced health monitoring with trend analysis.
// Enhancement: Advanced Health Metrics with trend analysis and performance tracking
package internal

import (
	"math"
	"sync"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/schema/iface"
)

// HealthTrendAnalyzer implements advanced health monitoring with trend analysis
type HealthTrendAnalyzer struct {
	mu               sync.RWMutex
	healthHistory    []iface.HealthStatus
	maxHistorySize   int
	analysisInterval time.Duration
	trendWindow      time.Duration
}

// NewHealthTrendAnalyzer creates a new health trend analyzer
func NewHealthTrendAnalyzer(maxHistorySize int, analysisInterval, trendWindow time.Duration) *HealthTrendAnalyzer {
	return &HealthTrendAnalyzer{
		healthHistory:    make([]iface.HealthStatus, 0, maxHistorySize),
		maxHistorySize:   maxHistorySize,
		analysisInterval: analysisInterval,
		trendWindow:      trendWindow,
	}
}

// RecordHealthStatus records a health status for trend analysis
func (hta *HealthTrendAnalyzer) RecordHealthStatus(status iface.HealthStatus) {
	hta.mu.Lock()
	defer hta.mu.Unlock()

	// Add new status to history
	hta.healthHistory = append(hta.healthHistory, status)

	// Trim to max size
	if len(hta.healthHistory) > hta.maxHistorySize {
		hta.healthHistory = hta.healthHistory[1:]
	}
}

// AnalyzeTrends analyzes health trends over the configured time window
func (hta *HealthTrendAnalyzer) AnalyzeTrends() iface.HealthTrends {
	hta.mu.RLock()
	defer hta.mu.RUnlock()

	if len(hta.healthHistory) < 2 {
		return iface.HealthTrends{
			SuccessRateTrend:  "stable",
			ResponseTimeTrend: "stable",
			ErrorRateTrend:    "stable",
			OverallTrend:      "stable",
			TrendConfidence:   0.0,
		}
	}

	// Filter history to trend window
	cutoff := time.Now().Add(-hta.trendWindow)
	var recentHistory []iface.HealthStatus
	for _, status := range hta.healthHistory {
		if status.LastChecked.After(cutoff) {
			recentHistory = append(recentHistory, status)
		}
	}

	if len(recentHistory) < 2 {
		return iface.HealthTrends{
			SuccessRateTrend:  "stable",
			ResponseTimeTrend: "stable",
			ErrorRateTrend:    "stable",
			OverallTrend:      "stable",
			TrendConfidence:   0.0,
		}
	}

	// Calculate trends
	successRateTrend := hta.calculateSuccessRateTrend(recentHistory)
	responseTimeTrend := hta.calculateResponseTimeTrend(recentHistory)
	errorRateTrend := hta.calculateErrorRateTrend(recentHistory)

	// Calculate overall trend
	overallTrend := hta.calculateOverallTrend(successRateTrend, responseTimeTrend, errorRateTrend)

	// Calculate confidence based on data points and variance
	confidence := hta.calculateTrendConfidence(recentHistory)

	return iface.HealthTrends{
		SuccessRateTrend:  successRateTrend,
		ResponseTimeTrend: responseTimeTrend,
		ErrorRateTrend:    errorRateTrend,
		OverallTrend:      overallTrend,
		TrendConfidence:   confidence,
	}
}

// calculateSuccessRateTrend analyzes success rate trend
func (hta *HealthTrendAnalyzer) calculateSuccessRateTrend(history []iface.HealthStatus) string {
	if len(history) < 2 {
		return "stable"
	}

	// Calculate linear regression slope for success rate
	n := len(history)
	var sumX, sumY, sumXY, sumX2 float64

	for i, status := range history {
		x := float64(i)
		y := status.SuccessRate
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}

	slope := (float64(n)*sumXY - sumX*sumY) / (float64(n)*sumX2 - sumX*sumX)

	switch {
	case slope > 0.01: // Improving by more than 1% per data point
		return "improving"
	case slope < -0.01: // Degrading by more than 1% per data point
		return "degrading"
	default:
		return "stable"
	}
}

// calculateResponseTimeTrend analyzes response time trend
func (hta *HealthTrendAnalyzer) calculateResponseTimeTrend(history []iface.HealthStatus) string {
	if len(history) < 2 {
		return "stable"
	}

	// Convert response times to nanoseconds for analysis
	n := len(history)
	var sumX, sumY, sumXY, sumX2 float64

	for i, status := range history {
		x := float64(i)
		y := float64(status.ResponseTime.Nanoseconds())
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}

	slope := (float64(n)*sumXY - sumX*sumY) / (float64(n)*sumX2 - sumX*sumX)

	// Calculate relative slope (normalized by average response time)
	avgResponseTime := sumY / float64(n)
	relativeSlope := slope / avgResponseTime

	switch {
	case relativeSlope < -0.05: // Improving by more than 5% per data point
		return "improving"
	case relativeSlope > 0.05: // Degrading by more than 5% per data point
		return "degrading"
	default:
		return "stable"
	}
}

// calculateErrorRateTrend analyzes error rate trend
func (hta *HealthTrendAnalyzer) calculateErrorRateTrend(history []iface.HealthStatus) string {
	if len(history) < 2 {
		return "stable"
	}

	// Calculate error rate slope
	n := len(history)
	var sumX, sumY, sumXY, sumX2 float64

	for i, status := range history {
		x := float64(i)
		errorRate := 1.0 - status.SuccessRate // Convert success rate to error rate
		sumX += x
		sumY += errorRate
		sumXY += x * errorRate
		sumX2 += x * x
	}

	slope := (float64(n)*sumXY - sumX*sumY) / (float64(n)*sumX2 - sumX*sumX)

	switch {
	case slope < -0.01: // Error rate decreasing by more than 1% per data point
		return "improving"
	case slope > 0.01: // Error rate increasing by more than 1% per data point
		return "degrading"
	default:
		return "stable"
	}
}

// calculateOverallTrend determines overall trend based on individual trends
func (hta *HealthTrendAnalyzer) calculateOverallTrend(successTrend, responseTrend, errorTrend string) string {
	improvingCount := 0
	degradingCount := 0

	trends := []string{successTrend, responseTrend, errorTrend}
	for _, trend := range trends {
		switch trend {
		case "improving":
			improvingCount++
		case "degrading":
			degradingCount++
		}
	}

	switch {
	case improvingCount > degradingCount:
		return "improving"
	case degradingCount > improvingCount:
		return "degrading"
	default:
		return "stable"
	}
}

// calculateTrendConfidence calculates confidence level for trend analysis
func (hta *HealthTrendAnalyzer) calculateTrendConfidence(history []iface.HealthStatus) float64 {
	if len(history) < 3 {
		return 0.0
	}

	// Calculate confidence based on data consistency and sample size
	dataPoints := float64(len(history))
	maxDataPoints := float64(hta.maxHistorySize)

	// Base confidence on sample size (more data = higher confidence)
	sampleConfidence := math.Min(dataPoints/maxDataPoints, 1.0)

	// Calculate variance in success rates to assess consistency
	var successRates []float64
	for _, status := range history {
		successRates = append(successRates, status.SuccessRate)
	}

	variance := hta.calculateVariance(successRates)
	consistencyConfidence := math.Max(0.0, 1.0-variance*10) // Lower variance = higher confidence

	// Combined confidence (weighted average)
	return (sampleConfidence*0.7 + consistencyConfidence*0.3)
}

// calculateVariance calculates variance of a float64 slice
func (hta *HealthTrendAnalyzer) calculateVariance(values []float64) float64 {
	if len(values) < 2 {
		return 0.0
	}

	// Calculate mean
	var sum float64
	for _, v := range values {
		sum += v
	}
	mean := sum / float64(len(values))

	// Calculate variance
	var variance float64
	for _, v := range values {
		diff := v - mean
		variance += diff * diff
	}
	variance /= float64(len(values))

	return variance
}

// GetHealthSummary generates a comprehensive health summary with trends
func (hta *HealthTrendAnalyzer) GetHealthSummary(since time.Time) iface.HealthSummary {
	hta.mu.RLock()
	defer hta.mu.RUnlock()

	// Filter history to requested time window
	var filteredHistory []iface.HealthStatus
	for _, status := range hta.healthHistory {
		if status.LastChecked.After(since) {
			filteredHistory = append(filteredHistory, status)
		}
	}

	summary := iface.HealthSummary{
		TimeWindow:       time.Since(since),
		TotalChecks:      int64(len(filteredHistory)),
		ComponentSummary: make(map[string]iface.ComponentSummary),
		Events:           []iface.HealthEvent{}, // Would be populated from event history
		Trends:           hta.AnalyzeTrends(),
	}

	if len(filteredHistory) == 0 {
		return summary
	}

	// Calculate aggregated metrics
	var totalResponseTime time.Duration
	var healthyCount int64

	for _, status := range filteredHistory {
		totalResponseTime += status.ResponseTime
		if status.Status == iface.HealthStatusHealthy {
			healthyCount++
		}
	}

	summary.HealthyChecks = healthyCount
	summary.UnhealthyChecks = summary.TotalChecks - healthyCount
	if summary.TotalChecks > 0 {
		summary.AvgResponseTime = time.Duration(int64(totalResponseTime) / summary.TotalChecks)
	}

	// Generate component summary (aggregated for now, could be per-component)
	if len(filteredHistory) > 0 {
		lastStatus := filteredHistory[len(filteredHistory)-1]
		summary.ComponentSummary["schema_package"] = iface.ComponentSummary{
			Component:       "schema_package",
			TotalChecks:     summary.TotalChecks,
			SuccessRate:     float64(healthyCount) / float64(summary.TotalChecks),
			AvgResponseTime: summary.AvgResponseTime,
			LastStatus:      lastStatus.Status,
			LastChecked:     lastStatus.LastChecked,
		}
	}

	return summary
}

// RealTimePerformanceMonitor provides real-time performance tracking and alerting
type RealTimePerformanceMonitor struct {
	mu                 sync.RWMutex
	operationMetrics   map[string]*OperationMetrics
	alertThresholds    map[string]float64
	alertingEnabled    bool
	alertCallbacks     []func(alert PerformanceAlert)
	monitoringInterval time.Duration
	stopChan           chan bool
	isRunning          bool
}

// OperationMetrics contains real-time metrics for a specific operation
type OperationMetrics struct {
	Name              string
	Count             int64
	SuccessCount      int64
	ErrorCount        int64
	TotalDuration     time.Duration
	MinDuration       time.Duration
	MaxDuration       time.Duration
	LastOperation     time.Time
	CurrentThroughput float64   // Operations per second
	MovingAverage     []float64 // Sliding window of response times
}

// PerformanceAlert represents a performance-related alert
type PerformanceAlert struct {
	Severity    string
	Component   string
	Operation   string
	Message     string
	Threshold   float64
	ActualValue float64
	Timestamp   time.Time
	Context     map[string]interface{}
}

// NewRealTimePerformanceMonitor creates a new real-time performance monitor
func NewRealTimePerformanceMonitor(alertThresholds map[string]float64, monitoringInterval time.Duration) *RealTimePerformanceMonitor {
	return &RealTimePerformanceMonitor{
		operationMetrics:   make(map[string]*OperationMetrics),
		alertThresholds:    alertThresholds,
		alertingEnabled:    true,
		alertCallbacks:     make([]func(PerformanceAlert), 0),
		monitoringInterval: monitoringInterval,
		stopChan:           make(chan bool, 1),
	}
}

// StartMonitoring begins real-time performance monitoring
func (rtpm *RealTimePerformanceMonitor) StartMonitoring() error {
	rtpm.mu.Lock()
	defer rtpm.mu.Unlock()

	if rtpm.isRunning {
		return nil // Already running
	}

	rtpm.isRunning = true

	go func() {
		ticker := time.NewTicker(rtpm.monitoringInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				rtpm.analyzeMetricsAndAlert()
			case <-rtpm.stopChan:
				return
			}
		}
	}()

	return nil
}

// StopMonitoring stops real-time performance monitoring
func (rtpm *RealTimePerformanceMonitor) StopMonitoring() error {
	rtpm.mu.Lock()
	defer rtpm.mu.Unlock()

	if !rtpm.isRunning {
		return nil // Not running
	}

	rtpm.isRunning = false
	rtpm.stopChan <- true
	return nil
}

// RecordOperation records an operation for real-time monitoring
func (rtpm *RealTimePerformanceMonitor) RecordOperation(operationName string, duration time.Duration, success bool) {
	rtpm.mu.Lock()
	defer rtpm.mu.Unlock()

	metrics, exists := rtpm.operationMetrics[operationName]
	if !exists {
		metrics = &OperationMetrics{
			Name:          operationName,
			MinDuration:   duration,
			MaxDuration:   duration,
			MovingAverage: make([]float64, 0, 100), // Keep last 100 operations
		}
		rtpm.operationMetrics[operationName] = metrics
	}

	// Update metrics
	metrics.Count++
	metrics.TotalDuration += duration
	metrics.LastOperation = time.Now()

	if success {
		metrics.SuccessCount++
	} else {
		metrics.ErrorCount++
	}

	// Update min/max
	if duration < metrics.MinDuration {
		metrics.MinDuration = duration
	}
	if duration > metrics.MaxDuration {
		metrics.MaxDuration = duration
	}

	// Update moving average
	durationMs := float64(duration.Nanoseconds()) / 1e6 // Convert to milliseconds
	metrics.MovingAverage = append(metrics.MovingAverage, durationMs)
	if len(metrics.MovingAverage) > 100 {
		metrics.MovingAverage = metrics.MovingAverage[1:]
	}

	// Calculate current throughput (operations per second over last minute)
	oneMinuteAgo := time.Now().Add(-time.Minute)
	recentOps := 0
	if metrics.LastOperation.After(oneMinuteAgo) {
		// This is a simplified calculation - in practice, would maintain time-windowed counter
		recentOps = int(math.Min(float64(metrics.Count), 60.0)) // Approximate
	}
	metrics.CurrentThroughput = float64(recentOps) / 60.0
}

// analyzeMetricsAndAlert analyzes current metrics and triggers alerts if needed
func (rtpm *RealTimePerformanceMonitor) analyzeMetricsAndAlert() {
	rtpm.mu.RLock()
	metrics := make(map[string]*OperationMetrics)
	for name, m := range rtpm.operationMetrics {
		// Create copy for thread safety
		metrics[name] = &OperationMetrics{
			Name:              m.Name,
			Count:             m.Count,
			SuccessCount:      m.SuccessCount,
			ErrorCount:        m.ErrorCount,
			TotalDuration:     m.TotalDuration,
			MinDuration:       m.MinDuration,
			MaxDuration:       m.MaxDuration,
			LastOperation:     m.LastOperation,
			CurrentThroughput: m.CurrentThroughput,
			MovingAverage:     append([]float64(nil), m.MovingAverage...),
		}
	}
	thresholds := make(map[string]float64)
	for k, v := range rtpm.alertThresholds {
		thresholds[k] = v
	}
	rtpm.mu.RUnlock()

	if !rtpm.alertingEnabled {
		return
	}

	// Analyze each operation for alert conditions
	for operationName, opMetrics := range metrics {
		// Check success rate threshold
		if successRateThreshold, exists := thresholds["success_rate"]; exists {
			if opMetrics.Count > 0 {
				successRate := float64(opMetrics.SuccessCount) / float64(opMetrics.Count)
				if successRate < successRateThreshold {
					alert := PerformanceAlert{
						Severity:    "high",
						Component:   "schema_package",
						Operation:   operationName,
						Message:     "Success rate below threshold",
						Threshold:   successRateThreshold,
						ActualValue: successRate,
						Timestamp:   time.Now(),
						Context: map[string]interface{}{
							"total_ops":   opMetrics.Count,
							"success_ops": opMetrics.SuccessCount,
							"error_ops":   opMetrics.ErrorCount,
						},
					}
					rtpm.triggerAlert(alert)
				}
			}
		}

		// Check average response time threshold
		if responseTimeThreshold, exists := thresholds["avg_response_time_ms"]; exists {
			if opMetrics.Count > 0 {
				avgDuration := float64(opMetrics.TotalDuration.Nanoseconds()) / float64(opMetrics.Count) / 1e6 // Convert to ms
				if avgDuration > responseTimeThreshold {
					alert := PerformanceAlert{
						Severity:    "medium",
						Component:   "schema_package",
						Operation:   operationName,
						Message:     "Average response time exceeds threshold",
						Threshold:   responseTimeThreshold,
						ActualValue: avgDuration,
						Timestamp:   time.Now(),
						Context: map[string]interface{}{
							"min_duration_ms": float64(opMetrics.MinDuration.Nanoseconds()) / 1e6,
							"max_duration_ms": float64(opMetrics.MaxDuration.Nanoseconds()) / 1e6,
							"operation_count": opMetrics.Count,
						},
					}
					rtpm.triggerAlert(alert)
				}
			}
		}

		// Check throughput threshold
		if throughputThreshold, exists := thresholds["min_throughput_ops_sec"]; exists {
			if opMetrics.CurrentThroughput < throughputThreshold {
				alert := PerformanceAlert{
					Severity:    "medium",
					Component:   "schema_package",
					Operation:   operationName,
					Message:     "Throughput below threshold",
					Threshold:   throughputThreshold,
					ActualValue: opMetrics.CurrentThroughput,
					Timestamp:   time.Now(),
					Context: map[string]interface{}{
						"last_operation": opMetrics.LastOperation,
						"total_ops":      opMetrics.Count,
					},
				}
				rtpm.triggerAlert(alert)
			}
		}
	}
}

// triggerAlert triggers all registered alert callbacks
func (rtpm *RealTimePerformanceMonitor) triggerAlert(alert PerformanceAlert) {
	for _, callback := range rtpm.alertCallbacks {
		go callback(alert) // Run callbacks asynchronously
	}
}

// RegisterAlertCallback registers a callback function for performance alerts
func (rtpm *RealTimePerformanceMonitor) RegisterAlertCallback(callback func(PerformanceAlert)) {
	rtpm.mu.Lock()
	defer rtpm.mu.Unlock()
	rtpm.alertCallbacks = append(rtpm.alertCallbacks, callback)
}

// GetCurrentMetrics returns current performance metrics
func (rtpm *RealTimePerformanceMonitor) GetCurrentMetrics() map[string]OperationMetrics {
	rtpm.mu.RLock()
	defer rtpm.mu.RUnlock()

	result := make(map[string]OperationMetrics)
	for name, metrics := range rtpm.operationMetrics {
		// Return copies to prevent external modification
		result[name] = OperationMetrics{
			Name:              metrics.Name,
			Count:             metrics.Count,
			SuccessCount:      metrics.SuccessCount,
			ErrorCount:        metrics.ErrorCount,
			TotalDuration:     metrics.TotalDuration,
			MinDuration:       metrics.MinDuration,
			MaxDuration:       metrics.MaxDuration,
			LastOperation:     metrics.LastOperation,
			CurrentThroughput: metrics.CurrentThroughput,
			MovingAverage:     append([]float64(nil), metrics.MovingAverage...),
		}
	}

	return result
}

// PerformanceInsights provides insights based on current metrics
func (rtpm *RealTimePerformanceMonitor) PerformanceInsights() []PerformanceInsight {
	metrics := rtpm.GetCurrentMetrics()
	var insights []PerformanceInsight

	for operationName, opMetrics := range metrics {
		// Analyze operation performance
		if opMetrics.Count > 10 { // Need sufficient data
			avgDuration := float64(opMetrics.TotalDuration.Nanoseconds()) / float64(opMetrics.Count)
			successRate := float64(opMetrics.SuccessCount) / float64(opMetrics.Count)

			// Generate insights based on performance characteristics
			if avgDuration < 50000 { // Less than 50μs (nanoseconds)
				insights = append(insights, PerformanceInsight{
					Category:       "performance",
					Level:          "excellent",
					Operation:      operationName,
					Message:        "Operation performing excellently with sub-50μs response times",
					Recommendation: "Current performance is exceptional, consider this as baseline for other operations",
				})
			} else if avgDuration > 1000000 { // More than 1ms
				insights = append(insights, PerformanceInsight{
					Category:       "performance",
					Level:          "needs_attention",
					Operation:      operationName,
					Message:        "Operation response time exceeds 1ms target",
					Recommendation: "Consider optimization or caching for this operation",
				})
			}

			if successRate < 0.95 { // Less than 95% success rate
				insights = append(insights, PerformanceInsight{
					Category:       "reliability",
					Level:          "concern",
					Operation:      operationName,
					Message:        "Operation success rate below 95%",
					Recommendation: "Investigate error patterns and implement additional error handling",
				})
			}
		}
	}

	return insights
}

// PerformanceInsight represents a performance insight or recommendation
type PerformanceInsight struct {
	Category       string    `json:"category"` // "performance", "reliability", "efficiency"
	Level          string    `json:"level"`    // "excellent", "good", "needs_attention", "concern"
	Operation      string    `json:"operation"`
	Message        string    `json:"message"`
	Recommendation string    `json:"recommendation"`
	Timestamp      time.Time `json:"timestamp"`
}

// Default performance thresholds for schema operations
func DefaultPerformanceThresholds() map[string]float64 {
	return map[string]float64{
		"success_rate":            0.99,   // 99% success rate
		"avg_response_time_ms":    1.0,    // 1ms average response time
		"max_response_time_ms":    5.0,    // 5ms max response time
		"min_throughput_ops_sec":  1000.0, // 1000 ops/sec minimum throughput
		"max_error_rate":          0.01,   // 1% max error rate
		"memory_efficiency_mb_op": 0.001,  // 1KB max per operation
	}
}

// Global real-time performance monitor
var (
	globalPerformanceMonitor *RealTimePerformanceMonitor
	performanceMonitorOnce   sync.Once
)

// InitializeRealTimePerformanceMonitoring initializes global real-time performance monitoring
func InitializeRealTimePerformanceMonitoring(thresholds map[string]float64, interval time.Duration) {
	performanceMonitorOnce.Do(func() {
		if thresholds == nil {
			thresholds = DefaultPerformanceThresholds()
		}
		if interval == 0 {
			interval = time.Minute // Default monitoring interval
		}

		globalPerformanceMonitor = NewRealTimePerformanceMonitor(thresholds, interval)

		// Register default alert callback
		globalPerformanceMonitor.RegisterAlertCallback(func(alert PerformanceAlert) {
			// In production, this would integrate with logging/alerting systems
			// For now, we'll just track the alert internally
		})

		// Start monitoring
		globalPerformanceMonitor.StartMonitoring()
	})
}

// GetGlobalPerformanceMonitor returns the global performance monitor
func GetGlobalPerformanceMonitor() *RealTimePerformanceMonitor {
	return globalPerformanceMonitor
}

// RecordGlobalOperation records an operation in the global performance monitor
func RecordGlobalOperation(operationName string, duration time.Duration, success bool) {
	if globalPerformanceMonitor != nil {
		globalPerformanceMonitor.RecordOperation(operationName, duration, success)
	}
}

// GetGlobalPerformanceInsights returns current performance insights
func GetGlobalPerformanceInsights() []PerformanceInsight {
	if globalPerformanceMonitor != nil {
		return globalPerformanceMonitor.PerformanceInsights()
	}
	return []PerformanceInsight{}
}
