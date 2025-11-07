package monitoring

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	configIface "github.com/lookatitude/beluga-ai/pkg/config/iface"
	"github.com/lookatitude/beluga-ai/pkg/monitoring/iface"
	"github.com/lookatitude/beluga-ai/pkg/monitoring/internal/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
func TestNewMonitor(t *testing.T) {
	tests := []struct {
		name    string
		opts    []Option
		wantErr bool
	}{
		{
			name:    "default monitor",
			opts:    nil,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			monitor, err := NewMonitor(tt.opts...)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, monitor)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, monitor)

			// Verify all components are initialized
			assert.NotNil(t, monitor.Logger(), "Logger should be initialized")
			assert.NotNil(t, monitor.Tracer(), "Tracer should be initialized")
			assert.NotNil(t, monitor.Metrics(), "Metrics should be initialized")
			assert.NotNil(t, monitor.HealthChecker(), "HealthChecker should be initialized")
			assert.NotNil(t, monitor.SafetyChecker(), "SafetyChecker should be initialized")
			assert.NotNil(t, monitor.EthicalChecker(), "EthicalChecker should be initialized")
			assert.NotNil(t, monitor.BestPracticesChecker(), "BestPracticesChecker should be initialized")
		})
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
}

func TestNewMonitorWithConfig(t *testing.T) {
	tests := []struct {
		name       string
		mainConfig *configIface.Config
		opts       []Option
		wantErr    bool
	}{
		{
			name:       "with nil config",
			mainConfig: nil,
			wantErr:    false,
		},
		{
			name:       "with empty config",
			mainConfig: &configIface.Config{},
			wantErr:    false,
		},
		{
			name:       "with config and options",
			mainConfig: &configIface.Config{},
			opts:       nil,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			monitor, err := NewMonitorWithConfig(tt.mainConfig, tt.opts...)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, monitor)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, monitor)
				assert.NotNil(t, monitor.Logger())
				assert.NotNil(t, monitor.Tracer())
			}
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
		})
	}
}

func TestMonitorLifecycle(t *testing.T) {
	tests := []struct {
		name    string
		opts    []Option
		wantErr bool
	}{
		{
			name:    "basic lifecycle",
			opts:    nil,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			monitor, err := NewMonitor(tt.opts...)
			require.NoError(t, err)
			require.NotNil(t, monitor)

			ctx := context.Background()

			// Test Start
			err = monitor.Start(ctx)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)

			// Test IsHealthy after start
			healthy := monitor.IsHealthy(ctx)
			assert.True(t, healthy, "Monitor should be healthy after starting")

			// Test Stop
			err = monitor.Stop(ctx)
			assert.NoError(t, err)

			// Test IsHealthy after stop (should still be true as components are just stopped)
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
			healthy = monitor.IsHealthy(ctx)
			assert.True(t, healthy, "Monitor should still report healthy after stopping")
		})
	}
}

