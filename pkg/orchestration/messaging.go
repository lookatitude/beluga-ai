package orchestration

import (
	"encoding/json"
	"fmt"
	"log"
	"time"
)

// Message represents the structure of inter-agent messages.
type Message struct {
	ID        string                 `json:"id"`
	Timestamp time.Time              `json:"timestamp"`
	Sender    string                 `json:"sender"`
	Receiver  string                 `json:"receiver"`
	Type      string                 `json:"type"`
	Payload   map[string]interface{} `json:"payload"`
}

// MessagingSystem handles inter-agent communication.
type MessagingSystem struct {
	messages chan Message
}

// NewMessagingSystem initializes a new messaging system.
func NewMessagingSystem(bufferSize int) *MessagingSystem {
	return &MessagingSystem{
		messages: make(chan Message, bufferSize),
	}
}

// SendMessage sends a message to the messaging system.
func (ms *MessagingSystem) SendMessage(msg Message) error {
	select {
	case ms.messages <- msg:
		log.Printf("Message sent: %v", msg)
		return nil
	default:
		return fmt.Errorf("message queue is full")
	}
}

// SendMessageWithRetry sends a message with retry logic.
func (ms *MessagingSystem) SendMessageWithRetry(msg Message, retries int, backoff time.Duration) error {
	for i := 0; i <= retries; i++ {
		if err := ms.SendMessage(msg); err != nil {
			log.Printf("Failed to send message (attempt %d): %v", i+1, err)
			time.Sleep(backoff)
			backoff *= 2 // Exponential backoff
		} else {
			return nil
		}
	}
	return fmt.Errorf("failed to send message after %d retries", retries)
}

// ReceiveMessage receives a message from the messaging system.
func (ms *MessagingSystem) ReceiveMessage() (Message, error) {
	select {
	case msg := <-ms.messages:
		log.Printf("Message received: %v", msg)
		return msg, nil
	case <-time.After(5 * time.Second):
		return Message{}, fmt.Errorf("no messages available")
	}
}

// ValidateMessage validates the structure of a message.
func ValidateMessage(msg Message) error {
	if msg.ID == "" || msg.Sender == "" || msg.Receiver == "" || msg.Type == "" {
		return fmt.Errorf("invalid message: missing required fields")
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