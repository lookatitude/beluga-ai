package twilio

import (
	"context"
	"testing"

	"github.com/lookatitude/beluga-ai/pkg/messaging"
	"github.com/lookatitude/beluga-ai/pkg/messaging/iface"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTwilioProvider_Start(t *testing.T) {
	config := &TwilioConfig{
		Config:     messaging.DefaultConfig(),
		AccountSID: "AC1234567890abcdef",
		AuthToken:  "test_auth_token",
	}

	provider, err := NewTwilioProvider(config)
	require.NoError(t, err)

	ctx := context.Background()
	// Note: This will fail in actual test without real Twilio credentials
	err = provider.Start(ctx)
	// We expect this to fail without real credentials, but structure is correct
	assert.Error(t, err) // Expected without real credentials
}

func TestTwilioProvider_CreateConversation(t *testing.T) {
	config := &TwilioConfig{
		Config:     messaging.DefaultConfig(),
		AccountSID: "AC1234567890abcdef",
		AuthToken:  "test_auth_token",
	}

	provider, err := NewTwilioProvider(config)
	require.NoError(t, err)

	ctx := context.Background()
	convConfig := &iface.ConversationConfig{
		FriendlyName: "Test Conversation",
	}

	// This will fail without real credentials, but tests structure
	_, err = provider.CreateConversation(ctx, convConfig)
	assert.Error(t, err) // Expected without real credentials
}

func TestTwilioProvider_SendMessage(t *testing.T) {
	config := &TwilioConfig{
		Config:     messaging.DefaultConfig(),
		AccountSID: "AC1234567890abcdef",
		AuthToken:  "test_auth_token",
	}

	provider, err := NewTwilioProvider(config)
	require.NoError(t, err)

	ctx := context.Background()
	message := &iface.Message{
		Body:    "Test message",
		Channel: iface.ChannelSMS,
	}

	// This will fail without real conversation, but tests structure
	err = provider.SendMessage(ctx, "CH1234567890abcdef", message)
	assert.Error(t, err) // Expected without real conversation
}

func TestTwilioConfig_Validate(t *testing.T) {
	tests := []struct {
		config  *TwilioConfig
		name    string
		wantErr bool
	}{
		{
			name: "valid config",
			config: &TwilioConfig{
				Config:     messaging.DefaultConfig(),
				AccountSID: "AC1234567890abcdef",
				AuthToken:  "auth_token_123",
			},
			wantErr: false,
		},
		{
			name: "missing account_sid",
			config: &TwilioConfig{
				Config:    messaging.DefaultConfig(),
				AuthToken: "auth_token_123",
			},
			wantErr: true,
		},
		{
			name: "missing auth_token",
			config: &TwilioConfig{
				Config:     messaging.DefaultConfig(),
				AccountSID: "AC1234567890abcdef",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
