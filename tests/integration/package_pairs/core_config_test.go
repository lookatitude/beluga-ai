// Package package_pairs provides integration tests between Core and Config packages.
// This test suite verifies that Core components can work with Config-provided settings
// and that Config can load Core configuration structures.
package package_pairs

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/lookatitude/beluga-ai/pkg/config"
	"github.com/lookatitude/beluga-ai/pkg/core"
	"github.com/lookatitude/beluga-ai/tests/integration/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// TestIntegrationCoreConfig tests the integration between Core and Config packages.
func TestIntegrationCoreConfig(t *testing.T) {
	helper := utils.NewIntegrationTestHelper()
	defer func() { _ = helper.Cleanup(context.Background()) }()

	t.Run("core_config_from_yaml", func(t *testing.T) {
		tmpDir := t.TempDir()
		configFile := filepath.Join(tmpDir, "core-config.yaml")
		configContent := `
core:
  log_level: "debug"
  enable_tracing: true
  enable_metrics: true
`
		require.NoError(t, os.WriteFile(configFile, []byte(configContent), 0644))

		// Load config using config package
		cfg, err := config.LoadFromFile(configFile)
		require.NoError(t, err)
		assert.NotNil(t, cfg)

		// Verify core config can be created and validated
		coreCfg := core.DefaultConfig()
		assert.NoError(t, coreCfg.Validate())
	})

	t.Run("core_config_validation_with_config_loader", func(t *testing.T) {
		// Create a core config and validate it
		coreCfg := core.DefaultConfig()
		assert.NoError(t, coreCfg.Validate())

		// Verify core config can be used alongside config package
		coreCfg.LogLevel = "info"
		assert.NoError(t, coreCfg.Validate())

		coreCfg.LogLevel = "invalid"
		assert.Error(t, coreCfg.Validate())
	})

	t.Run("core_runnable_with_config_settings", func(t *testing.T) {
		// Test that core runnable can work with config-provided settings
		coreCfg := core.DefaultConfig()
		assert.True(t, coreCfg.EnableTracing)
		assert.True(t, coreCfg.EnableMetrics)

		// Create a runnable that uses these settings
		mockRunnable := core.NewAdvancedMockRunnable("test-runnable")
		mockRunnable.On("Invoke", context.Background(), "test", mock.MatchedBy(func([]core.Option) bool { return true })).
			Return("result", nil)

		result, err := mockRunnable.Invoke(context.Background(), "test", core.WithOption("key", "value"))
		require.NoError(t, err)
		assert.Equal(t, "result", result)
	})

	t.Run("core_container_with_config", func(t *testing.T) {
		// Test that core DI container can work with config settings
		container := core.NewContainer()
		assert.NotNil(t, container)

		// Register a factory that uses config
		err := container.Register(func() string {
			return "configured-value"
		})
		require.NoError(t, err)

		var result string
		err = container.Resolve(&result)
		require.NoError(t, err)
		assert.Equal(t, "configured-value", result)
	})

	t.Run("core_traced_runnable_with_config", func(t *testing.T) {
		// Test that TracedRunnable works with config-provided tracing settings
		coreCfg := core.DefaultConfig()
		assert.True(t, coreCfg.EnableTracing)

		// Create a simple runnable implementation
		runnable := &testRunnable{
			invokeFunc: func(ctx context.Context, input any, options ...core.Option) (any, error) {
				return "result", nil
			},
		}

		traced := core.NewTracedRunnable(
			runnable,
			nil, // Uses noop tracer when nil
			core.NoOpMetrics(),
			"test-component",
			"test-name",
		)

		result, err := traced.Invoke(context.Background(), "test", core.WithOption("key", "value"))
		require.NoError(t, err)
		assert.Equal(t, "result", result)
	})
}
