// Package monitoring provides comprehensive observability, safety, and ethical monitoring
// for AI systems. It integrates structured logging, distributed tracing, metrics collection,
// health checks, safety validation, and ethical AI monitoring to ensure reliable and
// responsible AI operations.
//
// The package follows the Beluga AI framework's design patterns with:
// - Interface segregation for focused, single-responsibility interfaces
// - Dependency inversion through constructor injection
// - Factory pattern with functional options for configuration
// - Comprehensive error handling and observability integration
//
// Example usage:
//
//	import "github.com/lookatitude/beluga-ai/pkg/monitoring"
//
//	// Create a comprehensive monitoring system
//	monitor, err := monitoring.NewMonitor(
//		monitoring.WithServiceName("my-ai-service"),
//		monitoring.WithOpenTelemetry("localhost:4317"),
//		monitoring.WithSafetyChecks(true),
//		monitoring.WithEthicalValidation(true),
//	)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Use in your AI operations
//	ctx, span := monitor.Tracer().StartSpan(ctx, "ai_inference")
//	defer span.End()
//
//	result, err := monitor.SafetyChecker().CheckContent(ctx, userInput, "chat")
//	if err != nil {
//		monitor.Logger().Error(ctx, "Safety check failed", map[string]interface{}{
//			"error": err.Error(),
//		})
//		return err
//	}
//
//	if !result.Safe {
//		monitor.Logger().Warning(ctx, "Content flagged as unsafe", map[string]interface{}{
//			"risk_score": result.RiskScore,
//		})
//		return errors.New("content flagged as unsafe")
//	}
package monitoring

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"

	"github.com/lookatitude/beluga-ai/pkg/config/iface"
	monitoringIface "github.com/lookatitude/beluga-ai/pkg/monitoring/iface"
	"github.com/lookatitude/beluga-ai/pkg/monitoring/internal/best_practices"
	"github.com/lookatitude/beluga-ai/pkg/monitoring/internal/ethics"
	"github.com/lookatitude/beluga-ai/pkg/monitoring/internal/logger"
	"github.com/lookatitude/beluga-ai/pkg/monitoring/internal/metrics"
	"github.com/lookatitude/beluga-ai/pkg/monitoring/internal/safety"
	"github.com/lookatitude/beluga-ai/pkg/monitoring/internal/tracer"
)

// Monitor provides the main interface for comprehensive monitoring.
type Monitor interface {
	// Core monitoring components
	Logger() monitoringIface.Logger
	Tracer() monitoringIface.Tracer
	Metrics() monitoringIface.MetricsCollector
	HealthChecker() monitoringIface.HealthChecker

	// Safety and ethics
	SafetyChecker() monitoringIface.SafetyChecker
	EthicalChecker() monitoringIface.EthicalChecker
	BestPracticesChecker() monitoringIface.BestPracticesChecker

	// Lifecycle management
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	IsHealthy(ctx context.Context) bool
}

// Option represents functional options for Monitor configuration.
type Option func(*monitorConfig)

// monitorConfig holds the configuration for the Monitor.
type monitorConfig struct {
	serviceName         string
	otelEndpoint        string
	logging             LoggingConfig
	config              Config
	logLevel            LogLevel
	enableOpenTelemetry bool
	enableSafetyChecks  bool
	enableEthicalChecks bool
	enableTracing       bool
	enableMetrics       bool
	enableHealthChecks  bool
	enableLogging       bool
}

// NewMonitor creates a new comprehensive monitoring system.
// The monitor provides logging, tracing, metrics, health checks, safety validation,
// and ethical AI monitoring capabilities.
//
// Parameters:
//   - opts: Optional configuration functions (WithServiceName, WithOpenTelemetry, etc.)
//
// Returns:
//   - Monitor: A new monitoring system instance
//   - error: Configuration or initialization errors
//
// Example:
//
//	monitor, err := monitoring.NewMonitor(
//	    monitoring.WithServiceName("my-ai-service"),
//	    monitoring.WithOpenTelemetry("localhost:4317"),
//	    monitoring.WithSafetyChecks(true),
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	ctx, span := monitor.Tracer().StartSpan(ctx, "operation")
//
// Example usage can be found in examples/monitoring/basic/main.go
func NewMonitor(opts ...Option) (Monitor, error) {
	config := &monitorConfig{
		serviceName:         "beluga-ai-service",
		enableOpenTelemetry: false,
		enableSafetyChecks:  true,
		enableEthicalChecks: true,
		enableTracing:       true,
		enableMetrics:       true,
		enableHealthChecks:  true,
		enableLogging:       true,
		logLevel:            INFO,
		config:              DefaultConfig(),
	}

	for _, opt := range opts {
		opt(config)
	}

	// Validate configuration using config package
	if err := config.config.ValidateWithMainConfig(nil); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Initialize OpenTelemetry if enabled
	if config.enableOpenTelemetry {
		if err := initOpenTelemetry(config.serviceName, config.otelEndpoint); err != nil {
			return nil, fmt.Errorf("failed to initialize OpenTelemetry: %w", err)
		}
	}

	monitor := &defaultMonitor{
		config: config,
	}

	// Initialize components
	if err := monitor.initializeComponents(); err != nil {
		return nil, fmt.Errorf("failed to initialize monitoring components: %w", err)
	}

	return monitor, nil
}

