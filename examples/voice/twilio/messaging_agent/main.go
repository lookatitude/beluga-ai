package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/messaging"
	"github.com/lookatitude/beluga-ai/pkg/messaging/iface"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		cancel()
	}()

	// Configure Twilio messaging backend
	config := &messaging.Config{
		Provider: "twilio",
		ProviderSpecific: map[string]any{
			"account_sid": os.Getenv("TWILIO_ACCOUNT_SID"),
			"auth_token":  os.Getenv("TWILIO_AUTH_TOKEN"),
			"webhook_url": os.Getenv("TWILIO_WEBHOOK_URL"),
		},
		Timeout: 30 * time.Second,
	}

	// Create backend
	messagingBackend, err := messaging.NewBackend(ctx, "twilio", config)
	if err != nil {
		log.Fatalf("Failed to create messaging backend: %v", err)
	}

	// Start backend
	if err := messagingBackend.Start(ctx); err != nil {
		log.Fatalf("Failed to start messaging backend: %v", err)
	}
	defer messagingBackend.Stop(ctx)

	log.Println("Twilio messaging backend started. Waiting for messages...")

	// Agent callback function
	agentCallback := func(ctx context.Context, message string) (string, error) {
		// Simple echo agent - replace with your agent logic
		return fmt.Sprintf("You said: %s", message), nil
	}

	// Example: Create a conversation
	conversation, err := messagingBackend.CreateConversation(ctx, &iface.ConversationConfig{
		FriendlyName: "Customer Support",
	})
	if err != nil {
		log.Fatalf("Failed to create conversation: %v", err)
	}

	log.Printf("Conversation created: %s", conversation.ConversationSID)

	// Example: Send a message
	err = messagingBackend.SendMessage(ctx, conversation.ConversationSID, &iface.Message{
		Body:    "Hello! How can I help you?",
		Channel: iface.ChannelSMS,
	})
	if err != nil {
		log.Fatalf("Failed to send message: %v", err)
	}

	// Example: Receive messages
	messageCh, err := messagingBackend.ReceiveMessages(ctx, conversation.ConversationSID)
	if err != nil {
		log.Fatalf("Failed to receive messages: %v", err)
	}

	// Process incoming messages
	go func() {
		for message := range messageCh {
			log.Printf("Received message: %s", message.Body)

			// Process through agent
			response, err := agentCallback(ctx, message.Body)
			if err != nil {
				log.Printf("Agent error: %v", err)
				continue
			}

			// Send response
			err = messagingBackend.SendMessage(ctx, conversation.ConversationSID, &iface.Message{
				Body:    response,
				Channel: message.Channel,
			})
			if err != nil {
				log.Printf("Failed to send response: %v", err)
			}
		}
	}()

	// Wait for context cancellation
	<-ctx.Done()
	log.Println("Shutting down...")
}
