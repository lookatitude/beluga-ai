package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/lookatitude/beluga-ai/workflow"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// Config holds configuration for the NATS workflow store.
type Config struct {
	// URL is the NATS server URL (e.g., "nats://localhost:4222").
	URL string
	// Bucket is the KV store bucket name for workflow states.
	Bucket string
	// Conn is an optional pre-existing NATS connection. If provided, URL is ignored.
	Conn *nats.Conn
}

// kvStore abstracts NATS KV operations for testability.
type kvStore interface {
	get(ctx context.Context, key string) ([]byte, error)
	put(ctx context.Context, key string, value []byte) error
	delete(ctx context.Context, key string) error
	keys(ctx context.Context) ([]string, error)
}

// natsKV wraps a real NATS JetStream KeyValue.
type natsKV struct {
	kv jetstream.KeyValue
}

func (n *natsKV) get(ctx context.Context, key string) ([]byte, error) {
	entry, err := n.kv.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	return entry.Value(), nil
}

func (n *natsKV) put(ctx context.Context, key string, value []byte) error {
	_, err := n.kv.Put(ctx, key, value)
	return err
}

func (n *natsKV) delete(ctx context.Context, key string) error {
	return n.kv.Delete(ctx, key)
}

func (n *natsKV) keys(ctx context.Context) ([]string, error) {
	return n.kv.Keys(ctx)
}

// Store is a NATS JetStream KV-backed WorkflowStore implementation.
type Store struct {
	conn   *nats.Conn
	kv     kvStore
	owns   bool // whether we own the connection and should close it
	bucket string
}

// New creates a new NATS workflow store with the given configuration.
func New(cfg Config) (*Store, error) {
	bucket := cfg.Bucket
	if bucket == "" {
		bucket = "beluga_workflows"
	}

	var conn *nats.Conn
	var owns bool
	if cfg.Conn != nil {
		conn = cfg.Conn
		owns = false
	} else {
		url := cfg.URL
		if url == "" {
			url = nats.DefaultURL
		}
		var err error
		conn, err = nats.Connect(url)
		if err != nil {
			return nil, fmt.Errorf("nats: connect: %w", err)
		}
		owns = true
	}

	js, err := jetstream.New(conn)
	if err != nil {
		if owns {
			conn.Close()
		}
		return nil, fmt.Errorf("nats: jetstream: %w", err)
	}

	kv, err := js.CreateOrUpdateKeyValue(context.Background(), jetstream.KeyValueConfig{
		Bucket: bucket,
	})
	if err != nil {
		if owns {
			conn.Close()
		}
		return nil, fmt.Errorf("nats: create kv bucket: %w", err)
	}

	return &Store{
		conn:   conn,
		kv:     &natsKV{kv: kv},
		owns:   owns,
		bucket: bucket,
	}, nil
}

// newWithKV creates a NATS workflow store with a custom KV implementation (for testing).
func newWithKV(kv kvStore) *Store {
	return &Store{
		kv:   kv,
		owns: false,
	}
}

// Close closes the underlying NATS connection if owned by this store.
func (s *Store) Close() {
	if s.owns && s.conn != nil {
		s.conn.Close()
	}
}

// Save persists the workflow state to the NATS KV store.
func (s *Store) Save(ctx context.Context, state workflow.WorkflowState) error {
	if state.WorkflowID == "" {
		return fmt.Errorf("nats/save: workflow ID is required")
	}

	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("nats/save: marshal: %w", err)
	}

	if err := s.kv.put(ctx, state.WorkflowID, data); err != nil {
		return fmt.Errorf("nats/save: put: %w", err)
	}
	return nil
}

// Load retrieves the workflow state by ID from the NATS KV store.
func (s *Store) Load(ctx context.Context, workflowID string) (*workflow.WorkflowState, error) {
	data, err := s.kv.get(ctx, workflowID)
	if err != nil {
		if isNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("nats/load: get: %w", err)
	}

	var state workflow.WorkflowState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("nats/load: unmarshal: %w", err)
	}
	return &state, nil
}

// List returns workflows matching the filter by scanning all keys.
func (s *Store) List(ctx context.Context, filter workflow.WorkflowFilter) ([]workflow.WorkflowState, error) {
	keys, err := s.kv.keys(ctx)
	if err != nil {
		if isNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("nats/list: keys: %w", err)
	}

	var results []workflow.WorkflowState
	for _, key := range keys {
		data, err := s.kv.get(ctx, key)
		if err != nil {
			continue
		}

		var state workflow.WorkflowState
		if err := json.Unmarshal(data, &state); err != nil {
			continue
		}

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

// Delete removes a workflow state by ID from the NATS KV store.
func (s *Store) Delete(ctx context.Context, workflowID string) error {
	if err := s.kv.delete(ctx, workflowID); err != nil {
		if isNotFound(err) {
			return nil
		}
		return fmt.Errorf("nats/delete: %w", err)
	}
	return nil
}

// isNotFound checks if the error indicates a key was not found.
func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	errMsg := err.Error()
	return strings.Contains(errMsg, "not found") ||
		strings.Contains(errMsg, "no keys found") ||
		err == jetstream.ErrKeyNotFound
}

// Compile-time check.
var _ workflow.WorkflowStore = (*Store)(nil)
