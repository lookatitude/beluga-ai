package mockworkflow

import (
	"context"
	"fmt"
	"sync"

	"github.com/lookatitude/beluga-ai/workflow"
)

// Compile-time interface check
var _ workflow.WorkflowStore = (*MockWorkflowStore)(nil)

// MockWorkflowStore is a configurable mock for the WorkflowStore interface.
// It stores workflow states in-memory and tracks all method calls.
type MockWorkflowStore struct {
	mu sync.Mutex

	states   map[string]workflow.WorkflowState
	err      error
	saveFn   func(ctx context.Context, state workflow.WorkflowState) error
	loadFn   func(ctx context.Context, workflowID string) (*workflow.WorkflowState, error)
	listFn   func(ctx context.Context, filter workflow.WorkflowFilter) ([]workflow.WorkflowState, error)
	deleteFn func(ctx context.Context, workflowID string) error

	saveCalls   int
	loadCalls   int
	listCalls   int
	deleteCalls int
	lastState   *workflow.WorkflowState
	lastFilter  *workflow.WorkflowFilter
}

// Option configures a MockWorkflowStore.
type Option func(*MockWorkflowStore)

// New creates a MockWorkflowStore with the given options.
func New(opts ...Option) *MockWorkflowStore {
	m := &MockWorkflowStore{
		states: make(map[string]workflow.WorkflowState),
	}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

// WithStates pre-loads the mock with the given workflow states.
func WithStates(states []workflow.WorkflowState) Option {
	return func(m *MockWorkflowStore) {
		for _, state := range states {
			m.states[state.WorkflowID] = state
		}
	}
}

// WithError configures the mock to return the given error from all methods.
func WithError(err error) Option {
	return func(m *MockWorkflowStore) {
		m.err = err
	}
}

// WithSaveFunc sets a custom function to call on Save, overriding the canned error.
func WithSaveFunc(fn func(ctx context.Context, state workflow.WorkflowState) error) Option {
	return func(m *MockWorkflowStore) {
		m.saveFn = fn
	}
}

// WithLoadFunc sets a custom function to call on Load, overriding the canned error.
func WithLoadFunc(fn func(ctx context.Context, workflowID string) (*workflow.WorkflowState, error)) Option {
	return func(m *MockWorkflowStore) {
		m.loadFn = fn
	}
}

// WithListFunc sets a custom function to call on List, overriding the canned error.
func WithListFunc(fn func(ctx context.Context, filter workflow.WorkflowFilter) ([]workflow.WorkflowState, error)) Option {
	return func(m *MockWorkflowStore) {
		m.listFn = fn
	}
}

// WithDeleteFunc sets a custom function to call on Delete, overriding the canned error.
func WithDeleteFunc(fn func(ctx context.Context, workflowID string) error) Option {
	return func(m *MockWorkflowStore) {
		m.deleteFn = fn
	}
}

// Save persists the workflow state.
func (m *MockWorkflowStore) Save(ctx context.Context, state workflow.WorkflowState) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.saveCalls++
	stateCopy := state
	m.lastState = &stateCopy

	if m.saveFn != nil {
		return m.saveFn(ctx, state)
	}

	if m.err != nil {
		return m.err
	}

	m.states[state.WorkflowID] = state
	return nil
}

// Load retrieves the workflow state by ID.
func (m *MockWorkflowStore) Load(ctx context.Context, workflowID string) (*workflow.WorkflowState, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.loadCalls++

	if m.loadFn != nil {
		return m.loadFn(ctx, workflowID)
	}

	if m.err != nil {
		return nil, m.err
	}

	state, ok := m.states[workflowID]
	if !ok {
		return nil, fmt.Errorf("workflow not found: %s", workflowID)
	}

	stateCopy := state
	return &stateCopy, nil
}

// List returns workflows matching the filter.
func (m *MockWorkflowStore) List(ctx context.Context, filter workflow.WorkflowFilter) ([]workflow.WorkflowState, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.listCalls++
	filterCopy := filter
	m.lastFilter = &filterCopy

	if m.listFn != nil {
		return m.listFn(ctx, filter)
	}

	if m.err != nil {
		return nil, m.err
	}

	var results []workflow.WorkflowState
	for _, state := range m.states {
		// Apply status filter if specified
		if filter.Status != "" && state.Status != filter.Status {
			continue
		}
		results = append(results, state)
	}

	// Apply limit if specified
	if filter.Limit > 0 && len(results) > filter.Limit {
		results = results[:filter.Limit]
	}

	return results, nil
}

// Delete removes a workflow state by ID.
func (m *MockWorkflowStore) Delete(ctx context.Context, workflowID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.deleteCalls++

	if m.deleteFn != nil {
		return m.deleteFn(ctx, workflowID)
	}

	if m.err != nil {
		return m.err
	}

	delete(m.states, workflowID)
	return nil
}

// SaveCalls returns the number of times Save has been called.
func (m *MockWorkflowStore) SaveCalls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.saveCalls
}

// LoadCalls returns the number of times Load has been called.
func (m *MockWorkflowStore) LoadCalls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.loadCalls
}

// ListCalls returns the number of times List has been called.
func (m *MockWorkflowStore) ListCalls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.listCalls
}

// DeleteCalls returns the number of times Delete has been called.
func (m *MockWorkflowStore) DeleteCalls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.deleteCalls
}

// LastState returns a copy of the state passed to the most recent Save call.
func (m *MockWorkflowStore) LastState() *workflow.WorkflowState {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.lastState == nil {
		return nil
	}
	stateCopy := *m.lastState
	return &stateCopy
}

// LastFilter returns a copy of the filter passed to the most recent List call.
func (m *MockWorkflowStore) LastFilter() *workflow.WorkflowFilter {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.lastFilter == nil {
		return nil
	}
	filterCopy := *m.lastFilter
	return &filterCopy
}

// States returns a copy of all currently stored workflow states.
func (m *MockWorkflowStore) States() []workflow.WorkflowState {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]workflow.WorkflowState, 0, len(m.states))
	for _, state := range m.states {
		result = append(result, state)
	}
	return result
}

// SetError updates the error for subsequent calls.
func (m *MockWorkflowStore) SetError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.err = err
}

// Reset clears all stored states and call counters.
func (m *MockWorkflowStore) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.states = make(map[string]workflow.WorkflowState)
	m.err = nil
	m.saveFn = nil
	m.loadFn = nil
	m.listFn = nil
	m.deleteFn = nil
	m.saveCalls = 0
	m.loadCalls = 0
	m.listCalls = 0
	m.deleteCalls = 0
	m.lastState = nil
	m.lastFilter = nil
}
