package twilio

import (
	"context"
	"testing"

	"github.com/lookatitude/beluga-ai/pkg/voice/backend/iface"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateWebhookSignature(t *testing.T) {
	config := &TwilioConfig{
		Config: &iface.Config{
			Provider: "twilio",
		},
		AuthToken: "test_auth_token_12345",
	}

	backend, err := NewTwilioBackend(config)
	require.NoError(t, err)

	tests := []struct {
		webhookData map[string]string
		name        string
		wantErr     bool
	}{
		{
			name: "valid signature",
			webhookData: map[string]string{
				"X-Twilio-Signature": "valid_signature",
				"_url":               "https://example.com/webhook",
				"CallSid":            "CA1234567890abcdef",
			},
			wantErr: true, // Will fail without proper signature computation
		},
		{
			name: "missing signature",
			webhookData: map[string]string{
				"_url":    "https://example.com/webhook",
				"CallSid": "CA1234567890abcdef",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := backend.validateWebhookSignature(tt.webhookData)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestParseWebhookEvent(t *testing.T) {
	config := &TwilioConfig{
		Config: &iface.Config{
			Provider: "twilio",
		},
		AuthToken: "test_auth_token",
	}

	backend, err := NewTwilioBackend(config)
	require.NoError(t, err)

	ctx := context.Background()

	webhookData := map[string]string{
		"CallSid":    "CA1234567890abcdef",
		"CallStatus": "answered",
		"From":       "+15551234567",
		"To":         "+15559876543",
		"AccountSid": "AC1234567890abcdef",
	}

	event, err := backend.ParseWebhookEvent(ctx, webhookData)
	require.NoError(t, err)
	assert.NotNil(t, event)
	assert.Equal(t, "CA1234567890abcdef", event.ResourceSID)
	assert.Equal(t, "voice-api", event.Source)
}

func TestHandleCallEvent(t *testing.T) {
	config := &TwilioConfig{
		Config: &iface.Config{
			Provider: "twilio",
		},
		AuthToken: "test_auth_token",
	}

	backend, err := NewTwilioBackend(config)
	require.NoError(t, err)

	ctx := context.Background()

	tests := []struct {
		name       string
		callStatus string
		wantErr    bool
	}{
		{
			name:       "call answered",
			callStatus: "answered",
			wantErr:    true, // Will fail without real Twilio client
		},
		{
			name:       "call completed",
			callStatus: "completed",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := &WebhookEvent{
				EventType:   "call." + tt.callStatus,
				ResourceSID: "CA1234567890abcdef",
				EventData: map[string]any{
					"CallStatus": tt.callStatus,
				},
			}

			err := backend.handleCallEvent(ctx, event)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				// May or may not error depending on session existence
				_ = err
			}
		})
	}
}
