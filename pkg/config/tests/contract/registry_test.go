package contract

import (
	"context"
	"testing"

	"github.com/lookatitude/beluga-ai/pkg/config"
	"github.com/lookatitude/beluga-ai/pkg/config/iface"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConfigProviderRegistry_Contract tests the ConfigProviderRegistry interface contract.
// This ensures the registry implementation meets all contractual requirements.
func TestConfigProviderRegistry_Contract(t *testing.T) {
	ctx := context.Background()

	// Test provider registration
	t.Run("RegisterGlobal", func(t *testing.T) {
		providerName := "contract-test-provider"
		creator := func(options config.ProviderOptions) (iface.Provider, error) {
			return config.NewAdvancedMockConfigProvider(providerName, "contract-test"), nil
		}

		// Should not error on first registration
		err := config.RegisterGlobal(providerName, creator)
		assert.NoError(t, err, "RegisterGlobal should succeed for new provider")

		// Should error on duplicate registration
		err = config.RegisterGlobal(providerName, creator)
		assert.Error(t, err, "RegisterGlobal should fail for duplicate provider")
		if configErr, ok := config.AsConfigError(err); ok {
			assert.Equal(t, config.ErrCodeProviderAlreadyRegistered, configErr.GetCode())
		}
	})

	// Test provider listing
	t.Run("ListProviders", func(t *testing.T) {
		providers := config.ListProviders()
		assert.NotNil(t, providers, "ListProviders should return non-nil slice")
		assert.Contains(t, providers, "contract-test-provider", "Should contain registered provider")
	})

	// Test provider discovery
	t.Run("IsProviderRegistered", func(t *testing.T) {
		assert.True(t, config.IsProviderRegistered("contract-test-provider"), "Provider should be registered")
		assert.False(t, config.IsProviderRegistered("nonexistent-provider"), "Nonexistent provider should not be registered")
	})

	// Test provider creation
	t.Run("NewRegistryProvider", func(t *testing.T) {
		options := config.ProviderOptions{
			ProviderType: "contract-test-provider",
			ConfigName:   "test-config",
			ConfigPaths:  []string{"./test"},
		}

		provider, err := config.NewRegistryProvider(ctx, "contract-test-provider", options)
		assert.NoError(t, err, "NewRegistryProvider should succeed for registered provider")
		assert.NotNil(t, provider, "Provider should not be nil")

		// Test that provider implements required interface
		mockProvider, ok := provider.(*config.AdvancedMockConfigProvider)
		require.True(t, ok, "Provider should be AdvancedMockConfigProvider")
		assert.Equal(t, "contract-test-provider", mockProvider.GetName())
	})

	// Test provider creation with unregistered provider
	t.Run("NewRegistryProvider_Unregistered", func(t *testing.T) {
		options := config.ProviderOptions{
			ProviderType: "unregistered-provider",
			ConfigName:   "test-config",
			ConfigPaths:  []string{"./test"},
		}

		provider, err := config.NewRegistryProvider(ctx, "unregistered-provider", options)
		assert.Error(t, err, "NewRegistryProvider should fail for unregistered provider")
		assert.Nil(t, provider, "Provider should be nil for unregistered provider")
		if configErr, ok := config.AsConfigError(err); ok {
			assert.Equal(t, config.ErrCodeProviderNotFound, configErr.GetCode())
		}
	})

	// Test provider metadata
	t.Run("ProviderMetadata", func(t *testing.T) {
		// Register provider with metadata
		metadata := config.ProviderMetadata{
			Name:                 "metadata-test-provider",
			Description:          "Test provider with metadata",
			SupportedFormats:     []string{"yaml", "json"},
			Capabilities:         []string{"file_loading", "validation"},
			HealthCheckSupported: true,
		}

		creator := func(options config.ProviderOptions) (iface.Provider, error) {
			return config.NewAdvancedMockConfigProvider("metadata-test-provider", "metadata-test"), nil
		}

		err := config.RegisterGlobalWithMetadata("metadata-test-provider", creator, metadata)
		assert.NoError(t, err, "RegisterGlobalWithMetadata should succeed")

		// Retrieve metadata
		retrievedMetadata, err := config.GetProviderMetadata("metadata-test-provider")
		assert.NoError(t, err, "GetProviderMetadata should succeed")
		assert.NotNil(t, retrievedMetadata, "Metadata should not be nil")
		assert.Equal(t, "metadata-test-provider", retrievedMetadata.Name)
		assert.Equal(t, "Test provider with metadata", retrievedMetadata.Description)
		assert.Contains(t, retrievedMetadata.SupportedFormats, "yaml")
		assert.Contains(t, retrievedMetadata.Capabilities, "file_loading")
		assert.True(t, retrievedMetadata.HealthCheckSupported)
	})

	// Test provider unregistration
	t.Run("UnregisterProvider", func(t *testing.T) {
		// First register a provider to unregister
		providerName := "unregister-test-provider"
		creator := func(options config.ProviderOptions) (iface.Provider, error) {
			return config.NewAdvancedMockConfigProvider(providerName, "unregister-test"), nil
		}

		err := config.RegisterGlobal(providerName, creator)
		assert.NoError(t, err, "Should register provider for unregistration test")

		assert.True(t, config.IsProviderRegistered(providerName), "Provider should be registered before unregistration")

		// Unregister the provider
		err = config.UnregisterProvider(providerName)
		assert.NoError(t, err, "UnregisterProvider should succeed")

		assert.False(t, config.IsProviderRegistered(providerName), "Provider should not be registered after unregistration")

		// Verify provider list no longer contains it
		providers := config.ListProviders()
		assert.NotContains(t, providers, providerName, "Unregistered provider should not be in list")
	})

	// Test format-based provider discovery
	t.Run("GetProvidersForFormat", func(t *testing.T) {
		// Register providers with different format support
		yamlProvider := "yaml-test-provider"
		jsonProvider := "json-test-provider"

		yamlCreator := func(options config.ProviderOptions) (iface.Provider, error) {
			return config.NewAdvancedMockConfigProvider(yamlProvider, "yaml-test"), nil
		}
		yamlMetadata := config.ProviderMetadata{
			Name:             yamlProvider,
			SupportedFormats: []string{"yaml", "yml"},
			Capabilities:     []string{"file_loading"},
		}

		jsonCreator := func(options config.ProviderOptions) (iface.Provider, error) {
			return config.NewAdvancedMockConfigProvider(jsonProvider, "json-test"), nil
		}
		jsonMetadata := config.ProviderMetadata{
			Name:             jsonProvider,
			SupportedFormats: []string{"json"},
			Capabilities:     []string{"file_loading"},
		}

		err := config.RegisterGlobalWithMetadata(yamlProvider, yamlCreator, yamlMetadata)
		assert.NoError(t, err)
		err = config.RegisterGlobalWithMetadata(jsonProvider, jsonCreator, jsonMetadata)
		assert.NoError(t, err)

		// Test format queries
		yamlProviders, err := config.GetProvidersForFormat("yaml")
		assert.NoError(t, err)
		assert.Contains(t, yamlProviders, yamlProvider)
		assert.NotContains(t, yamlProviders, jsonProvider)

		jsonProviders, err := config.GetProvidersForFormat("json")
		assert.NoError(t, err)
		assert.Contains(t, jsonProviders, jsonProvider)
		assert.NotContains(t, jsonProviders, yamlProvider)

		// Test unsupported format
		unsupportedProviders, err := config.GetProvidersForFormat("xml")
		assert.NoError(t, err)
		assert.Empty(t, unsupportedProviders, "Should return empty slice for unsupported format")
	})

	// Test capability-based provider discovery
	t.Run("GetProviderByCapability", func(t *testing.T) {
		// Register providers with different capabilities
		fileProvider := "file-capability-provider"
		envProvider := "env-capability-provider"

		fileCreator := func(options config.ProviderOptions) (iface.Provider, error) {
			return config.NewAdvancedMockConfigProvider(fileProvider, "file-capability"), nil
		}
		fileMetadata := config.ProviderMetadata{
			Name:         fileProvider,
			Capabilities: []string{"file_loading", "validation"},
		}

		envCreator := func(options config.ProviderOptions) (iface.Provider, error) {
			return config.NewAdvancedMockConfigProvider(envProvider, "env-capability"), nil
		}
		envMetadata := config.ProviderMetadata{
			Name:         envProvider,
			Capabilities: []string{"env_vars", "validation"},
		}

		err := config.RegisterGlobalWithMetadata(fileProvider, fileCreator, fileMetadata)
		assert.NoError(t, err)
		err = config.RegisterGlobalWithMetadata(envProvider, envCreator, envMetadata)
		assert.NoError(t, err)

		// Test capability queries
		fileProviders, err := config.GetProviderByCapability("file_loading")
		assert.NoError(t, err)
		assert.Contains(t, fileProviders, fileProvider)
		assert.NotContains(t, fileProviders, envProvider)

		envProviders, err := config.GetProviderByCapability("env_vars")
		assert.NoError(t, err)
		assert.Contains(t, envProviders, envProvider)
		assert.NotContains(t, envProviders, fileProvider)

		validationProviders, err := config.GetProviderByCapability("validation")
		assert.NoError(t, err)
		assert.Contains(t, validationProviders, fileProvider)
		assert.Contains(t, validationProviders, envProvider)

		// Test unsupported capability
		unsupportedProviders, err := config.GetProviderByCapability("unknown_capability")
		assert.NoError(t, err)
		assert.Empty(t, unsupportedProviders, "Should return empty slice for unsupported capability")
	})
}

// TestConfigProviderRegistry_ErrorConditions tests error conditions in registry operations
func TestConfigProviderRegistry_ErrorConditions(t *testing.T) {
	t.Run("RegisterGlobal_EmptyName", func(t *testing.T) {
		creator := func(options config.ProviderOptions) (iface.Provider, error) {
			return config.NewAdvancedMockConfigProvider("test", "test"), nil
		}

		err := config.RegisterGlobal("", creator)
		assert.Error(t, err, "Should fail with empty provider name")
		if configErr, ok := config.AsConfigError(err); ok {
			assert.Equal(t, config.ErrCodeInvalidProviderName, configErr.GetCode())
		}
	})

	t.Run("RegisterGlobal_NilCreator", func(t *testing.T) {
		err := config.RegisterGlobal("nil-creator-test", nil)
		assert.Error(t, err, "Should fail with nil creator")
		if configErr, ok := config.AsConfigError(err); ok {
			assert.Equal(t, config.ErrCodeProviderCreationFailed, configErr.GetCode())
		}
	})

	t.Run("NewRegistryProvider_EmptyName", func(t *testing.T) {
		options := config.ProviderOptions{
			ProviderType: "test",
			ConfigName:   "test",
			ConfigPaths:  []string{"./test"},
		}

		provider, err := config.NewRegistryProvider(context.Background(), "", options)
		assert.Error(t, err, "Should fail with empty provider name")
		assert.Nil(t, provider)
		if configErr, ok := config.AsConfigError(err); ok {
			assert.Equal(t, config.ErrCodeInvalidProviderName, configErr.GetCode())
		}
	})

	t.Run("GetProviderMetadata_NotFound", func(t *testing.T) {
		metadata, err := config.GetProviderMetadata("nonexistent-metadata-provider")
		assert.Error(t, err, "Should fail for nonexistent provider metadata")
		assert.Nil(t, metadata)
		if configErr, ok := config.AsConfigError(err); ok {
			assert.Equal(t, config.ErrCodeProviderNotFound, configErr.GetCode())
		}
	})

	t.Run("UnregisterProvider_NotFound", func(t *testing.T) {
		err := config.UnregisterProvider("nonexistent-unregister-provider")
		assert.Error(t, err, "Should fail for nonexistent provider unregistration")
		if configErr, ok := config.AsConfigError(err); ok {
			assert.Equal(t, config.ErrCodeProviderNotFound, configErr.GetCode())
		}
	})
}
