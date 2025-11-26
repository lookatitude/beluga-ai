package messagebus

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"
)

// Note: Message type is defined in messagebus.go to avoid duplication

// MessagingSystem handles inter-agent communication using the MessageBus interface.
type MessagingSystem struct {
	messageBus MessageBus
}

// NewMessagingSystem initializes a new messaging system with a MessageBus.
func NewMessagingSystem(messageBus MessageBus) *MessagingSystem {
	return &MessagingSystem{
		messageBus: messageBus,
	}
}

// SendMessage sends a message to the messaging system.
func (ms *MessagingSystem) SendMessage(msg Message) error {
	// Convert Message to MessageBus format
	return ms.messageBus.Publish(context.Background(), msg.Topic, msg.Payload, msg.Metadata)
}

// SendMessageWithRetry sends a message with retry logic.
func (ms *MessagingSystem) SendMessageWithRetry(msg Message, retries int, backoff time.Duration) error {
	// Ensure at least one attempt is made, even if retries is negative
	maxAttempts := retries + 1
	if maxAttempts < 1 {
		maxAttempts = 1
	}

	for i := 0; i < maxAttempts; i++ {
		if err := ms.SendMessage(msg); err != nil {
			if i < maxAttempts-1 {
				log.Printf("Failed to send message (attempt %d): %v", i+1, err)
				time.Sleep(backoff)
				backoff *= 2 // Exponential backoff
			}
		} else {
			return nil
		}
	}
	return fmt.Errorf("failed to send message after %d retries", retries)
}

// ReceiveMessage receives a message from the messaging system.
// Note: This is a simplified implementation - in practice you'd need a subscription.
func (ms *MessagingSystem) ReceiveMessage() (Message, error) {
	return Message{}, errors.New("ReceiveMessage not implemented - use MessageBus subscription")
}

// ValidateMessage validates the structure of a message.
func ValidateMessage(msg Message) error {
	if msg.ID == "" || msg.Topic == "" {
		return errors.New("invalid message: missing required fields")
	}
	// Check for whitespace-only strings
	if strings.TrimSpace(msg.ID) == "" || strings.TrimSpace(msg.Topic) == "" {
		return errors.New("invalid message: ID and Topic cannot be whitespace-only")
	}
	return nil
}

// SerializeMessage serializes a message to JSON.
func SerializeMessage(msg Message) (string, error) {
	data, err := json.Marshal(msg)
	if err != nil {
		return "", fmt.Errorf("failed to serialize message: %w", err)
	}
	return string(data), nil
}

// DeserializeMessage deserializes a JSON string to a Message.
func DeserializeMessage(data string) (Message, error) {
	var msg Message
	if err := json.Unmarshal([]byte(data), &msg); err != nil {
		return Message{}, fmt.Errorf("failed to deserialize message: %w", err)
	}
	return msg, nil
}