// NewMonitorWithConfig creates a new monitoring system integrated with the main config package.
func NewMonitorWithConfig(mainConfig *iface.Config, opts ...Option) (Monitor, error) {
	config := &monitorConfig{
		serviceName:         "beluga-ai-service",
		enableOpenTelemetry: false,
		enableSafetyChecks:  true,
		enableEthicalChecks: true,
		enableTracing:       true,
		enableMetrics:       true,
		enableHealthChecks:  true,
		enableLogging:       true,
		logLevel:            INFO,
	}

	for _, opt := range opts {
		opt(config)
	}

	// Load monitoring config from main config
	monitoringConfig, err := LoadFromMainConfig(mainConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to load monitoring config from main config: %w", err)
	}
	config.config = monitoringConfig

	// Validate configuration with main config integration
	if err := config.config.ValidateWithMainConfig(mainConfig); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Initialize OpenTelemetry if enabled
	if config.enableOpenTelemetry {
		if err := initOpenTelemetry(config.serviceName, config.otelEndpoint); err != nil {
			return nil, fmt.Errorf("failed to initialize OpenTelemetry: %w", err)
		}
	}

	monitor := &defaultMonitor{
		config: config,
	}

	// Initialize components
	if err := monitor.initializeComponents(); err != nil {
		return nil, fmt.Errorf("failed to initialize monitoring components: %w", err)
	}

	return monitor, nil
}

// defaultMonitor implements the Monitor interface.
type defaultMonitor struct {
	config               *monitorConfig
	logger               monitoringIface.Logger
	tracer               monitoringIface.Tracer
	metrics              monitoringIface.MetricsCollector
	healthChecker        monitoringIface.HealthChecker
	safetyChecker        monitoringIface.SafetyChecker
	ethicalChecker       monitoringIface.EthicalChecker
	bestPracticesChecker monitoringIface.BestPracticesChecker
}

// initializeComponents sets up all monitoring components.
func (m *defaultMonitor) initializeComponents() error {
	// Initialize logger
	if m.config.enableLogging {
		logger := m.createLogger()
		m.logger = logger
	}

	// Initialize tracer
	if m.config.enableTracing {
		m.tracer = tracer.NewTracer(m.config.serviceName)
	}

	// Initialize metrics collector
	if m.config.enableMetrics {
		metricsCollector := metrics.NewMetricsCollector()
		m.metrics = metricsCollector
	}

	// Initialize health checker
	if m.config.enableHealthChecks {
		healthChecker := metrics.NewSimpleHealthChecker()
		m.healthChecker = healthChecker
	}

	// Initialize safety checker
	if m.config.enableSafetyChecks {
		loggerImpl, ok := m.logger.(*logger.StructuredLogger)
		if !ok {
			return errors.New("logger does not implement StructuredLogger")
		}
		safetyChecker := safety.NewSafetyChecker(loggerImpl)
		m.safetyChecker = safetyChecker
	}

	// Initialize ethical checker
	if m.config.enableEthicalChecks {
		loggerImpl, ok := m.logger.(*logger.StructuredLogger)
		if !ok {
			return errors.New("logger does not implement StructuredLogger")
		}
		ethicalChecker := ethics.NewEthicalAIChecker(loggerImpl)
		m.ethicalChecker = ethicalChecker
	}

	// Initialize best practices checker
	loggerImpl, ok := m.logger.(*logger.StructuredLogger)
	if !ok {
		return errors.New("logger does not implement StructuredLogger")
	}
	metricsImpl, ok := m.metrics.(*metrics.MetricsCollector)
	if !ok {
		return errors.New("metrics does not implement MetricsCollector")
	}
	bestPracticesChecker := best_practices.NewBestPracticesChecker(loggerImpl, metricsImpl)
	m.bestPracticesChecker = bestPracticesChecker

	return nil
}

// Core interface implementations.
func (m *defaultMonitor) Logger() monitoringIface.Logger                 { return m.logger }
func (m *defaultMonitor) Tracer() monitoringIface.Tracer                 { return m.tracer }
func (m *defaultMonitor) Metrics() monitoringIface.MetricsCollector      { return m.metrics }
func (m *defaultMonitor) HealthChecker() monitoringIface.HealthChecker   { return m.healthChecker }
func (m *defaultMonitor) SafetyChecker() monitoringIface.SafetyChecker   { return m.safetyChecker }
func (m *defaultMonitor) EthicalChecker() monitoringIface.EthicalChecker { return m.ethicalChecker }
func (m *defaultMonitor) BestPracticesChecker() monitoringIface.BestPracticesChecker {
	return m.bestPracticesChecker
}