func TestMonitorLifecycleWithCancellation(t *testing.T) {
	monitor, err := NewMonitor()
	require.NoError(t, err)

	// Test with cancelled context
	// Note: Start and Stop don't currently check context cancellation,
	// they just log. This test verifies they don't panic.
	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	err = monitor.Start(cancelledCtx)
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	// Start doesn't check context, so it should succeed
	assert.NoError(t, err)

	err = monitor.Stop(cancelledCtx)
	// Stop doesn't check context, so it should succeed
	assert.NoError(t, err)
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name      string
		config    Config
		wantErr   bool
		errString string
	}{
		{
			name:      "valid default config",
			config:    DefaultConfig(),
			wantErr:   false,
			errString: "",
		},
		{
			name: "invalid service name - empty",
			config: func() Config {
				c := DefaultConfig()
				c.ServiceName = ""
				return c
			}(),
			wantErr:   true,
			errString: "ServiceName",
		},
		{
			name: "invalid service name - too long",
			config: func() Config {
				c := DefaultConfig()
				c.ServiceName = strings.Repeat("a", 101) // 101 characters
				return c
			}(),
			wantErr:   true,
			errString: "ServiceName",
		},
		{
			name: "invalid tracing sample rate - negative",
			config: func() Config {
				c := DefaultConfig()
				c.Tracing.SampleRate = -0.1
				return c
			}(),
			wantErr:   true,
			errString: "SampleRate",
		},
		{
			name: "invalid tracing sample rate - too high",
			config: func() Config {
				c := DefaultConfig()
				c.Tracing.SampleRate = 1.5
				return c
			}(),
			wantErr:   true,
			errString: "SampleRate",
		},
		{
			name: "invalid log level",
			config: func() Config {
				c := DefaultConfig()
				c.Logging.Level = "invalid_level"
				return c
			}(),
			wantErr:   true,
			errString: "loglevel",
		},
		{
			name: "invalid health check interval - too short",
			config: func() Config {
				c := DefaultConfig()
				c.Health.CheckInterval = 1 * time.Second
				return c
			}(),
			wantErr:   true,
			errString: "CheckInterval",
		},
		{
			name: "invalid safety risk threshold - too low",
			config: func() Config {
				c := DefaultConfig()
				c.Safety.RiskThreshold = -0.1
				return c
			}(),
			wantErr:   true,
			errString: "RiskThreshold",
		},
		{
			name: "invalid safety risk threshold - too high",
			config: func() Config {
				c := DefaultConfig()
				c.Safety.RiskThreshold = 1.5
				return c
			}(),
			wantErr:   true,
			errString: "RiskThreshold",
		},
		{
			name: "invalid ethics fairness threshold",
			config: func() Config {
				c := DefaultConfig()
				c.Ethics.FairnessThreshold = 1.2
				return c
			}(),
			wantErr:   true,
			errString: "FairnessThreshold",
		},
		{
			name: "invalid OpenTelemetry endpoint when enabled",
			config: func() Config {
				c := DefaultConfig()
				c.OpenTelemetry.Enabled = true
				c.OpenTelemetry.Endpoint = ""
				return c
			}(),
			wantErr:   true,
			errString: "endpoint",
		},
		{
			name: "invalid histogram buckets - not sorted",
			config: func() Config {
				c := DefaultConfig()
				c.Metrics.HistogramBuckets = []float64{1.0, 0.5, 2.0} // Not sorted
				return c
			}(),
			wantErr:   true,
			errString: "sorted",
		},
		{
			name: "invalid custom pattern regex",
			config: func() Config {
				c := DefaultConfig()
				c.Safety.CustomPatterns = map[string][]string{
					"test": {"[invalid regex"},
				}
				return c
			}(),
			wantErr:   true,
			errString: "regex",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				assert.Error(t, err, "Expected validation to fail for config: %+v", tt.config)
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
				if tt.errString != "" {
					assert.Contains(t, err.Error(), tt.errString, "Error message should contain expected string")
				}
			} else {
				assert.NoError(t, err, "Expected validation to pass for config")
			}
		})
	}
}

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name    string
		opts    []ConfigOption
		wantErr bool
	}{
		{
			name:    "load default config",
			opts:    nil,
			wantErr: false,
		},
		{
			name:    "load config with service name",
			opts:    []ConfigOption{WithServiceName("test-service")},
			wantErr: false,
		},
		{
			name:    "load config with OpenTelemetry",
			opts:    []ConfigOption{WithOpenTelemetry("localhost:4317")},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := LoadConfig(tt.opts...)
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
			if tt.wantErr {
				assert.Error(t, err)
				assert.Equal(t, Config{}, config)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, config.ServiceName)
				assert.NoError(t, config.Validate())
			}
		})
	}
}

func TestLoadFromMainConfig(t *testing.T) {
	tests := []struct {
		name       string
		mainConfig *configIface.Config
		wantErr    bool
	}{
		{
			name:       "load from nil config",
			mainConfig: nil,
			wantErr:    false,
		},
		{
			name:       "load from empty config",
			mainConfig: &configIface.Config{},
			wantErr:    false,
		},
		{
			name:       "load from config with data",
			mainConfig: &configIface.Config{},
			wantErr:    false,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := LoadFromMainConfig(tt.mainConfig)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, config)
				assert.NoError(t, config.Validate())
			}
		})
	}
}

