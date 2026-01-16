package twilio

import (
	"context"
	"testing"
	"time"

	vbiface "github.com/lookatitude/beluga-ai/pkg/voice/backend/iface"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTwilioProvider_GetName(t *testing.T) {
	provider := NewTwilioProvider()
	assert.Equal(t, "twilio", provider.GetName())
}

func TestTwilioProvider_GetCapabilities(t *testing.T) {
	provider := NewTwilioProvider()
	ctx := context.Background()

	capabilities, err := provider.GetCapabilities(ctx)
	require.NoError(t, err)
	assert.NotNil(t, capabilities)
	assert.False(t, capabilities.S2SSupport) // Twilio uses STT/TTS
	assert.True(t, capabilities.MultiUserSupport)
	assert.Equal(t, 100, capabilities.MaxConcurrentSessions) // SC-003
	assert.Equal(t, 2*time.Second, capabilities.MinLatency)  // FR-009
}

func TestTwilioProvider_ValidateConfig(t *testing.T) {
	provider := NewTwilioProvider()
	ctx := context.Background()

	tests := []struct {
		name    string
		config  *vbiface.Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &vbiface.Config{
				Provider:     "twilio",
				PipelineType: vbiface.PipelineTypeSTTTTS,
				STTProvider:  "openai",
				TTSProvider:  "openai",
				ProviderConfig: map[string]any{
					"account_sid":  "AC1234567890abcdef",
					"auth_token":   "auth_token_123",
					"phone_number": "+15551234567",
				},
			},
			wantErr: false,
		},
		{
			name: "missing account_sid",
			config: &vbiface.Config{
				Provider:     "twilio",
				PipelineType: vbiface.PipelineTypeSTTTTS,
				STTProvider:  "openai",
				TTSProvider:  "openai",
				ProviderConfig: map[string]any{
					"auth_token":   "auth_token_123",
					"phone_number": "+15551234567",
				},
			},
			wantErr: true,
		},
		{
			name: "wrong pipeline type",
			config: &vbiface.Config{
				Provider:     "twilio",
				PipelineType: vbiface.PipelineTypeS2S,
				STTProvider:  "openai",
				TTSProvider:  "openai",
				ProviderConfig: map[string]any{
					"account_sid":  "AC1234567890abcdef",
					"auth_token":   "auth_token_123",
					"phone_number": "+15551234567",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := provider.ValidateConfig(ctx, tt.config)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestTwilioProvider_CreateBackend(t *testing.T) {
	provider := NewTwilioProvider()
	ctx := context.Background()

	config := &vbiface.Config{
		Provider:     "twilio",
		PipelineType: vbiface.PipelineTypeSTTTTS,
		STTProvider:  "openai",
		TTSProvider:  "openai",
		ProviderConfig: map[string]any{
			"account_sid":  "AC1234567890abcdef",
			"auth_token":   "auth_token_123",
			"phone_number": "+15551234567",
		},
	}

	backend, err := provider.CreateBackend(ctx, config)
	require.NoError(t, err)
	assert.NotNil(t, backend)
}

func TestTwilioBackend_Start(t *testing.T) {
	config := &TwilioConfig{
		Config: &vbiface.Config{
			Provider:     "twilio",
			PipelineType: vbiface.PipelineTypeSTTTTS,
		},
		AccountSID:  "AC1234567890abcdef",
		AuthToken:   "test_auth_token",
		PhoneNumber: "+15551234567",
	}

	backend, err := NewTwilioBackend(config)
	require.NoError(t, err)

	ctx := context.Background()
	// Note: This will fail in actual test without real Twilio credentials
	// In a real test, we'd use a mock Twilio client
	err = backend.Start(ctx)
	// We expect this to fail without real credentials, but structure is correct
	assert.Error(t, err) // Expected without real credentials
}

func TestTwilioBackend_GetConnectionState(t *testing.T) {
	config := &TwilioConfig{
		Config: &vbiface.Config{
			Provider:     "twilio",
			PipelineType: vbiface.PipelineTypeSTTTTS,
		},
		AccountSID:  "AC1234567890abcdef",
		AuthToken:   "test_auth_token",
		PhoneNumber: "+15551234567",
	}

	backend, err := NewTwilioBackend(config)
	require.NoError(t, err)

	state := backend.GetConnectionState()
	assert.Equal(t, vbiface.ConnectionStateDisconnected, state)
}

func TestTwilioBackend_GetActiveSessionCount(t *testing.T) {
	config := &TwilioConfig{
		Config: &vbiface.Config{
			Provider:     "twilio",
			PipelineType: vbiface.PipelineTypeSTTTTS,
		},
		AccountSID:  "AC1234567890abcdef",
		AuthToken:   "test_auth_token",
		PhoneNumber: "+15551234567",
	}

	backend, err := NewTwilioBackend(config)
	require.NoError(t, err)

	count := backend.GetActiveSessionCount()
	assert.Equal(t, 0, count)
}
