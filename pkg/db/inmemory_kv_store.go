package db

import (
	"context"
	"fmt"
	"sync"
)

// KeyValueStore defines a generic interface for a simple key-value store.
// This can be used for various purposes, such as caching, session management,
// or storing small pieces of data.
type KeyValueStore interface {
	// Get retrieves a value by its key.
	// Returns the value and true if the key exists, otherwise nil and false.
	Get(ctx context.Context, key string) (interface{}, bool, error)

	// Set stores a key-value pair.
	Set(ctx context.Context, key string, value interface{}) error

	// Delete removes a key-value pair.
	Delete(ctx context.Context, key string) error

	// Exists checks if a key exists in the store.
	Exists(ctx context.Context, key string) (bool, error)

	// Clear removes all entries from the store.
	Clear(ctx context.Context) error
}

// InMemoryKeyValueStore is a simple in-memory implementation of KeyValueStore.
// It is not persistent and data will be lost when the application stops.
// It uses a sync.RWMutex for concurrent access.
type InMemoryKeyValueStore struct {
	data map[string]interface{}
	mu   sync.RWMutex
}

// NewInMemoryKeyValueStore creates a new InMemoryKeyValueStore.
func NewInMemoryKeyValueStore() *InMemoryKeyValueStore {
	return &InMemoryKeyValueStore{
		data: make(map[string]interface{}),
	}
}

// Get retrieves a value by its key.
func (s *InMemoryKeyValueStore) Get(ctx context.Context, key string) (interface{}, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	val, ok := s.data[key]
	return val, ok, nil
}

// Set stores a key-value pair.
func (s *InMemoryKeyValueStore) Set(ctx context.Context, key string, value interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if key == "" {
		return fmt.Errorf("key cannot be empty")
	}
	s.data[key] = value
	return nil
}

// Delete removes a key-value pair.
func (s *InMemoryKeyValueStore) Delete(ctx context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, key)
	return nil
}

// Exists checks if a key exists in the store.
func (s *InMemoryKeyValueStore) Exists(ctx context.Context, key string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.data[key]
	return ok, nil
}

// Clear removes all entries from the store.
func (s *InMemoryKeyValueStore) Clear(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data = make(map[string]interface{}) // Reinitialize the map
	return nil
}

// Ensure InMemoryKeyValueStore implements the KeyValueStore interface.
var _ KeyValueStore = (*InMemoryKeyValueStore)(nil)

