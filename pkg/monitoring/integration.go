// Package monitoring provides integration helpers for dependency injection
package monitoring

import (
	"context"

	"github.com/lookatitude/beluga-ai/pkg/monitoring/iface"
)

// IntegrationHelper provides helper functions for integrating monitoring with other packages
type IntegrationHelper struct {
	monitor iface.Monitor
}

// NewIntegrationHelper creates a new integration helper
func NewIntegrationHelper(monitor iface.Monitor) *IntegrationHelper {
	return &IntegrationHelper{monitor: monitor}
}

// WithMonitoring wraps a function with monitoring
func (ih *IntegrationHelper) WithMonitoring(operationName string, fn func() error) error {
	ctx := context.Background()
	ctx, span := ih.monitor.Tracer().StartSpan(ctx, operationName)
	defer ih.monitor.Tracer().FinishSpan(span)

	timer := ih.monitor.Metrics().StartTimer(ctx, operationName+"_duration", map[string]string{
		"operation": operationName,
	})
	defer timer.Stop(ctx, operationName+" completed")

	err := fn()
	if err != nil {
		span.SetError(err)
		ih.monitor.Logger().Error(ctx, operationName+" failed", map[string]interface{}{
			"error": err.Error(),
		})
		return err
	}

	ih.monitor.Logger().Info(ctx, operationName+" completed successfully")
	return nil
}

// WithMonitoringAndContext wraps a function with monitoring and context
func (ih *IntegrationHelper) WithMonitoringAndContext(ctx context.Context, operationName string, fn func(context.Context) error) error {
	ctx, span := ih.monitor.Tracer().StartSpan(ctx, operationName)
	defer ih.monitor.Tracer().FinishSpan(span)

	timer := ih.monitor.Metrics().StartTimer(ctx, operationName+"_duration", map[string]string{
		"operation": operationName,
	})
	defer timer.Stop(ctx, operationName+" completed")

	err := fn(ctx)
	if err != nil {
		span.SetError(err)
		ih.monitor.Logger().Error(ctx, operationName+" failed", map[string]interface{}{
			"error": err.Error(),
		})
		return err
	}

	ih.monitor.Logger().Info(ctx, operationName+" completed successfully")
	return nil
}

// RecordMetric records a custom metric
func (ih *IntegrationHelper) RecordMetric(ctx context.Context, name, description string, value float64, labels map[string]string) {
	ih.monitor.Metrics().Counter(ctx, name, description, value, labels)
}

// LogEvent logs an event with structured data
func (ih *IntegrationHelper) LogEvent(ctx context.Context, level, message string, fields map[string]interface{}) {
	switch level {
	case "debug":
		ih.monitor.Logger().Debug(ctx, message, fields)
	case "info":
		ih.monitor.Logger().Info(ctx, message, fields)
	case "warning":
		ih.monitor.Logger().Warning(ctx, message, fields)
	case "error":
		ih.monitor.Logger().Error(ctx, message, fields)
	case "fatal":
		ih.monitor.Logger().Fatal(ctx, message, fields)
	default:
		ih.monitor.Logger().Info(ctx, message, fields)
	}
}

// CheckSafety performs a safety check on content
func (ih *IntegrationHelper) CheckSafety(ctx context.Context, content, contextInfo string) (iface.SafetyResult, error) {
	return ih.monitor.SafetyChecker().CheckContent(ctx, content, contextInfo)
}

// ValidateBestPractices validates best practices for data
func (ih *IntegrationHelper) ValidateBestPractices(ctx context.Context, data interface{}, component string) []iface.ValidationIssue {
	return ih.monitor.BestPracticesChecker().Validate(ctx, data, component)
}

// WithHealthCheck registers a health check
func (ih *IntegrationHelper) WithHealthCheck(name string, check iface.HealthCheckFunc) error {
	return ih.monitor.HealthChecker().RegisterCheck(name, check)
}

// IsSystemHealthy checks if the system is healthy
func (ih *IntegrationHelper) IsSystemHealthy(ctx context.Context) bool {
	return ih.monitor.IsHealthy(ctx)
}

// Example usage for other packages:
//
// type MyService struct {
//     monitor *monitoring.IntegrationHelper
// }
//
// func NewMyService(monitor iface.Monitor) *MyService {
//     return &MyService{
//         monitor: monitoring.NewIntegrationHelper(monitor),
//     }
// }
//
// func (s *MyService) DoSomething(ctx context.Context, data string) error {
//     return s.monitor.WithMonitoringAndContext(ctx, "my_service.do_something", func(ctx context.Context) error {
//         // Your business logic here
//         result, err := s.monitor.CheckSafety(ctx, data, "my_service")
//         if err != nil {
//             return err
//         }
//         if !result.Safe {
//             return errors.New("content flagged as unsafe")
//         }
//
//         // More logic...
//         s.monitor.LogEvent(ctx, "info", "Operation completed", map[string]interface{}{
//             "data_length": len(data),
//         })
//
//         return nil
//     })
// }
