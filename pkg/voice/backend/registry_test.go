package backend

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	vbiface "github.com/lookatitude/beluga-ai/pkg/voice/backend/iface"
)

// mockVoiceBackend is a mock implementation for testing.
type mockVoiceBackend struct {
	config   *vbiface.Config
	provider string
}

func (m *mockVoiceBackend) Start(ctx context.Context) error {
	return nil
}

func (m *mockVoiceBackend) Stop(ctx context.Context) error {
	return nil
}

func (m *mockVoiceBackend) CreateSession(ctx context.Context, config *vbiface.SessionConfig) (vbiface.VoiceSession, error) {
	return nil, nil
}

func (m *mockVoiceBackend) GetSession(ctx context.Context, sessionID string) (vbiface.VoiceSession, error) {
	return nil, nil
}

func (m *mockVoiceBackend) ListSessions(ctx context.Context) ([]vbiface.VoiceSession, error) {
	return nil, nil
}

func (m *mockVoiceBackend) CloseSession(ctx context.Context, sessionID string) error {
	return nil
}

func (m *mockVoiceBackend) HealthCheck(ctx context.Context) (*vbiface.HealthStatus, error) {
	return &vbiface.HealthStatus{Status: "healthy"}, nil
}

func (m *mockVoiceBackend) GetConnectionState() vbiface.ConnectionState {
	return vbiface.ConnectionStateConnected
}

func (m *mockVoiceBackend) GetActiveSessionCount() int {
	return 0
}

func (m *mockVoiceBackend) GetConfig() *vbiface.Config {
	return m.config
}

func (m *mockVoiceBackend) UpdateConfig(ctx context.Context, config *vbiface.Config) error {
	m.config = config
	return nil
}

func newTestRegistry() *BackendRegistry {
	return &BackendRegistry{
		creators: make(map[string]func(context.Context, *vbiface.Config) (vbiface.VoiceBackend, error)),
	}
}

// validTestConfig returns a valid config for testing.
func validTestConfig() *vbiface.Config {
	return &vbiface.Config{
		Provider:      "mock",
		PipelineType:  vbiface.PipelineTypeSTTTTS,
		STTProvider:   "openai",
		TTSProvider:   "openai",
		LatencyTarget: 500 * time.Millisecond,
		Timeout:       30 * time.Second,
		RetryDelay:    time.Second,
	}
}

func TestGetRegistry(t *testing.T) {
	// GetRegistry should return the same instance
	reg1 := GetRegistry()
	reg2 := GetRegistry()

	assert.NotNil(t, reg1)
	assert.Same(t, reg1, reg2)
}

func TestRegistryRegister(t *testing.T) {
	reg := newTestRegistry()

	creator := func(ctx context.Context, cfg *vbiface.Config) (vbiface.VoiceBackend, error) {
		return &mockVoiceBackend{provider: "test", config: cfg}, nil
	}

	reg.Register("test-provider", creator)

	assert.True(t, reg.IsRegistered("test-provider"))
	assert.False(t, reg.IsRegistered("nonexistent"))
}

func TestRegistryCreate(t *testing.T) {
	reg := newTestRegistry()

	creator := func(ctx context.Context, cfg *vbiface.Config) (vbiface.VoiceBackend, error) {
		return &mockVoiceBackend{provider: cfg.Provider, config: cfg}, nil
	}

	reg.Register("mock", creator)

	t.Run("successful creation", func(t *testing.T) {
		config := validTestConfig()

		backend, err := reg.Create(context.Background(), "mock", config)

		require.NoError(t, err)
		require.NotNil(t, backend)
	})

	t.Run("provider not found", func(t *testing.T) {
		config := validTestConfig()

		_, err := reg.Create(context.Background(), "nonexistent", config)

		require.Error(t, err)
		assert.True(t, IsError(err))
		backendErr := AsError(err)
		assert.Equal(t, ErrCodeProviderNotFound, backendErr.Code)
	})

	t.Run("invalid config", func(t *testing.T) {
		config := &vbiface.Config{
			PipelineType:  vbiface.PipelineTypeSTTTTS,
			LatencyTarget: 500 * time.Millisecond,
			Timeout:       30 * time.Second,
			// Missing required STT/TTS providers
		}

		_, err := reg.Create(context.Background(), "mock", config)

		require.Error(t, err)
		assert.True(t, IsError(err))
		backendErr := AsError(err)
		assert.Equal(t, ErrCodeInvalidConfig, backendErr.Code)
	})
}

