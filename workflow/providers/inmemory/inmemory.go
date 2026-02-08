// Package inmemory provides an in-memory WorkflowStore for development and testing.
// It does not provide durable persistence across process restarts.
package inmemory

import (
	"context"
	"fmt"
	"sync"

	"github.com/lookatitude/beluga-ai/workflow"
)

// Store is an in-memory WorkflowStore implementation.
type Store struct {
	workflows map[string]workflow.WorkflowState
	mu        sync.RWMutex
}

// New creates a new in-memory workflow store.
func New() *Store {
	return &Store{
		workflows: make(map[string]workflow.WorkflowState),
	}
}

// Save persists the workflow state in memory.
func (s *Store) Save(_ context.Context, state workflow.WorkflowState) error {
	if state.WorkflowID == "" {
		return fmt.Errorf("inmemory/save: workflow ID is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.workflows[state.WorkflowID] = state
	return nil
}

// Load retrieves the workflow state by ID.
func (s *Store) Load(_ context.Context, workflowID string) (*workflow.WorkflowState, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	state, ok := s.workflows[workflowID]
	if !ok {
		return nil, nil
	}
	return &state, nil
}

// List returns workflows matching the filter.
func (s *Store) List(_ context.Context, filter workflow.WorkflowFilter) ([]workflow.WorkflowState, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []workflow.WorkflowState
	for _, state := range s.workflows {
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

// Delete removes a workflow state by ID.
func (s *Store) Delete(_ context.Context, workflowID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.workflows, workflowID)
	return nil
}

// Compile-time check.
var _ workflow.WorkflowStore = (*Store)(nil)