func TestValidateWithMainConfig(t *testing.T) {
	tests := []struct {
		name       string
		config     Config
		mainConfig *configIface.Config
		wantErr    bool
	}{
		{
			name:       "validate with nil main config",
			config:     DefaultConfig(),
			mainConfig: nil,
			wantErr:    false,
		},
		{
			name:       "validate with empty main config",
			config:     DefaultConfig(),
			mainConfig: &configIface.Config{},
			wantErr:    false,
		},
		{
			name: "validate with empty service name",
			config: func() Config {
				c := DefaultConfig()
				c.ServiceName = ""
				return c
			}(),
			mainConfig: &configIface.Config{},
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.ValidateWithMainConfig(tt.mainConfig)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestLogLevelString(t *testing.T) {
	tests := []struct {
		level    LogLevel
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
		expected string
	}{
		{DEBUG, "DEBUG"},
		{INFO, "INFO"},
		{WARNING, "WARNING"},
		{ERROR, "ERROR"},
		{FATAL, "FATAL"},
		{LogLevel(999), "UNKNOWN"},
		{LogLevel(-1), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.level.String())
		})
	}
}

func TestIntegrationHelper(t *testing.T) {
	mockMonitor := mock.NewMockMonitor()
	helper := NewIntegrationHelper(mockMonitor)

	ctx := context.Background()

	t.Run("WithMonitoringAndContext success", func(t *testing.T) {
		err := helper.WithMonitoringAndContext(ctx, "test_operation", func(ctx context.Context) error {
			return nil
		})
		assert.NoError(t, err)
	})

	t.Run("WithMonitoringAndContext with error", func(t *testing.T) {
		testErr := errors.New("test error")
		err := helper.WithMonitoringAndContext(ctx, "test_operation", func(ctx context.Context) error {
			return testErr
		})
		assert.Error(t, err)
		assert.Equal(t, testErr, err)
	})

	t.Run("CheckSafety", func(t *testing.T) {
		result, err := helper.CheckSafety(ctx, "test content", "test context")
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.True(t, result.Safe) // Mock returns safe
	})

	t.Run("ValidateBestPractices", func(t *testing.T) {
		issues := helper.ValidateBestPractices(ctx, "test code", "test component")
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
		assert.NotNil(t, issues)
		assert.IsType(t, []iface.ValidationIssue{}, issues)
	})

	t.Run("LogEvent", func(t *testing.T) {
		helper.LogEvent(ctx, "info", "test message", map[string]interface{}{"key": "value"})
		helper.LogEvent(ctx, "error", "error message", nil)
		helper.LogEvent(ctx, "unknown", "unknown level", nil) // Should default to info
	})

	t.Run("IsSystemHealthy", func(t *testing.T) {
		healthy := helper.IsSystemHealthy(ctx)
		assert.True(t, healthy) // Mock returns true
	})

	t.Run("RecordMetric", func(t *testing.T) {
		helper.RecordMetric(ctx, "test_metric", "Test metric", 1.0, map[string]string{"key": "value"})
	})
}

func TestIntegrationHelperWithRealisticScenarios(t *testing.T) {
	mockMonitor := mock.NewMockMonitor()
	helper := NewIntegrationHelper(mockMonitor)

	t.Run("Realistic AI operation monitoring", func(t *testing.T) {
		userInput := "Hello, how are you?"
		operationName := "ai_chat_response"

		err := helper.WithMonitoringAndContext(context.Background(), operationName, func(ctx context.Context) error {
			// Simulate safety check
			safetyResult, err := helper.CheckSafety(ctx, userInput, "chat")
			if err != nil {
				return err
			}

			if !safetyResult.Safe {
				helper.LogEvent(ctx, "warning", "Unsafe content detected", map[string]interface{}{
					"content":    userInput,
					"risk_score": safetyResult.RiskScore,
				})
				return errors.New("content flagged as unsafe")
			}

			// Simulate processing
			helper.LogEvent(ctx, "info", "Processing user input", map[string]interface{}{
				"input_length": len(userInput),
				"operation":    operationName,
			})

			// Simulate best practices validation
			issues := helper.ValidateBestPractices(ctx, "sample code", "ai_processor")
			helper.LogEvent(ctx, "info", "Best practices validated", map[string]interface{}{
				"issues_found": len(issues),
			})

			// Record success metric
			helper.RecordMetric(ctx, "ai_operations_total", "Total AI operations", 1.0, map[string]string{
				"operation": operationName,
				"status":    "success",
			})
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()

			return nil
		})

		assert.NoError(t, err)
	})

	t.Run("Error handling scenario", func(t *testing.T) {
		err := helper.WithMonitoringAndContext(context.Background(), "failing_operation", func(ctx context.Context) error {
			// Simulate an error
			testErr := errors.New("simulated processing error")
			helper.LogEvent(ctx, "error", "Operation failed", map[string]interface{}{
				"error": testErr.Error(),
			})
			return testErr
		})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "simulated processing error")
	})
}