// Lifecycle management.
func (m *defaultMonitor) Start(ctx context.Context) error {
	// Start health check monitoring
	if m.healthChecker != nil {
		// Register default health checks
		m.registerDefaultHealthChecks()
	}

	if m.logger != nil {
		m.logger.Info(ctx, "Monitoring system started",
			map[string]any{
				"service":    m.config.serviceName,
				"components": m.getEnabledComponents(),
			})
	}

	return nil
}

func (m *defaultMonitor) Stop(ctx context.Context) error {
	if m.logger != nil {
		m.logger.Info(ctx, "Monitoring system stopping",
			map[string]any{
				"service": m.config.serviceName,
			})
	}

	// Close resources
	if closer, ok := m.logger.(interface{ Close() error }); ok {
		if err := closer.Close(); err != nil {
			return fmt.Errorf("failed to close logger: %w", err)
		}
	}

	return nil
}

func (m *defaultMonitor) IsHealthy(ctx context.Context) bool {
	if m.healthChecker != nil {
		return m.healthChecker.IsHealthy(ctx)
	}
	return true
}

// registerDefaultHealthChecks registers basic health checks.
func (m *defaultMonitor) registerDefaultHealthChecks() {
	// System health check
	_ = m.healthChecker.RegisterCheck("system", func(ctx context.Context) monitoringIface.HealthCheckResult {
		return monitoringIface.HealthCheckResult{
			Status:    monitoringIface.StatusHealthy,
			Message:   "System is operational",
			CheckName: "system",
			Timestamp: time.Now(),
			Details: map[string]any{
				"service": m.config.serviceName,
			},
		}
	})

	// Components health check
	_ = m.healthChecker.RegisterCheck("components", func(ctx context.Context) monitoringIface.HealthCheckResult {
		status := monitoringIface.StatusHealthy
		message := "All components healthy"
		details := make(map[string]any)

		if m.logger == nil {
			status = monitoringIface.StatusDegraded
			message = "Logger not initialized"
		} else {
			details["logger"] = "healthy"
		}

		if m.tracer == nil {
			status = monitoringIface.StatusDegraded
			message = "Tracer not initialized"
		} else {
			details["tracer"] = "healthy"
		}

		if m.metrics == nil {
			status = monitoringIface.StatusDegraded
			message = "Metrics collector not initialized"
		} else {
			details["metrics"] = "healthy"
		}

		return monitoringIface.HealthCheckResult{
			Status:    status,
			Message:   message,
			CheckName: "components",
			Timestamp: time.Now(),
			Details:   details,
		}
	})
}

// getEnabledComponents returns a list of enabled components.
func (m *defaultMonitor) getEnabledComponents() []string {
	components := []string{}

	if m.config.enableLogging {
		components = append(components, "logging")
	}
	if m.config.enableTracing {
		components = append(components, "tracing")
	}
	if m.config.enableMetrics {
		components = append(components, "metrics")
	}
	if m.config.enableHealthChecks {
		components = append(components, "health_checks")
	}
	if m.config.enableSafetyChecks {
		components = append(components, "safety")
	}
	if m.config.enableEthicalChecks {
		components = append(components, "ethics")
	}

	return components
}

// createLogger creates a configured logger instance.
func (m *defaultMonitor) createLogger() monitoringIface.Logger {
	var opts []logger.LoggerOption

	// Configure logger based on config
	if m.config.logging.Format == "json" {
		opts = append(opts, logger.WithJSONOutput())
	}

	if m.config.logging.UseColors {
		opts = append(opts, logger.WithColors(true))
	}

	if m.config.logging.OutputFile != "" {
		opts = append(opts, logger.WithFileOutput(m.config.logging.OutputFile))
	}

	return logger.NewStructuredLogger(m.config.serviceName, opts...)
}

// initOpenTelemetry initializes OpenTelemetry with the provided endpoint.
func initOpenTelemetry(serviceName, endpoint string) error {
	// This would initialize OpenTelemetry SDK
	// For now, we'll use a placeholder implementation
	tracer := otel.Tracer(serviceName)
	_ = tracer // Use the tracer

	// In a real implementation, this would set up:
	// - Tracer provider
	// - Meter provider
	// - Resource attributes
	// - Exporters

	return nil
}

// LogLevel represents the severity of a log entry.
type LogLevel int

const (
	// Log levels.
	DEBUG LogLevel = iota
	INFO
	WARNING
	ERROR
	FATAL
)

// String returns the string representation of the log level.
func (l LogLevel) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARNING:
		return "WARNING"
	case ERROR:
		return "ERROR"
	case FATAL:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}
