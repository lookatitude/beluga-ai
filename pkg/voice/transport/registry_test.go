package transport

import (
	"testing"

	"github.com/lookatitude/beluga-ai/pkg/voice/transport/iface"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistry_Register(t *testing.T) {
	registry := GetRegistry()

	factory := func(config *Config) (iface.Transport, error) {
		return NewAdvancedMockTransport("test"), nil
	}

	registry.Register("test-provider", factory)
	assert.True(t, registry.IsRegistered("test-provider"))
}

func TestRegistry_GetProvider(t *testing.T) {
	registry := GetRegistry()

	factory := func(config *Config) (iface.Transport, error) {
		return NewAdvancedMockTransport("test"), nil
	}

	registry.Register("test-provider", factory)

	config := DefaultConfig()
	config.URL = "wss://example.com"
	provider, err := registry.GetProvider("test-provider", config)
	require.NoError(t, err)
	assert.NotNil(t, provider)
}

func TestRegistry_GetProvider_NotFound(t *testing.T) {
	registry := GetRegistry()
	config := DefaultConfig()
	config.URL = "wss://example.com"

	provider, err := registry.GetProvider("non-existent", config)
	assert.Error(t, err)
	assert.Nil(t, provider)
	assert.Contains(t, err.Error(), "not registered")
}

func TestRegistry_ListProviders(t *testing.T) {
	registry := GetRegistry()

	factory := func(config *Config) (iface.Transport, error) {
		return NewAdvancedMockTransport("test"), nil
	}

	registry.Register("provider1", factory)
	registry.Register("provider2", factory)

	providers := registry.ListProviders()
	assert.Contains(t, providers, "provider1")
	assert.Contains(t, providers, "provider2")
}

func TestRegistry_IsRegistered(t *testing.T) {
	registry := GetRegistry()

	factory := func(config *Config) (iface.Transport, error) {
		return NewAdvancedMockTransport("test"), nil
	}

	assert.False(t, registry.IsRegistered("new-provider"))
	registry.Register("new-provider", factory)
	assert.True(t, registry.IsRegistered("new-provider"))
}
