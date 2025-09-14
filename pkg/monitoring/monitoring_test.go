package monitoring

import (
	"context"
	"testing"

	configIface "github.com/lookatitude/beluga-ai/pkg/config/iface"
	"github.com/lookatitude/beluga-ai/pkg/monitoring/internal/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMonitor(t *testing.T) {
	monitor, err := NewMonitor()
	assert.NoError(t, err)
	assert.NotNil(t, monitor)

	// Test that components are initialized
	assert.NotNil(t, monitor.Logger())
	assert.NotNil(t, monitor.Tracer())
	assert.NotNil(t, monitor.Metrics())
	assert.NotNil(t, monitor.HealthChecker())
	assert.NotNil(t, monitor.SafetyChecker())
	assert.NotNil(t, monitor.EthicalChecker())
	assert.NotNil(t, monitor.BestPracticesChecker())
}

func TestNewMonitorWithConfig(t *testing.T) {
	tests := []struct {
		name       string
		mainConfig *configIface.Config
		wantErr    bool
	}{
		{
			name:       "with nil config",
			mainConfig: nil,
			wantErr:    false,
		},
		{
			name:       "with mock config",
			mainConfig: &configIface.Config{},
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			monitor, err := NewMonitorWithConfig(tt.mainConfig)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, monitor)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, monitor)
			}
		})
	}
}

func TestMonitorLifecycle(t *testing.T) {
	monitor, err := NewMonitor()
	require.NoError(t, err)

	ctx := context.Background()

	// Test Start
	err = monitor.Start(ctx)
	assert.NoError(t, err)

	// Test IsHealthy
	healthy := monitor.IsHealthy(ctx)
	assert.True(t, healthy)

	// Test Stop
	err = monitor.Stop(ctx)
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
			name:    "valid default config",
			config:  DefaultConfig(),
			wantErr: false,
		},
		{
			name: "invalid service name - empty",
			config: func() Config {
				c := DefaultConfig()
				c.ServiceName = ""
				return c
			}(),
			wantErr: true,
		},
		{
			name: "invalid service name - too long",
			config: func() Config {
				c := DefaultConfig()
				c.ServiceName = string(make([]byte, 101)) // 101 characters
				return c
			}(),
			wantErr: true,
		},
		{
			name: "invalid sample rate - negative",
			config: func() Config {
				c := DefaultConfig()
				c.Tracing.SampleRate = -0.1
				return c
			}(),
			wantErr: true,
		},
		{
			name: "invalid sample rate - too high",
			config: func() Config {
				c := DefaultConfig()
				c.Tracing.SampleRate = 1.5
				return c
			}(),
			wantErr: true,
		},
		{
			name: "invalid log level",
			config: func() Config {
				c := DefaultConfig()
				c.Logging.Level = "invalid"
				return c
			}(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errString != "" {
					assert.Contains(t, err.Error(), tt.errString)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestLoadConfig(t *testing.T) {
	config, err := LoadConfig()
	assert.NoError(t, err)
	assert.NotEmpty(t, config.ServiceName)
}

func TestLoadFromMainConfig(t *testing.T) {
	mockConfig := &configIface.Config{}
	config, err := LoadFromMainConfig(mockConfig)
	assert.NoError(t, err)
	assert.NotNil(t, config)
}

func TestValidateWithMainConfig(t *testing.T) {
	config := DefaultConfig()

	mockConfig := &configIface.Config{}
	err := config.ValidateWithMainConfig(mockConfig)
	assert.NoError(t, err)

	// Test with nil main config
	err = config.ValidateWithMainConfig(nil)
	assert.NoError(t, err)
}

func TestLogLevelString(t *testing.T) {
	tests := []struct {
		level    LogLevel
		expected string
	}{
		{DEBUG, "DEBUG"},
		{INFO, "INFO"},
		{WARNING, "WARNING"},
		{ERROR, "ERROR"},
		{FATAL, "FATAL"},
		{LogLevel(999), "UNKNOWN"},
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

	// Test WithMonitoringAndContext
	err := helper.WithMonitoringAndContext(ctx, "test_operation", func(ctx context.Context) error {
		return nil
	})
	assert.NoError(t, err)

	// Test CheckSafety
	result, err := helper.CheckSafety(ctx, "test content", "test context")
	assert.NoError(t, err)
	assert.True(t, result.Safe)

	// Test ValidateBestPractices
	issues := helper.ValidateBestPractices(ctx, "test code", "test component")
	assert.NotNil(t, issues)

	// Test LogEvent
	helper.LogEvent(ctx, "info", "test message", map[string]interface{}{"key": "value"})

	// Test IsSystemHealthy
	healthy := helper.IsSystemHealthy(ctx)
	assert.True(t, healthy)
}

func BenchmarkNewMonitor(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NewMonitor()
	}
}

func BenchmarkConfigValidation(b *testing.B) {
	config := DefaultConfig()
	b.ResetTimer()

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
