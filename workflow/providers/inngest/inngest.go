package inngest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"

	"github.com/lookatitude/beluga-ai/workflow"
)

// HTTPClient defines the HTTP client interface for Inngest API calls.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// Config holds configuration for the Inngest WorkflowStore.
type Config struct {
	// BaseURL is the Inngest API base URL. Defaults to "http://localhost:8288".
	BaseURL string
	// EventKey is the Inngest event key for authentication.
	EventKey string
	// Client is an optional HTTP client. Defaults to http.DefaultClient.
	Client HTTPClient
}

// Store is an Inngest-backed WorkflowStore implementation.
// It uses the Inngest HTTP API for state management and maintains an
// in-memory index for listing operations.
type Store struct {
	baseURL  string
	eventKey string
	client   HTTPClient

	mu    sync.RWMutex
	cache map[string]workflow.WorkflowState
}

// New creates a new Inngest workflow store with the given configuration.
func New(cfg Config) (*Store, error) {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "http://localhost:8288"
	}
	client := cfg.Client
	if client == nil {
		client = http.DefaultClient
	}
	return &Store{
		baseURL:  baseURL,
		eventKey: cfg.EventKey,
		client:   client,
		cache:    make(map[string]workflow.WorkflowState),
	}, nil
}

// Save persists the workflow state by sending it to the Inngest API.
func (s *Store) Save(ctx context.Context, state workflow.WorkflowState) error {
	if state.WorkflowID == "" {
		return fmt.Errorf("inngest/save: workflow ID is required")
	}

	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("inngest/save: marshal: %w", err)
	}

	url := fmt.Sprintf("%s/v1/workflows/%s", s.baseURL, state.WorkflowID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("inngest/save: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if s.eventKey != "" {
		req.Header.Set("Authorization", "Bearer "+s.eventKey)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("inngest/save: request: %w", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	if resp.StatusCode >= 400 {
		return fmt.Errorf("inngest/save: status %d", resp.StatusCode)
	}

	s.mu.Lock()
	s.cache[state.WorkflowID] = state
	s.mu.Unlock()

	return nil
}

// Load retrieves the workflow state by ID.
func (s *Store) Load(ctx context.Context, workflowID string) (*workflow.WorkflowState, error) {
	url := fmt.Sprintf("%s/v1/workflows/%s", s.baseURL, workflowID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("inngest/load: create request: %w", err)
	}
	if s.eventKey != "" {
		req.Header.Set("Authorization", "Bearer "+s.eventKey)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("inngest/load: request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode >= 400 {
		io.Copy(io.Discard, resp.Body)
		return nil, fmt.Errorf("inngest/load: status %d", resp.StatusCode)
	}

	var state workflow.WorkflowState
	if err := json.NewDecoder(resp.Body).Decode(&state); err != nil {
		return nil, fmt.Errorf("inngest/load: decode: %w", err)
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

// Delete removes a workflow state.
func (s *Store) Delete(ctx context.Context, workflowID string) error {
	url := fmt.Sprintf("%s/v1/workflows/%s", s.baseURL, workflowID)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("inngest/delete: create request: %w", err)
	}
	if s.eventKey != "" {
		req.Header.Set("Authorization", "Bearer "+s.eventKey)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("inngest/delete: request: %w", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	// 404 is acceptable for delete.
	if resp.StatusCode >= 400 && resp.StatusCode != http.StatusNotFound {
		return fmt.Errorf("inngest/delete: status %d", resp.StatusCode)
	}

	s.mu.Lock()
	delete(s.cache, workflowID)
	s.mu.Unlock()

	return nil
}

// Compile-time check.
var _ workflow.WorkflowStore = (*Store)(nil)
