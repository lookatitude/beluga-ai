package internal

import (
	"errors"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/schema/iface"
)

// BaseChatHistory provides a basic implementation of the ChatHistory interface.
// It stores messages in memory.
type BaseChatHistory struct {
	config   *ChatHistoryConfig
	messages []iface.Message
}

// NewBaseChatHistory creates a new BaseChatHistory.
func NewBaseChatHistory(config *ChatHistoryConfig) *BaseChatHistory {
	return &BaseChatHistory{
		config:   config,
		messages: make([]iface.Message, 0),
	}
}

// AddMessage adds a message to the history.
func (h *BaseChatHistory) AddMessage(message iface.Message) error {
	// Check max messages limit
	if h.config != nil && h.config.MaxMessages > 0 && len(h.messages) >= h.config.MaxMessages {
		// Remove oldest message
		h.messages = h.messages[1:]
	}

	h.messages = append(h.messages, message)
	return nil
}

// AddUserMessage adds a user message to the history.
func (h *BaseChatHistory) AddUserMessage(message string) error {
	return h.AddMessage(&ChatMessage{
		BaseMessage: BaseMessage{Content: message},
		Role:        RoleHuman,
	})
}

// AddAIMessage adds an AI message to the history.
func (h *BaseChatHistory) AddAIMessage(message string) error {
	return h.AddMessage(&AIMessage{
		BaseMessage: BaseMessage{Content: message},
	})
}

// Messages returns all messages in the history.
func (h *BaseChatHistory) Messages() ([]iface.Message, error) {
	return h.messages, nil
}

// Clear removes all messages from the history.
func (h *BaseChatHistory) Clear() error {
	h.messages = make([]iface.Message, 0)
	return nil
}

// ChatHistoryConfig defines configuration options for chat history implementations.
type ChatHistoryConfig struct {
	// MaxMessages limits the number of messages stored in history (0 = unlimited)
	MaxMessages int `yaml:"max_messages,omitempty" json:"max_messages,omitempty" validate:"min=0"`

	// TTL defines how long messages should be kept (0 = forever)
	TTL time.Duration `yaml:"ttl,omitempty" json:"ttl,omitempty"`

	// EnablePersistence determines if history should be persisted
	EnablePersistence bool `yaml:"enable_persistence,omitempty" json:"enable_persistence,omitempty"`
}

// Validate validates the ChatHistoryConfig struct.
func (c *ChatHistoryConfig) Validate() error {
	if c.MaxMessages < 0 {
		return errors.New("max_messages cannot be negative")
	}
	return nil
}
