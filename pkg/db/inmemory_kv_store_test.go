package db

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewInMemoryKeyValueStore(t *testing.T) {
	store := NewInMemoryKeyValueStore()
	require.NotNil(t, store, "NewInMemoryKeyValueStore should not return nil")
	assert.NotNil(t, store.data, "Store data map should be initialized")
}

func TestInMemoryKeyValueStore_SetAndGet(t *testing.T) {
	store := NewInMemoryKeyValueStore()
	ctx := context.Background()

	key := "testKey"
	value := "testValue"

	// Test Set
	err := store.Set(ctx, key, value)
	assert.NoError(t, err, "Set should not return an error for valid input")

	// Test Get
	retrievedValue, ok, err := store.Get(ctx, key)
	assert.NoError(t, err, "Get should not return an error for an existing key")
	assert.True(t, ok, "Get should return true for an existing key")
	assert.Equal(t, value, retrievedValue, "Retrieved value should match the set value")

	// Test Get non-existent key
	_, ok, err = store.Get(ctx, "nonExistentKey")
	assert.NoError(t, err, "Get should not return an error for a non-existent key")
	assert.False(t, ok, "Get should return false for a non-existent key")

	// Test Set with empty key
	err = store.Set(ctx, "", "emptyKeyValue")
	assert.Error(t, err, "Set should return an error for an empty key")
	assert.EqualError(t, err, "key cannot be empty", "Error message for empty key mismatch")
}

func TestInMemoryKeyValueStore_Delete(t *testing.T) {
	store := NewInMemoryKeyValueStore()
	ctx := context.Background()

	key := "deleteKey"
	value := "deleteValue"

	// Set a value first
	err := store.Set(ctx, key, value)
	require.NoError(t, err)

	// Delete the value
	err = store.Delete(ctx, key)
	assert.NoError(t, err, "Delete should not return an error for an existing key")

	// Try to Get the deleted value
	_, ok, err := store.Get(ctx, key)
	assert.NoError(t, err)
	assert.False(t, ok, "Get should return false after Delete")

	// Delete non-existent key
	err = store.Delete(ctx, "nonExistentKey")
	assert.NoError(t, err, "Delete should not return an error for a non-existent key")
}

func TestInMemoryKeyValueStore_Exists(t *testing.T) {
	store := NewInMemoryKeyValueStore()
	ctx := context.Background()

	key := "existsKey"
	value := "existsValue"

	// Check non-existent key
	ok, err := store.Exists(ctx, key)
	assert.NoError(t, err, "Exists should not return an error")
	assert.False(t, ok, "Exists should return false for a non-existent key")

	// Set a value
	err = store.Set(ctx, key, value)
	require.NoError(t, err)

	// Check existing key
	ok, err = store.Exists(ctx, key)
	assert.NoError(t, err, "Exists should not return an error")
	assert.True(t, ok, "Exists should return true for an existing key")

	// Delete the key and check again
	err = store.Delete(ctx, key)
	require.NoError(t, err)
	ok, err = store.Exists(ctx, key)
	assert.NoError(t, err, "Exists should not return an error")
	assert.False(t, ok, "Exists should return false after deleting the key")
}

func TestInMemoryKeyValueStore_Clear(t *testing.T) {
	store := NewInMemoryKeyValueStore()
	ctx := context.Background()

	// Set some values
	err := store.Set(ctx, "key1", "value1")
	require.NoError(t, err)
	err = store.Set(ctx, "key2", "value2")
	require.NoError(t, err)

	// Clear the store
	err = store.Clear(ctx)
	assert.NoError(t, err, "Clear should not return an error")

	// Check if keys exist
	ok, err := store.Exists(ctx, "key1")
	assert.NoError(t, err)
	assert.False(t, ok, "Key1 should not exist after Clear")

	ok, err = store.Exists(ctx, "key2")
	assert.NoError(t, err)
	assert.False(t, ok, "Key2 should not exist after Clear")

	// Check if data map is empty (or re-initialized)
	store.mu.RLock()
	assert.Empty(t, store.data, "Store data map should be empty after Clear")
	store.mu.RUnlock()
}

func TestInMemoryKeyValueStore_ConcurrentAccess(t *testing.T) {
	store := NewInMemoryKeyValueStore()
	ctx := context.Background()
	numGoroutines := 100
	done := make(chan bool)

	// Concurrent Set operations
	for i := 0; i < numGoroutines; i++ {
		go func(i int) {
			key := fmt.Sprintf("concurrentKey%d", i)
			value := fmt.Sprintf("concurrentValue%d", i)
			err := store.Set(ctx, key, value)
			assert.NoError(t, err)
			done <- true
		}(i)
	}

	// Wait for Set operations to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Concurrent Get operations
	for i := 0; i < numGoroutines; i++ {
		go func(i int) {
			key := fmt.Sprintf("concurrentKey%d", i)
			expectedValue := fmt.Sprintf("concurrentValue%d", i)
			val, ok, err := store.Get(ctx, key)
			assert.NoError(t, err)
			assert.True(t, ok)
			assert.Equal(t, expectedValue, val)
			done <- true
		}(i)
	}

	// Wait for Get operations to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Check total items
	store.mu.RLock()
	assert.Len(t, store.data, numGoroutines, "Store should contain all concurrently set items")
	store.mu.RUnlock()
}

