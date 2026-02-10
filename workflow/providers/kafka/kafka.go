package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/lookatitude/beluga-ai/workflow"
)

// Writer defines the interface for writing Kafka messages.
type Writer interface {
	WriteMessages(ctx context.Context, msgs ...Message) error
	Close() error
}

// Reader defines the interface for reading Kafka messages.
type Reader interface {
	ReadMessage(ctx context.Context) (Message, error)
	Close() error
}

// Message represents a simplified Kafka message.
type Message struct {
	Key   []byte
	Value []byte
}

// Config holds configuration for the Kafka WorkflowStore.
type Config struct {
	// Brokers is the list of Kafka broker addresses.
	Brokers []string
	// Topic is the Kafka topic for workflow states. Defaults to "beluga-workflows".
	Topic string
	// Writer is an optional pre-existing Kafka writer. If provided, Brokers/Topic are ignored for writes.
	Writer Writer
	// Reader is an optional pre-existing Kafka reader. If provided, Brokers/Topic are ignored for reads.
	Reader Reader
}

// Store is a Kafka-backed WorkflowStore implementation.
// It uses a compacted topic with workflow ID as the key, and maintains
// an in-memory index for fast lookups.
type Store struct {
	writer Writer
	reader Reader
	topic  string

	mu    sync.RWMutex
	cache map[string]workflow.WorkflowState
}

// New creates a new Kafka workflow store with the given configuration.
func New(cfg Config) (*Store, error) {
	topic := cfg.Topic
	if topic == "" {
		topic = "beluga-workflows"
	}

	if cfg.Writer == nil {
		return nil, fmt.Errorf("kafka: writer is required")
	}

	return &Store{
		writer: cfg.Writer,
		reader: cfg.Reader,
		topic:  topic,
		cache:  make(map[string]workflow.WorkflowState),
	}, nil
}

// NewWithWriterReader creates a Kafka workflow store with a writer and optional reader.
// This is useful for testing with mock implementations.
func NewWithWriterReader(writer Writer, reader Reader) *Store {
	return &Store{
		writer: writer,
		topic:  "beluga-workflows",
		reader: reader,
		cache:  make(map[string]workflow.WorkflowState),
	}
}

// Close closes the underlying Kafka writer and reader.
func (s *Store) Close() error {
	var errs []error
	if s.writer != nil {
		if err := s.writer.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if s.reader != nil {
		if err := s.reader.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}

// Save persists the workflow state by writing it to the Kafka topic.
func (s *Store) Save(ctx context.Context, state workflow.WorkflowState) error {
	if state.WorkflowID == "" {
		return fmt.Errorf("kafka/save: workflow ID is required")
	}

	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("kafka/save: marshal: %w", err)
	}

	if err := s.writer.WriteMessages(ctx, Message{
		Key:   []byte(state.WorkflowID),
		Value: data,
	}); err != nil {
		return fmt.Errorf("kafka/save: write: %w", err)
	}

	s.mu.Lock()
	s.cache[state.WorkflowID] = state
	s.mu.Unlock()

	return nil
}

// Load retrieves the workflow state by ID from the in-memory cache.
func (s *Store) Load(_ context.Context, workflowID string) (*workflow.WorkflowState, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	state, ok := s.cache[workflowID]
	if !ok {
		return nil, nil
	}
	return &state, nil
}

// List returns workflows matching the filter from the in-memory cache.
func (s *Store) List(_ context.Context, filter workflow.WorkflowFilter) ([]workflow.WorkflowState, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []workflow.WorkflowState
	for _, state := range s.cache {
		if filter.Status != "" && state.Status != filter.Status {
			continue
		}
		results = append(results, state)
		if filter.Limit > 0 && len(results) >= filter.Limit {
			break
		}
	}
	return results, nil
}

// Delete removes a workflow state by writing a tombstone (nil value) to the topic.
func (s *Store) Delete(ctx context.Context, workflowID string) error {
	// Write a tombstone message (nil value) for compaction.
	if err := s.writer.WriteMessages(ctx, Message{
		Key:   []byte(workflowID),
		Value: nil,
	}); err != nil {
		return fmt.Errorf("kafka/delete: write tombstone: %w", err)
	}

	s.mu.Lock()
	delete(s.cache, workflowID)
	s.mu.Unlock()

	return nil
}

// Compile-time check.
var _ workflow.WorkflowStore = (*Store)(nil)