func TestRegistryListProviders(t *testing.T) {
	reg := newTestRegistry()

	// Empty registry
	providers := reg.ListProviders()
	assert.Empty(t, providers)

	// Add providers
	creator := func(ctx context.Context, cfg *vbiface.Config) (vbiface.VoiceBackend, error) {
		return &mockVoiceBackend{config: cfg}, nil
	}

	reg.Register("provider1", creator)
	reg.Register("provider2", creator)
	reg.Register("provider3", creator)

	providers = reg.ListProviders()
	assert.Len(t, providers, 3)
	assert.Contains(t, providers, "provider1")
	assert.Contains(t, providers, "provider2")
	assert.Contains(t, providers, "provider3")
}

func TestRegistryIsRegistered(t *testing.T) {
	reg := newTestRegistry()

	creator := func(ctx context.Context, cfg *vbiface.Config) (vbiface.VoiceBackend, error) {
		return &mockVoiceBackend{config: cfg}, nil
	}

	assert.False(t, reg.IsRegistered("test"))

	reg.Register("test", creator)

	assert.True(t, reg.IsRegistered("test"))
	assert.False(t, reg.IsRegistered("other"))
}

func TestRegistryGetProvider(t *testing.T) {
	reg := newTestRegistry()

	creator := func(ctx context.Context, cfg *vbiface.Config) (vbiface.VoiceBackend, error) {
		return &mockVoiceBackend{config: cfg}, nil
	}

	reg.Register("test", creator)

	t.Run("existing provider", func(t *testing.T) {
		factory, err := reg.GetProvider("test")
		require.NoError(t, err)
		require.NotNil(t, factory)
	})

	t.Run("nonexistent provider", func(t *testing.T) {
		_, err := reg.GetProvider("nonexistent")
		require.Error(t, err)
		assert.True(t, IsError(err))
		backendErr := AsError(err)
		assert.Equal(t, ErrCodeProviderNotFound, backendErr.Code)
	})
}

func TestRegistryConcurrency(t *testing.T) {
	reg := newTestRegistry()

	creator := func(ctx context.Context, cfg *vbiface.Config) (vbiface.VoiceBackend, error) {
		return &mockVoiceBackend{config: cfg}, nil
	}

	var wg sync.WaitGroup
	numGoroutines := 100

	// Concurrent registrations
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			reg.Register("concurrent-provider", creator)
		}(i)
	}

	wg.Wait()

	// Verify registration
	assert.True(t, reg.IsRegistered("concurrent-provider"))

	// Concurrent reads
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = reg.ListProviders()
			_ = reg.IsRegistered("concurrent-provider")
		}()
	}

	wg.Wait()
}

func TestRegistryProviderOverwrite(t *testing.T) {
	reg := newTestRegistry()

	creator1 := func(ctx context.Context, cfg *vbiface.Config) (vbiface.VoiceBackend, error) {
		return &mockVoiceBackend{provider: "v1", config: cfg}, nil
	}

	creator2 := func(ctx context.Context, cfg *vbiface.Config) (vbiface.VoiceBackend, error) {
		return &mockVoiceBackend{provider: "v2", config: cfg}, nil
	}

	reg.Register("test", creator1)
	reg.Register("test", creator2)

	// Should use the latest registration
	config := validTestConfig()

	backend, err := reg.Create(context.Background(), "test", config)
	require.NoError(t, err)
	require.NotNil(t, backend)

	mock, ok := backend.(*mockVoiceBackend)
	require.True(t, ok)
	assert.Equal(t, "v2", mock.provider)
}
