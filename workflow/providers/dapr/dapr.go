package dapr

import (
	"context"
	"encoding/json"
	"sort"
	"sync"

	"github.com/lookatitude/beluga-ai/v2/core"
	"github.com/lookatitude/beluga-ai/v2/workflow"
)

// IndexKey is the state-store key under which the Store keeps the list of
// workflow IDs it has saved, so List works across processes and restarts.
// Workflow IDs equal to IndexKey are rejected.
const IndexKey = "beluga-workflow-index"

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
//
// Each workflow is stored under its ID. The set of IDs is also persisted in
// the state store under [IndexKey], so List sees workflows saved by other
// processes and before a restart. The index is updated with a plain
// read-modify-write: when several processes save new workflows at the same
// moment an ID can be missing from the index (Load by ID is always
// authoritative).
type Store struct {
	client    StateClient
	storeName string

	mu  sync.RWMutex
	ids map[string]bool
}

// New creates a new Dapr workflow store with the given configuration.
func New(cfg Config) (*Store, error) {
	if cfg.Client == nil {
		return nil, core.Errorf(core.ErrInvalidInput, "dapr: client is required")
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
		return core.Errorf(core.ErrInvalidInput, "dapr/save: workflow ID is required")
	}
	if state.WorkflowID == IndexKey {
		return core.Errorf(core.ErrInvalidInput, "dapr/save: workflow ID %q is reserved", IndexKey)
	}

	data, err := json.Marshal(state)
	if err != nil {
		return core.Errorf(core.ErrInvalidInput, "dapr/save: marshal: %w", err)
	}

	if err := s.client.SaveState(ctx, s.storeName, state.WorkflowID, data, nil); err != nil {
		return core.Wrapf(core.ErrProviderDown, "dapr/save: %w", err)
	}

	return s.updateIndex(ctx, state.WorkflowID, true)
}

// loadIndex reads the persisted ID index. A missing index is empty.
func (s *Store) loadIndex(ctx context.Context) (map[string]bool, error) {
	item, err := s.client.GetState(ctx, s.storeName, IndexKey, nil)
	if err != nil {
		return nil, core.Wrapf(core.ErrProviderDown, "dapr/index: load: %w", err)
	}
	ids := make(map[string]bool)
	if item == nil || len(item.Value) == 0 {
		return ids, nil
	}
	var list []string
	if err := json.Unmarshal(item.Value, &list); err != nil {
		return nil, core.Errorf(core.ErrInvalidInput, "dapr/index: decode: %w", err)
	}
	for _, id := range list {
		ids[id] = true
	}
	return ids, nil
}

// updateIndex adds (add=true) or removes id in both the local and the
// persisted index.
func (s *Store) updateIndex(ctx context.Context, id string, add bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	persisted, err := s.loadIndex(ctx)
	if err != nil {
		return err
	}
	if add {
		s.ids[id] = true
		if persisted[id] {
			return nil
		}
		persisted[id] = true
	} else {
		delete(s.ids, id)
		if !persisted[id] {
			return nil
		}
		delete(persisted, id)
	}

	list := make([]string, 0, len(persisted))
	for k := range persisted {
		list = append(list, k)
	}
	sort.Strings(list)
	data, err := json.Marshal(list)
	if err != nil {
		return core.Errorf(core.ErrInvalidInput, "dapr/index: marshal: %w", err)
	}
	if err := s.client.SaveState(ctx, s.storeName, IndexKey, data, nil); err != nil {
		return core.Wrapf(core.ErrProviderDown, "dapr/index: save: %w", err)
	}
	return nil
}

// Load retrieves the workflow state by ID from Dapr state store.
func (s *Store) Load(ctx context.Context, workflowID string) (*workflow.WorkflowState, error) {
	item, err := s.client.GetState(ctx, s.storeName, workflowID, nil)
	if err != nil {
		return nil, core.Wrapf(core.ErrProviderDown, "dapr/load: %w", err)
	}
	if item == nil || len(item.Value) == 0 {
		return nil, nil
	}

	var state workflow.WorkflowState
	if err := json.Unmarshal(item.Value, &state); err != nil {
		return nil, core.Errorf(core.ErrInvalidInput, "dapr/load: unmarshal: %w", err)
	}
	return &state, nil
}

// List returns workflows matching the filter by loading from the state
// store every ID in the persisted index (plus any saved by this process).
func (s *Store) List(ctx context.Context, filter workflow.WorkflowFilter) ([]workflow.WorkflowState, error) {
	persisted, err := s.loadIndex(ctx)
	if err != nil {
		return nil, err
	}
	s.mu.RLock()
	for id := range s.ids {
		persisted[id] = true
	}
	s.mu.RUnlock()
	ids := make([]string, 0, len(persisted))
	for id := range persisted {
		ids = append(ids, id)
	}
	sort.Strings(ids)

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
		return core.Wrapf(core.ErrProviderDown, "dapr/delete: %w", err)
	}

	return s.updateIndex(ctx, workflowID, false)
}

// Compile-time check.
var _ workflow.WorkflowStore = (*Store)(nil)
