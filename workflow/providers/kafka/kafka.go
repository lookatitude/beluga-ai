package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"sync"

	"github.com/lookatitude/beluga-ai/v2/core"
	"github.com/lookatitude/beluga-ai/v2/workflow"
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
	//
	// Deprecated: this package does not dial Kafka itself and never reads
	// Brokers. Configure the broker addresses on the Writer and Reader you
	// pass in.
	Brokers []string
	// Topic is the Kafka topic for workflow states. Defaults to
	// "beluga-workflows". It is informational only: the topic is chosen by
	// the Writer and Reader you pass in.
	Topic string
	// Writer is the Kafka writer used to persist states. Required.
	Writer Writer
	// Reader is an optional Kafka reader positioned at the start of the
	// workflow topic. It is only used by [Store.Restore] to rebuild the
	// in-memory index after a restart.
	Reader Reader
}

// Store is a Kafka-backed WorkflowStore implementation.
// It writes every state to a compacted topic with the workflow ID as the key,
// and serves Load and List from an in-memory index of the latest state per
// workflow.
//
// The index starts empty. Load and List only see states written by this
// process unless [Store.Restore] has been called to replay the topic, so call
// Restore at startup when states must survive a restart.
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
		return nil, core.Errorf(core.ErrInvalidInput, "kafka: writer is required")
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
	return errors.Join(errs...)
}

// Restore rebuilds the in-memory index by reading the topic through the
// configured Reader until the reader reports io.EOF or ctx is done (use a
// ctx with a deadline when the Reader blocks at the end of the topic).
// Later messages for a workflow replace earlier ones, and tombstones (empty
// values) remove the workflow. It returns the number of messages applied.
// Reaching the end of the topic, or ctx's deadline, is not an error.
func (s *Store) Restore(ctx context.Context) (int, error) {
	if s.reader == nil {
		return 0, core.Errorf(core.ErrInvalidInput, "kafka/restore: no reader configured")
	}
	applied := 0
	for {
		msg, err := s.reader.ReadMessage(ctx)
		if err != nil {
			if errors.Is(err, io.EOF) || ctx.Err() != nil {
				return applied, nil
			}
			return applied, core.Wrapf(core.ErrProviderDown, "kafka/restore: read: %w", err)
		}
		if err := s.apply(msg); err != nil {
			return applied, err
		}
		applied++
	}
}

// apply updates the index from one topic message.
func (s *Store) apply(msg Message) error {
	id := string(msg.Key)
	if id == "" {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(msg.Value) == 0 {
		delete(s.cache, id)
		return nil
	}
	var state workflow.WorkflowState
	if err := json.Unmarshal(msg.Value, &state); err != nil {
		return core.Errorf(core.ErrInvalidInput, "kafka/restore: decode %q: %w", id, err)
	}
	s.cache[id] = state
	return nil
}

// Save persists the workflow state by writing it to the Kafka topic.
func (s *Store) Save(ctx context.Context, state workflow.WorkflowState) error {
	if state.WorkflowID == "" {
		return core.Errorf(core.ErrInvalidInput, "kafka/save: workflow ID is required")
	}

	data, err := json.Marshal(state)
	if err != nil {
		return core.Errorf(core.ErrInvalidInput, "kafka/save: marshal: %w", err)
	}

	if err := s.writer.WriteMessages(ctx, Message{
		Key:   []byte(state.WorkflowID),
		Value: data,
	}); err != nil {
		return core.Wrapf(core.ErrProviderDown, "kafka/save: write: %w", err)
	}

	s.mu.Lock()
	s.cache[state.WorkflowID] = state
	s.mu.Unlock()

	return nil
}

// Load retrieves the workflow state by ID from the in-memory index (see
// [Store.Restore]).
func (s *Store) Load(_ context.Context, workflowID string) (*workflow.WorkflowState, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	state, ok := s.cache[workflowID]
	if !ok {
		return nil, nil
	}
	return &state, nil
}

// List returns workflows matching the filter from the in-memory index (see
// [Store.Restore]).
func (s *Store) List(_ context.Context, filter workflow.WorkflowFilter) ([]workflow.WorkflowState, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []workflow.WorkflowState
	for id := range s.cache {
		if filter.Status != "" && s.cache[id].Status != filter.Status {
			continue
		}
		results = append(results, s.cache[id])
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
		return core.Wrapf(core.ErrProviderDown, "kafka/delete: write tombstone: %w", err)
	}

	s.mu.Lock()
	delete(s.cache, workflowID)
	s.mu.Unlock()

	return nil
}

// Compile-time check.
var _ workflow.WorkflowStore = (*Store)(nil)
