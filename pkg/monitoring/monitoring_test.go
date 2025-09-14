package monitoring

import (
	"context"
	"testing"

	configIface "github.com/lookatitude/beluga-ai/pkg/config/iface"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMonitor(t *testing.T) {
	monitor, err := NewMonitor()
	require.NoError(t, err)
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
	// Create a mock config
	mockConfig := &configIface.Config{}

	monitor, err := NewMonitorWithConfig(mockConfig)
	require.NoError(t, err)
	assert.NotNil(t, monitor)
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

func TestMonitorWithOptions(t *testing.T) {
	// Since we don't have Option functions defined yet, just test basic creation
	monitor, err := NewMonitor()
	require.NoError(t, err)
	assert.NotNil(t, monitor)
}

func TestConfigValidation(t *testing.T) {
	config := DefaultConfig()

	// Valid config should pass
	err := config.Validate()
	assert.NoError(t, err)

	// Invalid service name should fail
	config.ServiceName = ""
	err = config.Validate()
	assert.Error(t, err)
}

func TestLoadConfig(t *testing.T) {
	config, err := LoadConfig()
	require.NoError(t, err)
	assert.NotEmpty(t, config.ServiceName)
}

func TestLoadFromMainConfig(t *testing.T) {
	mockConfig := &configIface.Config{}
	config, err := LoadFromMainConfig(mockConfig)
	require.NoError(t, err)
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
	assert.Equal(t, "DEBUG", DEBUG.String())
	assert.Equal(t, "INFO", INFO.String())
	assert.Equal(t, "WARNING", WARNING.String())
	assert.Equal(t, "ERROR", ERROR.String())
	assert.Equal(t, "FATAL", FATAL.String())
	assert.Equal(t, "UNKNOWN", LogLevel(999).String())
}
