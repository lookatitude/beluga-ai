package twilio

import (
	"context"
	"testing"

	"github.com/lookatitude/beluga-ai/pkg/messaging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTwilioProvider_ValidateWebhookSignature(t *testing.T) {
	config := &TwilioConfig{
		Config:    messaging.DefaultConfig(),
		AuthToken: "test_auth_token_12345",
	}

	provider, err := NewTwilioProvider(config)
	require.NoError(t, err)

	tests := []struct {
		webhookData map[string]string
		name        string
		wantErr     bool
	}{
		{
			name: "missing signature",
			webhookData: map[string]string{
				"_url": "https://example.com/webhook",
			},
			wantErr: true,
		},
		{
			name: "valid signature structure",
			webhookData: map[string]string{
				"X-Twilio-Signature": "valid_signature",
				"_url":               "https://example.com/webhook",
				"ConversationSid":    "CH1234567890abcdef",
			},
			wantErr: true, // Will fail without proper signature computation
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := provider.validateWebhookSignature(tt.webhookData)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestTwilioProvider_ParseWebhookEvent(t *testing.T) {
	config := &TwilioConfig{
		Config:    messaging.DefaultConfig(),
		AuthToken: "test_auth_token",
	}

	provider, err := NewTwilioProvider(config)
	require.NoError(t, err)

	ctx := context.Background()

	webhookData := map[string]string{
		"MessageSid":      "IM1234567890abcdef",
		"ConversationSid": "CH1234567890abcdef",
		"Body":            "Test message",
		"Author":          "+15551234567",
		"AccountSid":      "AC1234567890abcdef",
	}

	event, err := provider.ParseWebhookEvent(ctx, webhookData)
	require.NoError(t, err)
	assert.NotNil(t, event)
	assert.Equal(t, "IM1234567890abcdef", event.ResourceSID)
	assert.Equal(t, "conversations-api", event.Source)
}