func TestMonitorComponentIntegration(t *testing.T) {
	monitor, err := NewMonitor()
	require.NoError(t, err)

	t.Run("Logger integration", func(t *testing.T) {
		logger := monitor.Logger()
		assert.NotNil(t, logger)

		// Test different log levels
		logger.Info(context.Background(), "Test info message", map[string]interface{}{"test": "value"})
		logger.Error(context.Background(), "Test error message", nil)
		logger.Debug(context.Background(), "Test debug message", nil)
		logger.Warning(context.Background(), "Test warning message", nil)

		// Test context logger
		ctxLogger := logger.WithFields(map[string]interface{}{"component": "test"})
		ctxLogger.Info(context.Background(), "Context logger test")
	})

	t.Run("Tracer integration", func(t *testing.T) {
		tracer := monitor.Tracer()
		assert.NotNil(t, tracer)

		// Test span creation
		_, span := tracer.StartSpan(context.Background(), "test_operation")
		assert.NotNil(t, span)

		// Test span operations
		span.SetTag("test", "value")
		span.Log("Test log message", map[string]interface{}{"step": 1})

		tracer.FinishSpan(span)
	})

	t.Run("Metrics integration", func(t *testing.T) {
		metrics := monitor.Metrics()
		assert.NotNil(t, metrics)

		// Test different metric types
		metrics.Counter(context.Background(), "test_counter", "Test counter", 1, map[string]string{"test": "value"})
		metrics.Gauge(context.Background(), "test_gauge", "Test gauge", 42.0, nil)
		metrics.Histogram(context.Background(), "test_histogram", "Test histogram", 1.5, nil)

		// Test timer
		timer := metrics.StartTimer(context.Background(), "test_timer", nil)
		time.Sleep(10 * time.Millisecond)
		timer.Stop(context.Background(), "Test timer completed")
	})

	t.Run("Health checker integration", func(t *testing.T) {
		healthChecker := monitor.HealthChecker()
		assert.NotNil(t, healthChecker)

		// Test health check registration and execution
		checkName := "test_component"
		err := healthChecker.RegisterCheck(checkName, func(ctx context.Context) iface.HealthCheckResult {
			return iface.HealthCheckResult{
				Status:    iface.StatusHealthy,
				Message:   "Test component is healthy",
				CheckName: checkName,
				Timestamp: time.Now(),
			}
		})
		assert.NoError(t, err)

		results := healthChecker.RunChecks(context.Background())
		assert.Contains(t, results, checkName)
		assert.Equal(t, iface.StatusHealthy, results[checkName].Status)
	})

	t.Run("Safety checker integration", func(t *testing.T) {
		safetyChecker := monitor.SafetyChecker()
		assert.NotNil(t, safetyChecker)

		// Test safety check
		result, err := safetyChecker.CheckContent(context.Background(), "This is safe content", "test")
		assert.NoError(t, err)
		assert.NotNil(t, result)
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	t.Run("Ethical checker integration", func(t *testing.T) {
		ethicalChecker := monitor.EthicalChecker()
		assert.NotNil(t, ethicalChecker)

	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
		// Test ethical analysis
		analysis, err := ethicalChecker.CheckContent(context.Background(), "This is ethical content", iface.EthicalContext{
			ContentType: "text",
			Domain:      "general",
		})
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
		assert.NoError(t, err)
		assert.NotNil(t, analysis)
	})

	t.Run("Best practices checker integration", func(t *testing.T) {
		bestPracticesChecker := monitor.BestPracticesChecker()
		assert.NotNil(t, bestPracticesChecker)
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()

		// Test best practices validation
		issues := bestPracticesChecker.Validate(context.Background(), "sample code", "test_component")
		assert.IsType(t, []iface.ValidationIssue{}, issues)
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	})
}

// Benchmark tests
func BenchmarkNewMonitor(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NewMonitor()
	}
}

func BenchmarkNewMonitorWithOptions(b *testing.B) {
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NewMonitor()
	}
}

func BenchmarkConfigValidation(b *testing.B) {
	config := DefaultConfig()
	b.ResetTimer()
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()

	for i := 0; i < b.N; i++ {
		config.Validate()
	}
}

func BenchmarkLoadConfig(b *testing.B) {
	for i := 0; i < b.N; i++ {
		LoadConfig()
	}
}

func BenchmarkIntegrationHelper_WithMonitoringAndContext(b *testing.B) {
	mockMonitor := mock.NewMockMonitor()
	helper := NewIntegrationHelper(mockMonitor)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		helper.WithMonitoringAndContext(ctx, "test_operation", func(ctx context.Context) error {
			return nil
		})
	}
}

func BenchmarkIntegrationHelper_WithSafetyCheck(b *testing.B) {
	mockMonitor := mock.NewMockMonitor()
	helper := NewIntegrationHelper(mockMonitor)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		helper.CheckSafety(ctx, "test content", "benchmark")
	}
}

func BenchmarkMonitorComponentOperations(b *testing.B) {
	monitor, _ := NewMonitor()
	ctx := context.Background()

	b.Run("Logger", func(b *testing.B) {
		logger := monitor.Logger()
		for i := 0; i < b.N; i++ {
			logger.Info(ctx, "benchmark message", nil)
		}
	})

	b.Run("Metrics", func(b *testing.B) {
		metrics := monitor.Metrics()
		for i := 0; i < b.N; i++ {
			metrics.Counter(ctx, "bench_counter", "Benchmark counter", 1, nil)
		}
	})

	b.Run("HealthCheck", func(b *testing.B) {
		healthChecker := monitor.HealthChecker()
		for i := 0; i < b.N; i++ {
			healthChecker.RunChecks(ctx)
		}
	})
}
