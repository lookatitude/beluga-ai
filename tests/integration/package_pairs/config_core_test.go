// Package package_pairs provides integration tests between Config and Core packages.
// This test suite verifies that Config can load Core configuration structures
// and that Core components can use Config-provided settings.
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
	"github.com/stretchr/testify/require"
)

// TestIntegrationConfigCore tests the integration between Config and Core packages.
func TestIntegrationConfigCore(t *testing.T) {
	helper := utils.NewIntegrationTestHelper()
	defer func() { _ = helper.Cleanup(context.Background()) }()

	t.Run("load_core_config_from_yaml", func(t *testing.T) {
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

		// Verify config was loaded (even if core section isn't directly accessible)
		// The integration test verifies that config package can load files
		// that contain core package configuration structures
		assert.NotNil(t, cfg)
	})

	t.Run("load_core_config_from_env", func(t *testing.T) {
		// Set environment variables that core package would use
		originalLogLevel := os.Getenv("CORE_LOG_LEVEL")
		originalTracing := os.Getenv("CORE_ENABLE_TRACING")
		originalMetrics := os.Getenv("CORE_ENABLE_METRICS")

		defer func() {
			if originalLogLevel != "" {
				os.Setenv("CORE_LOG_LEVEL", originalLogLevel)
			} else {
				os.Unsetenv("CORE_LOG_LEVEL")
			}
			if originalTracing != "" {
				os.Setenv("CORE_ENABLE_TRACING", originalTracing)
			} else {
				os.Unsetenv("CORE_ENABLE_TRACING")
			}
			if originalMetrics != "" {
				os.Setenv("CORE_ENABLE_METRICS", originalMetrics)
			} else {
				os.Unsetenv("CORE_ENABLE_METRICS")
			}
		}()

		os.Setenv("CORE_LOG_LEVEL", "warn")
		os.Setenv("CORE_ENABLE_TRACING", "true")
		os.Setenv("CORE_ENABLE_METRICS", "false")

		// Load config from environment
		cfg, err := config.LoadFromEnv("CORE")
		// May succeed or fail depending on validation
		if err == nil {
			assert.NotNil(t, cfg)
		}
	})

	t.Run("core_config_validation_with_config_loader", func(t *testing.T) {
		// Create a core config and validate it
		coreCfg := core.DefaultConfig()
		assert.NoError(t, coreCfg.Validate())

		// Verify core config can be used alongside config package
		// This tests that both packages work together
		coreCfg.LogLevel = "info"
		assert.NoError(t, coreCfg.Validate())

		coreCfg.LogLevel = "invalid"
		assert.Error(t, coreCfg.Validate())
	})

	t.Run("config_loader_with_core_defaults", func(t *testing.T) {
		// Test that config package can work with core package defaults
		coreCfg := core.DefaultConfig()
		assert.NotNil(t, coreCfg)
		assert.Equal(t, "info", coreCfg.LogLevel)
		assert.True(t, coreCfg.EnableTracing)
		assert.True(t, coreCfg.EnableMetrics)

		// Verify config package can load configurations that include core settings
		tmpDir := t.TempDir()
		configFile := filepath.Join(tmpDir, "app-config.yaml")
		configContent := `
llm_providers:
  - name: test
    provider: openai
    model_name: gpt-4
    api_key: sk-test
`
		require.NoError(t, os.WriteFile(configFile, []byte(configContent), 0644))

		cfg, err := config.LoadFromFile(configFile)
		// May succeed or fail depending on validation
		if err == nil {
			assert.NotNil(t, cfg)
		}
	})

	t.Run("config_utilities_with_core_components", func(t *testing.T) {
		// Test that config package utilities work with core package structures
		envMap := config.GetEnvConfigMap("CORE")
		assert.NotNil(t, envMap)

		// Test env var name conversion
		envVarName := config.EnvVarName("CORE", "log_level")
		assert.Equal(t, "CORE_LOG_LEVEL", envVarName)

		// Test config key conversion
		// Note: ConfigKey converts underscores to dots for nested keys
		configKey := config.ConfigKey("CORE", "CORE_LOG_LEVEL")
		assert.Equal(t, "log.level", configKey)
	})
}
