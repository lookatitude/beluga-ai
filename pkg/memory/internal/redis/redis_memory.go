// Package redis provides Redis-backed memory implementations.
// It stores conversation history and context in Redis for distributed access.
package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/memory/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
)

// RedisMemory is a memory implementation that stores messages in Redis.
// It provides distributed memory access suitable for multi-instance deployments.
type RedisMemory struct {
	client      RedisClient
	sessionID   string
	memoryKey   string
	inputKey    string
	outputKey   string
	humanPrefix string
	aiPrefix    string
	ttl         time.Duration
}

// RedisClient defines the interface for Redis operations.
// This allows for dependency injection and testing.
type RedisClient interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key, value string, ttl time.Duration) error
	Del(ctx context.Context, keys ...string) error
	Exists(ctx context.Context, key string) (bool, error)
	Expire(ctx context.Context, key string, ttl time.Duration) error
}

// RedisConfig holds configuration for RedisMemory.
type RedisConfig struct {
	Client      RedisClient
	SessionID   string
	MemoryKey   string
	InputKey    string
	OutputKey   string
	HumanPrefix string
	AIPrefix    string
	TTL         time.Duration
}

// NewRedisMemory creates a new Redis-backed memory instance.
func NewRedisMemory(config *RedisConfig) (*RedisMemory, error) {
	if config == nil {
		return nil, errors.New("config cannot be nil")
	}

	if config.Client == nil {
		return nil, errors.New("redis client is required")
	}

	if config.SessionID == "" {
		return nil, errors.New("session ID is required")
	}

	memoryKey := config.MemoryKey
	if memoryKey == "" {
		memoryKey = "history"
	}

	inputKey := config.InputKey
	if inputKey == "" {
		inputKey = "input"
	}

	outputKey := config.OutputKey
	if outputKey == "" {
		outputKey = "output"
	}

	ttl := config.TTL
	if ttl == 0 {
		ttl = 24 * time.Hour // Default TTL of 24 hours
	}

	humanPrefix := config.HumanPrefix
	if humanPrefix == "" {
		humanPrefix = "Human"
	}

	aiPrefix := config.AIPrefix
	if aiPrefix == "" {
		aiPrefix = "AI"
	}

	return &RedisMemory{
		client:      config.Client,
		sessionID:   config.SessionID,
		memoryKey:   memoryKey,
		inputKey:    inputKey,
		outputKey:   outputKey,
		ttl:         ttl,
		humanPrefix: humanPrefix,
		aiPrefix:    aiPrefix,
	}, nil
}

// MemoryVariables returns the variables exposed by this memory implementation.
func (m *RedisMemory) MemoryVariables() []string {
	return []string{m.memoryKey}
}

// LoadMemoryVariables loads messages from Redis.
func (m *RedisMemory) LoadMemoryVariables(ctx context.Context, _ map[string]any) (map[string]any, error) {
	key := m.getRedisKey(m.memoryKey)

	// Check if key exists
	exists, err := m.client.Exists(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to check if key exists: %w", err)
	}

	if !exists {
		// Return empty history
		return map[string]any{m.memoryKey: []schema.Message{}}, nil
	}

	// Get messages from Redis
	data, err := m.client.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get messages from Redis: %w", err)
	}

	// Deserialize messages
	var messages []schema.Message
	if data != "" {
		if err := json.Unmarshal([]byte(data), &messages); err != nil {
			return nil, fmt.Errorf("failed to unmarshal messages: %w", err)
		}
	}

	return map[string]any{m.memoryKey: messages}, nil
}

// SaveContext saves a new interaction to Redis.
func (m *RedisMemory) SaveContext(ctx context.Context, inputs, outputs map[string]any) error {
	inputKey := m.inputKey
	outputKey := m.outputKey

	inputVal, inputOk := inputs[inputKey]
	outputVal, outputOk := outputs[outputKey]

	if !inputOk {
		return fmt.Errorf("input key %s not found in inputs", inputKey)
	}
	if !outputOk {
		return fmt.Errorf("output key %s not found in outputs", outputKey)
	}

	// Convert inputs/outputs to messages
	var newMessages []schema.Message

	// Add human message
	if inputStr, ok := inputVal.(string); ok {
		newMessages = append(newMessages, schema.NewHumanMessage(inputStr))
	} else if inputMsg, ok := inputVal.(schema.Message); ok {
		newMessages = append(newMessages, inputMsg)
	}

	// Add AI message
	if outputStr, ok := outputVal.(string); ok {
		newMessages = append(newMessages, schema.NewAIMessage(outputStr))
	} else if outputMsg, ok := outputVal.(schema.Message); ok {
		newMessages = append(newMessages, outputMsg)
	}

	// Get existing messages
	existing, err := m.LoadMemoryVariables(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to load existing messages: %w", err)
	}

	existingMessages, ok := existing[m.memoryKey].([]schema.Message)
	if !ok {
		existingMessages = []schema.Message{}
	}

	// Append new messages
	allMessages := append(existingMessages, newMessages...)

	// Serialize and save to Redis
	data, err := json.Marshal(allMessages)
	if err != nil {
		return fmt.Errorf("failed to marshal messages: %w", err)
	}

	key := m.getRedisKey(m.memoryKey)
	if err := m.client.Set(ctx, key, string(data), m.ttl); err != nil {
		return fmt.Errorf("failed to save messages to Redis: %w", err)
	}

	return nil
}

