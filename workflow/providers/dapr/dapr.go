// Package dapr provides a Dapr state store-backed WorkflowStore implementation
// for the Beluga AI workflow engine. It uses Dapr's state management API for
// persisting workflow state.
//
// Usage:
//
//	import _ "github.com/lookatitude/beluga-ai/workflow/providers/dapr"
//
//	store, err := dapr.New(dapr.Config{
//	    Client:    daprClient,
//	    StoreName: "statestore",
//	})
package dapr

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/lookatitude/beluga-ai/workflow"
)

// StateClient defines the subset of the Dapr client interface used for state operations.
type StateClient interface {
	SaveState(ctx context.Context, storeName, key string, data []byte, meta map[string]string, so ...any) error
	GetState(ctx context.Context, storeName, key string, meta map[string]string) (*StateItem, error)
	DeleteState(ctx context.Context, storeName, key string, meta map[string]string) error
}

// StateItem represents a Dapr state item returned by GetState.
type StateItem struct {
	Key   string
	Value []byte
	Etag  string
}

// Config holds configuration for the Dapr WorkflowStore.
type Config struct {
	// Client is the Dapr state client. Required.
	Client StateClient
	// StoreName is the Dapr state store component name. Defaults to "statestore".
	StoreName string
}

// Store is a Dapr state store-backed WorkflowStore implementation.
// It maintains an in-memory index of workflow IDs for listing.
type Store struct {
	client    StateClient
	storeName string

	mu  sync.RWMutex
	ids map[string]bool
}

// New creates a new Dapr workflow store with the given configuration.
func New(cfg Config) (*Store, error) {
	if cfg.Client == nil {
		return nil, fmt.Errorf("dapr: client is required")
	}
	storeName := cfg.StoreName
	if storeName == "" {
		storeName = "statestore"
	}
	return &Store{
		client:    cfg.Client,
		storeName: storeName,
		ids:       make(map[string]bool),
	}, nil
}

// NewWithClient creates a Dapr workflow store with the given client.
// This is useful for testing with mock implementations.
func NewWithClient(client StateClient, storeName string) *Store {
	if storeName == "" {
		storeName = "statestore"
	}
	return &Store{
		client:    client,
		storeName: storeName,
		ids:       make(map[string]bool),
	}
}

// Save persists the workflow state using Dapr state store.
func (s *Store) Save(ctx context.Context, state workflow.WorkflowState) error {
	if state.WorkflowID == "" {
		return fmt.Errorf("dapr/save: workflow ID is required")
	}

	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("dapr/save: marshal: %w", err)
	}

	if err := s.client.SaveState(ctx, s.storeName, state.WorkflowID, data, nil); err != nil {
		return fmt.Errorf("dapr/save: %w", err)
	}

	s.mu.Lock()
	s.ids[state.WorkflowID] = true
	s.mu.Unlock()

	return nil
}

// Load retrieves the workflow state by ID from Dapr state store.
func (s *Store) Load(ctx context.Context, workflowID string) (*workflow.WorkflowState, error) {
	item, err := s.client.GetState(ctx, s.storeName, workflowID, nil)
	if err != nil {
		return nil, fmt.Errorf("dapr/load: %w", err)
	}
	if item == nil || len(item.Value) == 0 {
		return nil, nil
	}

	var state workflow.WorkflowState
	if err := json.Unmarshal(item.Value, &state); err != nil {
		return nil, fmt.Errorf("dapr/load: unmarshal: %w", err)
	}
	return &state, nil
}

// List returns workflows matching the filter by loading from the state store.
func (s *Store) List(ctx context.Context, filter workflow.WorkflowFilter) ([]workflow.WorkflowState, error) {
	s.mu.RLock()
	ids := make([]string, 0, len(s.ids))
	for id := range s.ids {
		ids = append(ids, id)
	}
	s.mu.RUnlock()

	var results []workflow.WorkflowState
	for _, id := range ids {
		state, err := s.Load(ctx, id)
		if err != nil {
			continue
		}
		if state == nil {
			continue
		}
		if filter.Status != "" && state.Status != filter.Status {
			continue
		}
		results = append(results, *state)
		if filter.Limit > 0 && len(results) >= filter.Limit {
			break
		}
	}
	return results, nil
}

// Delete removes a workflow state from the Dapr state store.
func (s *Store) Delete(ctx context.Context, workflowID string) error {
	if err := s.client.DeleteState(ctx, s.storeName, workflowID, nil); err != nil {
		return fmt.Errorf("dapr/delete: %w", err)
	}

	s.mu.Lock()
	delete(s.ids, workflowID)
	s.mu.Unlock()

	return nil
}

// Compile-time check.
var _ workflow.WorkflowStore = (*Store)(nil)
