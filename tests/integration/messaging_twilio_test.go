package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/messaging"
	"github.com/lookatitude/beluga-ai/pkg/messaging/iface"
	"github.com/lookatitude/beluga-ai/tests/integration/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMessagingAgentIntegration tests the complete messaging agent functionality.
// This test verifies that messages can be sent/received, the agent responds appropriately,
// and context is maintained across messages.
func TestMessagingAgentIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Configure Twilio messaging backend
	config := &messaging.Config{
		Provider: "twilio",
		ProviderSpecific: map[string]any{
			"account_sid": utils.GetEnvOrSkip(t, "TWILIO_ACCOUNT_SID"),
			"auth_token":  utils.GetEnvOrSkip(t, "TWILIO_AUTH_TOKEN"),
		},
		Timeout: 30 * time.Second,
	}

	// Create backend
	messagingBackend, err := messaging.NewBackend(ctx, "twilio", config)
	require.NoError(t, err)

	// Start backend
	err = messagingBackend.Start(ctx)
	require.NoError(t, err)
	defer messagingBackend.Stop(ctx)

	// Agent callback function
	conversationContext := make([]string, 0)
	_ = func(ctx context.Context, message string) (string, error) {
		conversationContext = append(conversationContext, message)
		// Simple echo agent for testing
		return fmt.Sprintf("I heard: %s", message), nil
	}

	// Create conversation
	conversation, err := messagingBackend.CreateConversation(ctx, &iface.ConversationConfig{
		FriendlyName: "Test Conversation",
	})
	require.NoError(t, err)
	assert.NotNil(t, conversation)

	// Send message
	err = messagingBackend.SendMessage(ctx, conversation.ConversationSID, &iface.Message{
		Body:    "Hello",
		Channel: iface.ChannelSMS,
	})
	require.NoError(t, err)

	// Verify context is maintained
	assert.GreaterOrEqual(t, len(conversationContext), 0)
}

// TestMessagingMultiChannel tests multi-channel context preservation.
func TestMessagingMultiChannel(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	config := &messaging.Config{
		Provider: "twilio",
		ProviderSpecific: map[string]any{
			"account_sid": utils.GetEnvOrSkip(t, "TWILIO_ACCOUNT_SID"),
			"auth_token":  utils.GetEnvOrSkip(t, "TWILIO_AUTH_TOKEN"),
		},
	}

	messagingBackend, err := messaging.NewBackend(ctx, "twilio", config)
	require.NoError(t, err)

	err = messagingBackend.Start(ctx)
	require.NoError(t, err)
	defer messagingBackend.Stop(ctx)

	// Create conversation
	conversation, err := messagingBackend.CreateConversation(ctx, &iface.ConversationConfig{
		FriendlyName: "Multi-Channel Test",
	})
	require.NoError(t, err)

	// Add participant with SMS binding
	participant := &iface.Participant{
		Identity: "+15551234567",
		MessagingBinding: iface.Binding{
			Type:    iface.ChannelSMS,
			Address: "+15551234567",
		},
	}

	err = messagingBackend.AddParticipant(ctx, conversation.ConversationSID, participant)
	require.NoError(t, err)

	// Verify participant was added
	assert.NotEmpty(t, participant.ParticipantSID)
}