// Clear clears the memory contents from Redis.
func (m *RedisMemory) Clear(ctx context.Context) error {
	key := m.getRedisKey(m.memoryKey)
	if err := m.client.Del(ctx, key); err != nil {
		return fmt.Errorf("failed to clear memory from Redis: %w", err)
	}
	return nil
}

// getRedisKey generates a Redis key for the given memory key.
func (m *RedisMemory) getRedisKey(key string) string {
	return fmt.Sprintf("beluga:memory:%s:%s", m.sessionID, key)
}

// RedisChatMessageHistory implements ChatMessageHistory using Redis.
type RedisChatMessageHistory struct {
	client    RedisClient
	sessionID string
	ttl       time.Duration
}

// NewRedisChatMessageHistory creates a new Redis-backed chat message history.
func NewRedisChatMessageHistory(client RedisClient, sessionID string, ttl time.Duration) *RedisChatMessageHistory {
	if ttl == 0 {
		ttl = 24 * time.Hour
	}

	return &RedisChatMessageHistory{
		client:    client,
		sessionID: sessionID,
		ttl:       ttl,
	}
}

// AddMessage adds a message to the history.
func (h *RedisChatMessageHistory) AddMessage(ctx context.Context, message schema.Message) error {
	key := h.getRedisKey()

	// Get existing messages
	exists, err := h.client.Exists(ctx, key)
	if err != nil {
		return fmt.Errorf("failed to check if key exists: %w", err)
	}

	var messages []schema.Message
	if exists {
		data, err := h.client.Get(ctx, key)
		if err != nil {
			return fmt.Errorf("failed to get messages: %w", err)
		}

		if data != "" {
			if err := json.Unmarshal([]byte(data), &messages); err != nil {
				return fmt.Errorf("failed to unmarshal messages: %w", err)
			}
		}
	}

	// Append new message
	messages = append(messages, message)

	// Serialize and save
	data, err := json.Marshal(messages)
	if err != nil {
		return fmt.Errorf("failed to marshal messages: %w", err)
	}

	if err := h.client.Set(ctx, key, string(data), h.ttl); err != nil {
		return fmt.Errorf("failed to save messages: %w", err)
	}

	return nil
}

// AddUserMessage adds a human message to the history.
func (h *RedisChatMessageHistory) AddUserMessage(ctx context.Context, content string) error {
	return h.AddMessage(ctx, schema.NewHumanMessage(content))
}

// AddAIMessage adds an AI message to the history.
func (h *RedisChatMessageHistory) AddAIMessage(ctx context.Context, content string) error {
	return h.AddMessage(ctx, schema.NewAIMessage(content))
}

// GetMessages returns all messages in the history.
func (h *RedisChatMessageHistory) GetMessages(ctx context.Context) ([]schema.Message, error) {
	key := h.getRedisKey()

	exists, err := h.client.Exists(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to check if key exists: %w", err)
	}

	if !exists {
		return []schema.Message{}, nil
	}

	data, err := h.client.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get messages: %w", err)
	}

	if data == "" {
		return []schema.Message{}, nil
	}

	var messages []schema.Message
	if err := json.Unmarshal([]byte(data), &messages); err != nil {
		return nil, fmt.Errorf("failed to unmarshal messages: %w", err)
	}

	return messages, nil
}

// Clear removes all messages from the history.
func (h *RedisChatMessageHistory) Clear(ctx context.Context) error {
	key := h.getRedisKey()
	if err := h.client.Del(ctx, key); err != nil {
		return fmt.Errorf("failed to clear messages: %w", err)
	}
	return nil
}

// getRedisKey generates a Redis key for the message history.
func (h *RedisChatMessageHistory) getRedisKey() string {
	return "beluga:chat_history:" + h.sessionID
}

// Ensure RedisMemory implements the Memory interface.
var _ iface.Memory = (*RedisMemory)(nil)

// Ensure RedisChatMessageHistory implements the ChatMessageHistory interface.
var _ iface.ChatMessageHistory = (*RedisChatMessageHistory)(nil)
