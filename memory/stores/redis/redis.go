// Package redis provides a Redis-backed implementation of memory.MessageStore.
// Messages are stored as JSON in a Redis sorted set, scored by insertion
// timestamp, providing natural chronological ordering. Search uses
// case-insensitive substring matching on text content parts.
//
// This implementation requires a Redis server (v5.0+) and uses
// github.com/redis/go-redis/v9 as the client library.
package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/lookatitude/beluga-ai/memory"
	"github.com/lookatitude/beluga-ai/schema"
	"github.com/redis/go-redis/v9"
)

// Config holds configuration for the Redis MessageStore.
type Config struct {
	// Client is the Redis client to use. Required.
	Client *redis.Client
	// Key is the Redis key for the sorted set. Defaults to "beluga:messages".
	Key string
}

// MessageStore is a Redis-backed implementation of memory.MessageStore.
// Messages are stored as JSON in a Redis sorted set (ZSET) with the score
// set to a monotonically increasing sequence number to preserve insertion order.
type MessageStore struct {
	client *redis.Client
	key    string
	seq    atomic.Int64
}

// New creates a new Redis MessageStore with the given config.
func New(cfg Config) (*MessageStore, error) {
	if cfg.Client == nil {
		return nil, fmt.Errorf("redis: client is required")
	}
	key := cfg.Key
	if key == "" {
		key = "beluga:messages"
	}
	return &MessageStore{
		client: cfg.Client,
		key:    key,
	}, nil
}

// Append adds a message to the store.
func (s *MessageStore) Append(ctx context.Context, msg schema.Message) error {
	data, err := marshalMessage(msg)
	if err != nil {
		return fmt.Errorf("redis: marshal message: %w", err)
	}
	score := float64(s.seq.Add(1))
	return s.client.ZAdd(ctx, s.key, redis.Z{
		Score:  score,
		Member: string(data),
	}).Err()
}

// Search finds messages whose text content contains the query as a
// case-insensitive substring, returning at most k results.
func (s *MessageStore) Search(ctx context.Context, query string, k int) ([]schema.Message, error) {
	members, err := s.client.ZRangeByScore(ctx, s.key, &redis.ZRangeBy{
		Min: "-inf",
		Max: "+inf",
	}).Result()
	if err != nil {
		return nil, fmt.Errorf("redis: search: %w", err)
	}

	q := strings.ToLower(query)
	var results []schema.Message
	for _, member := range members {
		msg, err := unmarshalMessage([]byte(member))
		if err != nil {
			continue
		}
		if q == "" || containsText(msg, q) {
			results = append(results, msg)
			if len(results) >= k {
				break
			}
		}
	}
	return results, nil
}

// All returns all stored messages in chronological order.
func (s *MessageStore) All(ctx context.Context) ([]schema.Message, error) {
	members, err := s.client.ZRangeByScore(ctx, s.key, &redis.ZRangeBy{
		Min: "-inf",
		Max: "+inf",
	}).Result()
	if err != nil {
		return nil, fmt.Errorf("redis: all: %w", err)
	}

	msgs := make([]schema.Message, 0, len(members))
	for _, member := range members {
		msg, err := unmarshalMessage([]byte(member))
		if err != nil {
			continue
		}
		msgs = append(msgs, msg)
	}
	return msgs, nil
}

// Clear removes all messages from the store.
func (s *MessageStore) Clear(ctx context.Context) error {
	return s.client.Del(ctx, s.key).Err()
}

// containsText checks if any text part in the message contains the query.
func containsText(msg schema.Message, lowerQuery string) bool {
	for _, p := range msg.GetContent() {
		if tp, ok := p.(schema.TextPart); ok {
			if strings.Contains(strings.ToLower(tp.Text), lowerQuery) {
				return true
			}
		}
	}
	return false
}

// storedMessage is the JSON representation of a schema.Message.
type storedMessage struct {
	Role       string            `json:"role"`
	Parts      []storedPart      `json:"parts"`
	Metadata   map[string]any    `json:"metadata,omitempty"`
	ToolCalls  []schema.ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string            `json:"tool_call_id,omitempty"`
	ModelID    string            `json:"model_id,omitempty"`
	Timestamp  time.Time         `json:"timestamp"`
}

// storedPart is the JSON representation of a schema.ContentPart.
type storedPart struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

func marshalMessage(msg schema.Message) ([]byte, error) {
	sm := storedMessage{
		Role:     string(msg.GetRole()),
		Metadata: msg.GetMetadata(),
	}
	for _, p := range msg.GetContent() {
		sp := storedPart{Type: string(p.PartType())}
		if tp, ok := p.(schema.TextPart); ok {
			sp.Text = tp.Text
		}
		sm.Parts = append(sm.Parts, sp)
	}
	if ai, ok := msg.(*schema.AIMessage); ok {
		sm.ToolCalls = ai.ToolCalls
		sm.ModelID = ai.ModelID
	}
	if tm, ok := msg.(*schema.ToolMessage); ok {
		sm.ToolCallID = tm.ToolCallID
	}
	sm.Timestamp = time.Now()
	return json.Marshal(sm)
}

func unmarshalMessage(data []byte) (schema.Message, error) {
	var sm storedMessage
	if err := json.Unmarshal(data, &sm); err != nil {
		return nil, err
	}

	parts := make([]schema.ContentPart, 0, len(sm.Parts))
	for _, sp := range sm.Parts {
		switch schema.ContentType(sp.Type) {
		case schema.ContentText:
			parts = append(parts, schema.TextPart{Text: sp.Text})
		default:
			parts = append(parts, schema.TextPart{Text: sp.Text})
		}
	}

	switch schema.Role(sm.Role) {
	case schema.RoleSystem:
		return &schema.SystemMessage{Parts: parts, Metadata: sm.Metadata}, nil
	case schema.RoleHuman:
		return &schema.HumanMessage{Parts: parts, Metadata: sm.Metadata}, nil
	case schema.RoleAI:
		return &schema.AIMessage{Parts: parts, ToolCalls: sm.ToolCalls, ModelID: sm.ModelID, Metadata: sm.Metadata}, nil
	case schema.RoleTool:
		return &schema.ToolMessage{ToolCallID: sm.ToolCallID, Parts: parts, Metadata: sm.Metadata}, nil
	default:
		return &schema.HumanMessage{Parts: parts, Metadata: sm.Metadata}, nil
	}
}

// Verify interface compliance.
var _ memory.MessageStore = (*MessageStore)(nil)
